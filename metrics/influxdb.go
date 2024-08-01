package metrics

import (
	"fmt"
	influxdb_client "github.com/influxdata/influxdb1-client/v2"
	"os"
	"time"
)

/*
 * Implementation of metrics collectors.
 * Currently only InfluxDB is supported.
 */

type InfluxDB struct {
	client   influxdb_client.Client
	addr     string
	port     int
	database string
}

// InitializeClient creates a new HTTP based InfluxDB client. This client will be used for the lifetime of the application.
func (flxDB *InfluxDB) InitializeClient(addr string, port int, database string) error {
	var err error
	flxDB.client, err = influxdb_client.NewHTTPClient(influxdb_client.HTTPConfig{
		Addr: fmt.Sprintf("http://%s:%d", addr, port),
	})
	if err != nil {
		return err
	}
	flxDB.port = port
	flxDB.addr = addr
	flxDB.database = database
	return nil
}

// EmitSingle sends a single metric out using the current InfluxDB client. It adds the source host where the metric is coming from.
func (flxDB *InfluxDB) EmitSingle(m SingleMetric) {
	if m.Tags == nil {
		m.Tags = make(map[string]string)
		m.Tags["host"], _ = os.Hostname()
	}
	_, ok := m.Tags["host"]
	if !ok {
		m.Tags["host"], _ = os.Hostname()
	}
	// additionalFields cannot be empty
	if m.AdditionalFields == nil {
		m.AdditionalFields = make(map[string]interface{})
	}
	m.AdditionalFields["value"] = m.Value
	point, err := influxdb_client.NewPoint(m.Name,
		m.Tags,
		m.AdditionalFields,
		time.Now())
	if err != nil {
		fmt.Printf("Error creating point: %s\n", err)
		return
	}
	
	bp, err := influxdb_client.NewBatchPoints(influxdb_client.BatchPointsConfig{
		Database:  flxDB.database,
		Precision: "ns",
	})
	if err != nil {
		fmt.Printf("Error creating batch points: %s\n", err)
		return
	}
	bp.AddPoint(point)

	// Send the batch of points to InfluxDB
	err = flxDB.client.Write(bp)
	if err != nil {
		fmt.Printf("Error writing to InfluxDB: %s\n", err)
	}
}