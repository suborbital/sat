package rt

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/vektor/vlog"
)

var ErrCapabilityNotAvailable = errors.New("capability not available")

// Capabilities define the capabilities available to a Runnable
type Capabilities struct {
	config rcap.CapabilityConfig

	Auth          rcap.AuthCapability
	LoggerSource  rcap.LoggerCapability
	HTTPClient    rcap.HTTPCapability
	GraphQLClient rcap.GraphQLCapability
	Cache         rcap.CacheCapability
	FileSource    rcap.FileCapability
	Database      rcap.DatabaseCapability

	// RequestHandler and doFunc are special because they are more
	// sensitive; they could cause memory leaks or expose internal state,
	// so they cannot be swapped out for a different implementation.
	RequestConfig  *rcap.RequestHandlerConfig
	RequestHandler rcap.RequestHandlerCapability
	doFunc         coreDoFunc
}

// DefaultCapabilities returns the default capabilities with the provided Logger
func DefaultCapabilities(logger *vlog.Logger) *Capabilities {
	// this will never error with the default config, as the db capability is disabled
	caps, _ := CapabilitiesFromConfig(rcap.DefaultConfigWithLogger(logger))

	return caps
}

func CapabilitiesFromConfig(config rcap.CapabilityConfig) (*Capabilities, error) {
	database, err := rcap.NewSqlDatabase(config.DB)
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewSqlDatabase")
	}

	caps := &Capabilities{
		config:        config,
		Auth:          rcap.DefaultAuthProvider(*config.Auth),
		LoggerSource:  rcap.DefaultLoggerSource(*config.Logger),
		HTTPClient:    rcap.DefaultHTTPClient(*config.HTTP),
		GraphQLClient: rcap.DefaultGraphQLClient(*config.GraphQL),
		Cache:         rcap.SetupCache(*config.Cache),
		FileSource:    rcap.DefaultFileSource(*config.File),
		Database:      database,

		// RequestHandler and doFunc don't get set here since they are set by
		// the rt and rwasm internals; a better solution for this should probably be found
		RequestConfig: config.RequestHandler,
	}

	return caps, nil
}

// Config returns the configuration that was used to create the Capabilities
// the config cannot be changed, but it can be used to determine what was
// previously set so that the orginal config (like enabled settings) can be respected
func (c Capabilities) Config() rcap.CapabilityConfig {
	return c.config
}
