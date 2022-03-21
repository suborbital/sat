package sat

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/suborbital/vektor/vtest"
)

func TestEchoRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/hello-echo/hello-echo.wasm")
	if err != nil {
		t.Error(errors.Wrap(err, "failed to satForFile"))
		return
	}
	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)
	defer sat.Shutdown(ctx, syscall.SIGTERM)

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("my friend")))

	resp := vt.Do(req, t)

	resp.AssertStatus(200)
	resp.AssertBodyString("hello my friend")
}

func Test405Request(t *testing.T) {
	sat, tp, err := satForFile("../examples/hello-echo/hello-echo.wasm")
	if err != nil {
		t.Error(errors.Wrap(err, "failed to satForFile"))
		return
	}
	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)
	defer sat.Shutdown(ctx, syscall.SIGTERM)

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodGet, "/", nil)

	resp := vt.Do(req, t)

	fmt.Println(string(resp.Body))

	resp.AssertStatus(405)
}

func TestErrorRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/return-err/return-err.wasm")
	if err != nil {
		t.Error(errors.Wrap(err, "failed to satForFile"))
		return
	}
	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)
	defer sat.Shutdown(ctx, syscall.SIGTERM)

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte{}))

	resp := vt.Do(req, t)

	resp.AssertStatus(401)
	resp.AssertBodyString(`{"status":401,"message":"don't go there"}`)
}

func TestPanicRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/panic-at-the-disco/panic-at-the-disco.wasm")
	if err != nil {
		t.Error(errors.Wrap(err, "failed to satForFile"))
		return
	}
	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)
	defer sat.Shutdown(ctx, syscall.SIGTERM)

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte{}))

	resp := vt.Do(req, t)

	resp.AssertStatus(500)
	resp.AssertBodyString(`{"status":500,"message":"unknown error"}`)
}

func satForFile(filepath string) (*Sat, *trace.TracerProvider, error) {
	config, err := ConfigFromRunnableArg(filepath)
	if err != nil {
		return nil, nil, err
	}

	fmt.Printf("tracer\n\n\n%#v\n\n\n", config.TracerConfig)
	fmt.Printf("logger\n\n\n%#v\n\n\n", config.Logger)

	traceProvider, err := SetupTracing(config.TracerConfig, config.Logger)
	if err != nil {
		return nil, nil, errors.Wrap(err, "setup tracing")
	}

	sat, err := New(config, traceProvider)
	if err != nil {
		return nil, nil, err
	}

	return sat, traceProvider, nil
}
