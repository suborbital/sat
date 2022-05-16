package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/vektor/vlog"

	"github.com/suborbital/sat/sat"
	"github.com/suborbital/sat/sat/process"
)

var client = http.Client{Timeout: time.Second}

// watcher watches a "replicaSet" of Sats for a single FQFN
type watcher struct {
	fqfn      string
	instances map[string]*instance
	log       *vlog.Logger
}

type instance struct {
	fqfn    string
	metrics *sat.MetricsResponse
	uuid    string
	pid     int
}

type watcherReport struct {
	instCount    int
	totalThreads int
	failedPorts  []string
}

// newWatcher creates a new watcher instance for the given fqfn
func newWatcher(fqfn string, log *vlog.Logger) *watcher {
	w := &watcher{
		fqfn:      fqfn,
		instances: map[string]*instance{},
		log:       log,
	}

	return w
}

// add adds a new instance to the watched pool
func (w *watcher) add(fqfn, port, uuid string, pid int) {
	w.instances[port] = &instance{
		fqfn: fqfn,
		uuid: uuid,
		pid:  pid,
	}
}

// scaleDown terminates one random instance from the pool
func (w *watcher) scaleDown() error {
	// we use the range to get a semi-random instance
	// and then immediately return so that we only terminate one
	for p := range w.instances {
		w.log.Info("scaling down, terminating instance on port", p, "(", w.instances[p].fqfn, ")")

		return w.terminateInstance(p)
	}

	return nil
}

// terminateInstance terminates the instance from the given port
func (w *watcher) terminateInstance(p string) error {
	inst := w.instances[p]

	if inst != nil {
		if err := process.Delete(inst.uuid); err != nil {
			return errors.Wrapf(err, "failed to process.Delete for port %s ( %s )", p, inst.fqfn)
		}
	}

	delete(w.instances, p)
	w.log.Info("successfully terminated instance on port", p, "(", inst.fqfn, ")")

	return nil
}

// report fetches a metrics report from each watched instance and returns a summary
func (w *watcher) report() *watcherReport {
	if len(w.instances) == 0 {
		return nil
	}

	totalThreads := 0
	failedPorts := make([]string, 0)

	for p := range w.instances {
		metrics, err := getReport(p)
		if err != nil {
			w.log.Error(errors.Wrapf(err, "failed to getReport for %s", p))
			failedPorts = append(failedPorts, p)
		} else {
			w.instances[p].metrics = metrics
			totalThreads += metrics.Scheduler.TotalThreadCount
		}
	}

	report := &watcherReport{
		instCount:    len(w.instances) - len(failedPorts),
		totalThreads: totalThreads,
		failedPorts:  failedPorts,
	}

	return report
}

// getReport sends a request on localhost to the given port to fetch metrics
func getReport(port string) (*sat.MetricsResponse, error) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%s/meta/metrics", port), nil)

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Do metrics request")
	}

	defer resp.Body.Close()
	metricsJSON, _ := ioutil.ReadAll(resp.Body)

	metrics := &sat.MetricsResponse{}
	if err := json.Unmarshal(metricsJSON, metrics); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal metrics response")
	}

	return metrics, nil
}
