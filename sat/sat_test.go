package sat

import (
	"bytes"
	"fmt"
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

func Test405Request(t *testing.T) {
	sat, err := satForFile("../examples/hello-echo/hello-echo.wasm")
	if err != nil {
		t.Error(errors.Wrap(err, "failed to satForFile"))
		return
	}

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodGet, "/", nil)

	resp := vt.Do(req, t)

	fmt.Println(string(resp.Body))

	resp.AssertStatus(405)
}

func TestErrorRequest(t *testing.T) {
	sat, err := satForFile("../examples/return-err/return-err.wasm")
	if err != nil {
		t.Error(errors.Wrap(err, "failed to satForFile"))
		return
	}

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte{}))

	resp := vt.Do(req, t)

	resp.AssertStatus(401)
	resp.AssertBodyString(`{"status":401,"message":"don't go there"}`)
}

func TestPanicRequest(t *testing.T) {
	sat, err := satForFile("../examples/panic-at-the-disco/panic-at-the-disco.wasm")
	if err != nil {
		t.Error(errors.Wrap(err, "failed to satForFile"))
		return
	}

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte{}))

	resp := vt.Do(req, t)

	resp.AssertStatus(500)
	resp.AssertBodyString(`{"status":500,"message":"unknown error"}`)
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
