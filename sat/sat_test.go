package sat

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vtest"
)

func TestEchoRequest(t *testing.T) {
	sat, err := satForFile("../examples/hello-echo/hello-echo.wasm")
	if err != nil {
		t.Error(errors.Wrap(err, "failed to satForFile"))
		return
	}

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("my friend")))

	resp := vt.Do(req, t)

	resp.AssertStatus(200)
	resp.AssertBodyString("hello my friend")
}

func satForFile(filepath string) (*Sat, error) {
	config, err := ConfigFromRunnableArg(filepath)
	if err != nil {
		return nil, err
	}

	sat, err := New(config)
	if err != nil {
		return nil, err
	}

	return sat, nil
}
