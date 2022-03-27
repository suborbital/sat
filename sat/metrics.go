package sat

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
)

type MetricsResponse struct {
	Scheduler rt.ScalerMetrics `json:"scheduler"`
}

func (s *Sat) metricsHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		metrics, err := s.exec.Metrics()
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to exec.Metrics"))
			return nil, vk.E(http.StatusInternalServerError, "unknown error")
		}

		resp := &MetricsResponse{
			Scheduler: *metrics,
		}

		return resp, nil
	}
}
