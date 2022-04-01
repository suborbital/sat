//go:build integration
// +build integration

package tests

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	tc "github.com/testcontainers/testcontainers-go"
)

// IntegrationSuite will test @todo complete this.
type IntegrationSuite struct {
	suite.Suite

	container tc.Container
	ctxCloser context.CancelFunc
}

// TestIntegrationSuite gets run from go's test framework that kicks off the suite.
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

// SetupSuite runs first in the chain. Used to set up properties and settings
// that all test methods need access to.
func (i *IntegrationSuite) SetupSuite() {
	dir, err := os.Getwd()
	if err != nil {
		i.FailNowf("getwd", "what: %s", err.Error())
	}

	satWorkingDir := filepath.Join(dir, "../examples")
	ctx, ctxCloser := context.WithTimeout(context.Background(), 10*time.Second)
	i.ctxCloser = ctxCloser

	req := tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image: "suborbital/sat:dev",
			Env: map[string]string{
				"SAT_HTTP_PORT": "8080",
			},
			ExposedPorts: []string{"8080:8080"},
			Cmd: []string{
				"sat", "/runnables/hello-echo/hello-echo.wasm",
			},
			BindMounts: map[string]string{
				"/runnables": satWorkingDir,
			},
		},
		Started:      true,
		ProviderType: tc.ProviderDocker,
	}

	container, err := tc.GenericContainer(ctx, req)
	if err != nil {
		i.FailNowf("could not start up container", "error received: %s", err.Error())
	}

	i.container = container

	i.T().Log("starting up docker...")
	time.Sleep(2 * time.Second)
}

// TearDownSuite runs last, and is usually used to close database connections
// or clear up after running the suite.
func (i *IntegrationSuite) TearDownSuite() {
	i.ctxCloser()
	terminateCtx, termCtxCloser := context.WithTimeout(context.Background(), 3*time.Second)
	defer termCtxCloser()
	err := i.container.Terminate(terminateCtx)
	if err != nil {
		i.FailNowf("terminate failed", err.Error())
	}
	i.T().Log("hold on, terminating docker container...")
	time.Sleep(2 * time.Second)
}

// SetupTest runs before each individual test methods.
func (i *IntegrationSuite) SetupTest() {}

// TearDownTest runs after each individual test methods.
func (i *IntegrationSuite) TearDownTest() {}

// TestSatEndpoints is an example test method. Any method that starts with Test* is
// going to be run. The test methods should be independent of each other and
// their order of execution should not matter, and you should also be able to
// run an individual test method on the suite without any issues.
func (i *IntegrationSuite) TestSatEndpoints() {
	type testCase struct {
		name                string
		path                string
		requestVerb         string
		payload             []byte
		wantStatus          int
		wantResponsePayload []byte
	}

	tcs := []testCase{
		{
			name:                "sat runs hello echo successfully",
			path:                "",
			requestVerb:         http.MethodPost,
			payload:             []byte(`{"text":"from Bob Morane"}`),
			wantStatus:          http.StatusOK,
			wantResponsePayload: []byte(`hello {"text":"from Bob Morane"}`),
		},
	}

	//get := http.MethodGet
	//
	//data='{"text":"from Bob Morane"}'
	//curl -d "${data}" \
	//-H "Content-Type: application/json" \
	//-X POST "http://localhost:8080"
	//```
	client := http.Client{
		Timeout: 2 * time.Second,
	}

	baseUrl := "http://localhost:8080"

	for _, tCase := range tcs {
		i.Run(tCase.name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, tCase.requestVerb, baseUrl+"/"+tCase.path, bytes.NewReader(tCase.payload))
			i.Require().NoError(err)
			resp, err := client.Do(req)
			i.Require().NoError(err)

			responseBody, err := ioutil.ReadAll(resp.Body)
			i.Require().NoError(err)

			i.Equal(tCase.wantStatus, resp.StatusCode)
			i.Equal(tCase.wantResponsePayload, responseBody)
		})
	}
}
