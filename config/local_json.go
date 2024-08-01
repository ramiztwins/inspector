package config

import (
	"encoding/json"
	"inspector/mylogger"
	"io/ioutil"
)

/*
 * Implementation of local configuration in the json format.
 * This will be extended to other types of configs in the future, database based configuration being the first priority.
 */

type InfluxDBSubConfig struct {
	DatabaseURL  string `json:"database_url"`
	Port         int    `json:"port"`
	DatabaseName string `json:"database_name"`
	Protocol     string `json:"transport_protocol"`
}

type MySQLDBSubConfig struct {
	DatabaseURL  string `json:"database_url"`
	Port         int    `json:"port"`
	DatabaseName string `json:"database_name"`
}

type MetricsDBSubConfig struct {
	*InfluxDBSubConfig `json:"influxdb,omitempty"`
	*MySQLDBSubConfig  `json:"mysqldb,omitempty"`
}

type ProberContextSubConfig struct {
	Url               string            `json:"url"`
	Method            string            `json:"method"`
	RequestParameters map[string]string `json:"parameters"`
	// Holds the list of cookies
	Cookies map[string]string `json:"cookies"`
	// Whether to follow http redirects from the server or not. Empty stanza uses the default 10 level redirect limit.
	AllowRedirects bool `json:"allow_redirects,omitempty"`
	Timeout        int  `json:"timeout,omitempty"`
}

// ProberSubConfig holds configuration of each prober.
type ProberSubConfig struct {
	// Freeform identifier of the current prober.
	Id string `json:"id"`
	// A specific type of prober. It's a misnomer, should be renamed to type.
	Name string `json:"name"`
	// Prober configuration, dependent on the type of the prover above.
	Context ProberContextSubConfig `json:"context"`
}

// TargetSubConfig is a logical grouping of probers belonging to same entity.
type TargetSubConfig struct {
	// Freeform identifier of the current target.
	Id string `json:"id"`
	// Freeform name of the current target.
	Name string `json:"name"`
	// List of probers that live under this target.
	Probers []ProberSubConfig `json:"probers"`
}

// InspectorSubConfig is Inspector's own config which is global per instance of inspector.
type InspectorSubConfig struct {
	// Arbitrary Region identifier in which current instance of inspector is running
	Region string `json:"region"`
}

type Config struct {
	Inspector    InspectorSubConfig   `json:"inspector"`
	TimeSeriesDB []MetricsDBSubConfig `json:"metrics_db"`
	Targets      []TargetSubConfig    `json:"targets"`
}

// NewConfig creates a new configuration. It currently assumes only json configuration.
// TODO: extend NewConfig to support other types of config.
func NewConfig(path string) (*Config, error) {
	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data Config
	err = json.Unmarshal(fileContent, &data)
	if err != nil {
		mylogger.MainLogger.Errorf("Failed parsing config at path: %s with error: %s", path, err)
		return nil, err
	}
	return &data, err
}