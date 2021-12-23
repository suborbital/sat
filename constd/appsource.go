package main

import (
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
