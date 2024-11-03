package metrics

import (
	"fmt"
	"inspector/config"
	"inspector/mylogger"
)

type SingleMetric struct {
	Name             string
	Value            int64
	AdditionalFields map[string]interface{}
	Tags             map[string]string
}

type MetricsDB interface {
	InitializeClient(addr string, port int, database string) error
	EmitSingle(m SingleMetric)
	CollectMetrics(m SingleMetric)
	EmitMultiple()
	LaunchAggregation() error
}

func CreateSingleMetric(name string, value int64, additionalFields map[string]interface{}, tags map[string]string) SingleMetric {
	return SingleMetric{
		Name:             name,
		Value:            value,
		AdditionalFields: additionalFields,
		Tags:             tags,
	}
}

// NewMetricsDB initializes a metrics database specified by the config.
// Currently, only InfluxDB is supported.
func NewMetricsDB(c config.MetricsDBSubConfig) (MetricsDB, error) {
	var mdb MetricsDB
	if c.InfluxDBSubConfig != nil {
		mdb = &InfluxDB{}
		err := mdb.InitializeClient(c.InfluxDBSubConfig.DatabaseURL, c.InfluxDBSubConfig.Port, c.InfluxDBSubConfig.DatabaseName)
		if err != nil {
			return nil, err
		}
	} else {
		mylogger.MainLogger.Errorf("Specified metrics database is not supported in config: %v", c)
		return nil, fmt.Errorf("MetricsDB defined in configuration is not supported: %v", c)
	}
	return mdb, nil
}
