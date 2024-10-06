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
	metrics  []*influxdb_client.Point
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
	flxDB.metrics = make([]*influxdb_client.Point, 0)
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

// CollectMetrics accumulates metrics for subsequent sending.
func (flxDB *InfluxDB) CollectMetrics(m SingleMetric) {
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
	flxDB.metrics = append(flxDB.metrics, point)
}

// EmitMultiple sends all accumulated metrics to InfluxDB in one request.
func (flxDB *InfluxDB) EmitMultiple() {
	if len(flxDB.metrics) == 0 {
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
	bp.AddPoints(flxDB.metrics)

	// Send the batch of points to InfluxDB
	err = flxDB.client.Write(bp)
	if err != nil {
		fmt.Printf("Error writing to InfluxDB: %s\n", err)
		return
	}

	// Clear the accumulated metrics after successful sending
	flxDB.metrics = flxDB.metrics[:0]
}

// LaunchMetricsAggregation sets up retention policies and continuous queries in InfluxDB.
func (flxDB *InfluxDB) LaunchMetricsAggregation() error {
	statusMeasurement := "status"
	certificateMeasurement := "certificate_expiration"
	latencyMeasurements := []string{"connect_time", "response_time"}

	// Collect all queries to execute.
	queries := []string{
		CreateRetentionPolicyQuery(flxDB.database),
	}
	for _, measurement := range latencyMeasurements {
		queries = append(queries, LatencyQueries(flxDB.database, measurement)...)
	}
	queries = append(queries, StatusQueries(flxDB.database, statusMeasurement)...)
	queries = append(queries, CertificateExpirationQueries(flxDB.database, certificateMeasurement)...)

	// Execute all queries.
	for _, cmd := range queries {
		q := influxdb_client.Query{
			Command:  cmd,
			Database: flxDB.database,
		}
		response, err := flxDB.client.Query(q)
		if err != nil {
			return fmt.Errorf("error executing query '%s': %w", cmd, err)
		}
		if response.Error() != nil {
			return fmt.Errorf("query response error for query '%s': %w", cmd, response.Error())
		}
		// Error checking.
		for _, result := range response.Results {
			if result.Err != "" {
				return fmt.Errorf("query result error for query '%s': %s", cmd, result.Err)
			}
		}
	}
	return nil
}
