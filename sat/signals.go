package sat

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// setupSignals sets up clean shutdown from SIGINT and SIGTERM
func (s *Sat) setupSignals(shutdownChan chan error) {
	sigs := make(chan os.Signal, 64)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		s.l.Warn("encountered signal, beginning shutdown:", sig.String())

		// stop Grav with a 3s delay between Withdraw and Stop (to allow in-flight requests to drain)
		// s.v.Stop isn't called until all connections are ready to close (after said delay)
		// this is needed to ensure a safe withdraw from the constellation/mesh
		if err := s.g.Withdraw(); err != nil {
			s.l.Warn("encountered error during Withdraw, will proceed:", err.Error())
		}

		time.Sleep(time.Second * 3)

		if err := s.g.Stop(); err != nil {
			s.l.Warn("encountered error during Stop, will proceed:", err.Error())
		}

		ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
		err := s.v.StopCtx(ctx)

		s.l.Warn("handled signal, shutdown proceeding", sig.String())

		shutdownChan <- err
	}()
}
