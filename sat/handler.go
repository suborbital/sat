package sat

import (
	"net/http"

	"github.com/pkg/errors"

	// company packages.
	"github.com/suborbital/atmo/atmo/coordinator/executor"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
)

func (s *Sat) handler(exec *executor.Executor) vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to FromVKRequest"))
			return nil, vk.E(http.StatusInternalServerError, "unknown error")
		}

		var runErr rt.RunErr

		result, err := exec.Do(s.j, req, ctx, nil)
		if err != nil {
			if !errors.As(err, runErr) {
				s.l.Error(errors.Wrap(err, "failed to exec.Do"))
				return nil, vk.E(http.StatusInternalServerError, "unknown error")
			}
		} else if result == nil {
			s.l.Debug("fn", s.j, "returned a nil result")

			return nil, nil
		}

		// runErr would be an actual error returned from a function
		// should find a better way to determine if a RunErr is "non-nil"
		if runErr.Code != 0 || runErr.Message != "" {
			s.l.Debug("fn", s.j, "returned an error")
			return nil, vk.E(runErr.Code, runErr.Message)
		}

		resp := result.(*request.CoordinatedResponse)

		for headerKey, headerValue := range resp.RespHeaders {
			ctx.RespHeaders.Set(headerKey, headerValue)
		}

		return resp.Output, nil
	}
}
