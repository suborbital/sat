package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/bundle"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/subo/subo/util"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("missing argument: bundle")
	}

	bundlePath := os.Args[1]

	startAtmo(bundlePath)

	directive, err := unloadBundle(bundlePath)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to unloadBundle"))
	}

	startConstellation(directive)

	<-time.After(time.Hour)
}

func startAtmo(bundlePath string) {
	mountPath := filepath.Dir(bundlePath)

	go func() {
		if _, err := util.Run(fmt.Sprintf("docker run -p 8080:8080 -e ATMO_HTTP_PORT=8080 -v %s:/home/atmo --network bridge suborbital/atmo-proxy:dev atmo-proxy", mountPath)); err != nil {
			log.Fatal(errors.Wrap(err, "failed to Run Atmo"))
		}
	}()
}

func startConstellation(directive *directive.Directive) {
	satTag := "latest"
	if tag, exists := os.LookupEnv("CONSTD_SAT_TAG"); exists {
		satTag = tag
	}

	folderPath := filepath.Join(os.TempDir(), "suborbital", directiveFileHash(directive))

	for i := range directive.Runnables {
		runnable := directive.Runnables[i]

		go func() {
			fmt.Printf("launching %s\n", runnable.FQFN)

			port, err := randPort()
			if err != nil {
				log.Fatal(errors.Wrap(err, "failed to randPort"))
			}

			for {
				// build this monstrosity of an exec string
				_, err := util.Run(fmt.Sprintf(
					"docker run --rm -p %s:%s -e SAT_HTTP_PORT=%s -v %s:/runnables --network bridge --name %s suborbital/sat:%s sat %s.wasm",
					port, port, port,
					folderPath,
					runnable.Name,
					satTag,
					filepath.Join("/runnables", runnable.FQFN),
				))

				if err != nil {
					io.WriteString(os.Stderr, errors.Wrap(err, "sat exited with error").Error()+"\n")
				}

				time.Sleep(time.Millisecond * 500)
			}
		}()
	}
}

func unloadBundle(bundlePath string) (*directive.Directive, error) {
	bundle, err := bundle.Read(bundlePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to bundle.Read")
	}

	if err := bundle.Directive.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to Validate Directive")
	}

	folderPath := filepath.Join(os.TempDir(), "suborbital", directiveFileHash(bundle.Directive))

	// check if the folder already exists
	if _, err := os.Stat(folderPath); err != nil {
		if err := os.RemoveAll(folderPath); err != nil {
			return nil, errors.Wrap(err, "failed to RemoveAll existing bundle")
		}
	}

	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "faield to MkdirAll for bundle")
	}

	for _, r := range bundle.Directive.Runnables {
		filename := filepath.Join(folderPath, fmt.Sprintf("%s.wasm", r.FQFN))

		if err := os.WriteFile(filename, r.ModuleRef.Data, os.ModePerm); err != nil {
			return nil, errors.Wrapf(err, "failed to WriteFile for %s", r.FQFN)
		}
	}

	return bundle.Directive, nil
}

func directiveFileHash(directive *directive.Directive) string {
	sha := sha256.New()
	sha.Write([]byte(directive.Identifier))
	sha.Write([]byte(directive.AppVersion))

	return base64.URLEncoding.EncodeToString(sha.Sum(nil))
}

func randPort() (string, error) {
	// choose a random port above 1000
	randPort, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", errors.Wrap(err, "failed to rand.Int")
	}

	return fmt.Sprintf("%d", randPort.Int64()+10000), nil
}
