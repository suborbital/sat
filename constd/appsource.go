package main

import (
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/vektor/vk"
)

func startAppSourceServer(bundlePath string) (appsource.AppSource, chan error) {
	app := appsource.NewBundleSource(bundlePath)
	opts := options.NewWithModifiers()

	errchan := make(chan error)

	router, err := appsource.NewAppSourceVKRouter(app, *opts).GenerateRouter()
	if err != nil {
		errchan <- errors.Wrap(err, "failed to NewAppSourceVKRouter.GenerateRouter")
	}

	server := vk.New(
		vk.UseAppName("constd server"),
		vk.UseHTTPPort(9090),
		vk.UseEnvPrefix("CONSTD"),
	)

	server.SwapRouter(router)

	go func() {
		if err := server.Start(); err != nil {
			errchan <- errors.Wrap(err, "failed to server.Start")
		}
	}()

	return app, errchan
}

func startAppSourceWithRetry(source appsource.AppSource) error {
	backoffMS := float32(500)

	var err error

	i := 0
	for i < 5 {
		if err = source.Start(*options.NewWithModifiers()); err == nil {
			break
		}

		time.Sleep(time.Millisecond * time.Duration(backoffMS))
		backoffMS *= 1.4
		i++
	}

	return err
}
