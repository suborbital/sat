package sat

import (
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vlog"
)

func connectStaticPeers(logger *vlog.Logger, g *grav.Grav) error {
	count := 0
	var err error

	endpoints, useStatic := os.LookupEnv("SAT_PEERS")
	if useStatic {
		epts := strings.Split(endpoints, ",")

		for _, e := range epts {
			logger.Debug("connecting to static peer", e)

			for count < 10 {
				if err = g.ConnectEndpoint(e); err != nil {
					logger.Error(errors.Wrap(err, "failed to ConnectEndpoint, will retry"))
					count++

					time.Sleep(time.Second * 3)
				} else {
					break
				}
			}
		}
	}

	return err
}
