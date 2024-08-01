package probers

import (
	"fmt"
	"inspector/config"
	"inspector/metrics"
)

// Prober defines the minimal requirements for a new prober implementation.
// main package creates a new prober and calls these methods in a loop.
// Probers are not supposed to be reused. Every run of the prober must tear down after runOnce. Next run of the prober
// must create a new one. There are no safeguards for this, if you do reuse the prober -- you've been warned.
type Prober interface {
	// Initialize is a free form initialization function. Use it to initialize your prober's internal state.
	// You must initialize the target id at the very least.
	Initialize(targetID, proberID string) error
	// Connect is responsible for connection to the remote endpoint which is being monitored.
	Connect(chan metrics.SingleMetric) error
	// RunOnce is issued only once, and should include the main request logic for the prober.
	RunOnce(chan metrics.SingleMetric) error
	// TearDown is used for cleaning up the prober state. We do not reuse prober structures.
	TearDown() error
	// GetTargetID returns the target id this prober belongs to
	getTargetID() string
	// GetProberID returns the id of the current prober.
	getProberID() string
}

// NewProber creates a new prober using the type specific in the configuration file
// Currently only basic http prober is supported.
// TODO: add more prober types.
func NewProber(c config.ProberSubConfig) (Prober, error) {
	var newProber Prober
	switch c.Name {
	case "basic_http_prober":
		newProber = &HTTPProber{
			Url:            c.Context.Url,
			Method:         c.Context.Method,
			Parameters:     c.Context.RequestParameters,
			Cookies:        c.Context.Cookies,
			AllowRedirects: c.Context.AllowRedirects,
			Timeout:        c.Context.Timeout,
		}
		break
	default:
		return nil, fmt.Errorf("unsupported prober type: %s", c.Name)
	}
	return newProber, nil
}
