package sat

import (
	"net/http"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/atmo/atmo/coordinator/executor"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
)

func (s *Sat) handler(exec *executor.Executor) vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		spanCtx, span := s.tracer.Start(ctx.Context, "vkhandler", trace.WithAttributes(
			attribute.String("request_id", ctx.RequestID()),
		))
		defer span.End()

		ctx.Context = spanCtx

		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to FromVKRequest"))
			return vk.E(http.StatusInternalServerError, "unknown error")
		}

		var runErr rt.RunErr

		result, err := exec.Do(s.jobName, req, ctx, nil)
		if err != nil {
			if errors.As(err, &runErr) {
				// runErr would be an actual error returned from a function
				// should find a better way to determine if a RunErr is "non-nil"
				if runErr.Code != 0 || runErr.Message != "" {
					s.log.Debug("fn", s.jobName, "returned an error")
					return vk.E(runErr.Code, runErr.Message)
				}
			}

			s.log.Error(errors.Wrap(err, "failed to exec.Do"))
			return vk.E(http.StatusInternalServerError, "unknown error")
		}

		if result == nil {
			s.log.Debug("fn", s.jobName, "returned a nil result")
			return nil
		}

		resp := result.(*request.CoordinatedResponse)

		for headerKey, headerValue := range resp.RespHeaders {
			ctx.RespHeaders.Set(headerKey, headerValue)
		}

		return vk.RespondBytes(ctx.Context, w, resp.Output, http.StatusOK)
	}
}
