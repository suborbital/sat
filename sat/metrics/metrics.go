package metrics

import (
	"github.com/go-kit/kit/metrics"
)

type Metrics struct {
	FunctionExecutions metrics.Counter
	FunctionTime       metrics.Histogram
}
