package main

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

func registerWithEDP(config *config) error {
	if config.controlPlane == defaultControlPlane {
		return nil
	}

	registerURL := fmt.Sprintf("%s/upstream/register", config.controlPlane)

	req, err := http.NewRequest(http.MethodPost, registerURL, nil)
	if err != nil {
		return errors.Wrap(err, "failed to NewRequest")
	}

	if _, err := http.DefaultClient.Do(req); err != nil {
		return errors.Wrap(err, "failed to Do request")
	}

	return nil
}
