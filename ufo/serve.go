package ufo

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/webteleport/auth"
)

// DefaultTimeout is the default dialing timeout for the UFO server.
var DefaultTimeout = 10 * time.Second

// DefaultGcInterval is the default garbage collection interval for the UFO server.
var DefaultGcInterval = 0 * time.Second

// DefaultGcRetry is the default garbage collection retry limit.
var DefaultGcRetry int64 = 3

// Serve starts a UFO server on the given station URL.
func Serve(stationURL string, handler http.Handler) error {
	// Parse the station URL and inject client info
	u, err := createURLWithQueryParams(stationURL)
	if err != nil {
		return err
	}

	// Parse the 'quiet' query parameter
	quiet, err := parseQuietParam(u.Query())
	if err != nil {
		return err
	}

	// Parse the 'timeout' query parameter
	timeout, err := parseTimeoutParam(u.Query())
	if err != nil {
		return err
	}

	// Parse the 'gc' query parameter
	interval, err := parseGcIntervalParam(u.Query())
	if err != nil {
		return err
	}

	// Parse the 'retry' query parameter
	retry, err := parseGcRetryParam(u.Query())
	if err != nil {
		return err
	}

	// Serve with the parsed configuration
	return ServeWithConfig(&ServerConfig{
		StationURL: u,
		Handler:    handler,
		Timeout:    timeout,
		GcInterval: interval,
		GcRetry:    retry,
		Quiet:      quiet,
	})
}

// ServerConfig is the configuration for the UFO server.
type ServerConfig struct {
	StationURL *url.URL
	Handler    http.Handler
	Timeout    time.Duration
	GcInterval time.Duration
	GcRetry    int64
	Quiet      bool
}

// Serve starts a UFO server on the given station URL.
func ServeWithConfig(config *ServerConfig) error {
	u := config.StationURL

	// listen on the station URL with a timeout
	ln, err := listenWithTimeout(u.String(), config.Timeout)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	// log the status of the server
	if !config.Quiet {
		logServerStatus(ln, u)
	}

	// use the default serve mux if nil handler is provided
	if config.Handler == nil {
		config.Handler = http.DefaultServeMux
	}

	// close the listener when the server is unresponsive
	if config.GcInterval > 0 {
		go gc(ln, config.GcInterval, config.GcRetry)
	}

	err = http.Serve(ln, auth.WithPassword(config.Handler, u.Fragment))
	if err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
