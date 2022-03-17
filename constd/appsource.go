package main

import (
	"time"

	"github.com/pkg/errors"

	// company packages.
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/atmo/options"
	aopts "github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

func startAppSourceServer(bundlePath string) (appsource.AppSource, chan error) {
	app := appsource.NewBundleSource(bundlePath)
	opts := options.NewWithModifiers()

	errChan := make(chan error)

	router, err := appsource.NewAppSourceVKRouter(app, *opts).GenerateRouter()
	if err != nil {
		errChan <- errors.Wrap(err, "failed to NewAppSourceVKRouter.GenerateRouter")
	}

	server := vk.New(
		vk.UseAppName("constd server"),
		vk.UseHTTPPort(9090),
		vk.UseEnvPrefix("CONSTD"),
	)

	server.SwapRouter(router)

	go func() {
		if err := server.Start(); err != nil {
			errChan <- errors.Wrap(err, "failed to server.Start")
		}
	}()

	return app, errChan
}

func startAppSourceWithRetry(log *vlog.Logger, source appsource.AppSource) error {
	backoffMS := float32(1000)

	var err error

	atmoOpts := aopts.NewWithModifiers()

	for i := 0; i < 10; i++ {
		if err = source.Start(*atmoOpts); err != nil {
			log.Error(errors.Wrap(err, "failed to source.Start, will retry"))

			time.Sleep(time.Millisecond * time.Duration(backoffMS))
			backoffMS *= 1.4
		} else {
			break
		}
	}

	return err
}
