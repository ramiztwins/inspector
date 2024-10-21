// influxdb.go
package metrics

import (
	"fmt"
	"os"
	"time"

	influxdb_client "github.com/influxdata/influxdb1-client/v2"
	"inspector/mylogger"
)

// InfluxDB implements the MetricsDB interface for InfluxDB.
type InfluxDB struct {
	client   influxdb_client.Client
	addr     string
	port     int
	database string
	metrics  []*influxdb_client.Point
}

// InitializeClient creates a new HTTP-based InfluxDB client.
func (db *InfluxDB) InitializeClient(addr string, port int, database string) error {
	var err error
	db.client, err = influxdb_client.NewHTTPClient(influxdb_client.HTTPConfig{
		Addr: fmt.Sprintf("http://%s:%d", addr, port),
	})
	if err != nil {
		return err
	}
	db.addr = addr
	db.port = port
	db.database = database
	db.metrics = make([]*influxdb_client.Point, 0)
	return nil
}

// EmitSingle sends a single metric to InfluxDB.
func (db *InfluxDB) EmitSingle(m SingleMetric) {
	db.collectMetric(m)
	db.EmitMultiple()
}

// CollectMetrics accumulates metrics for batch sending.
func (db *InfluxDB) CollectMetrics(m SingleMetric) {
	db.collectMetric(m)
}

// collectMetric prepares a metric point for sending.
func (db *InfluxDB) collectMetric(m SingleMetric) {
	if m.Tags == nil {
		m.Tags = make(map[string]string)
	}
	if _, ok := m.Tags["host"]; !ok {
		m.Tags["host"], _ = os.Hostname()
	}
	if m.AdditionalFields == nil {
		m.AdditionalFields = make(map[string]interface{})
	}
	m.AdditionalFields["value"] = m.Value

	point, err := influxdb_client.NewPoint(m.Name, m.Tags, m.AdditionalFields, time.Now())
	if err != nil {
		mylogger.MainLogger.Errorf("Error creating point: %s", err)
		return
	}
	db.metrics = append(db.metrics, point)
}

// EmitMultiple sends all accumulated metrics to InfluxDB.
func (db *InfluxDB) EmitMultiple() {
	if len(db.metrics) == 0 {
		return
	}
	bp, err := influxdb_client.NewBatchPoints(influxdb_client.BatchPointsConfig{
		Database:  db.database,
		Precision: "ns",
	})
	if err != nil {
		mylogger.MainLogger.Errorf("Error creating batch points: %s", err)
		return
	}
	bp.AddPoints(db.metrics)

	if err = db.client.Write(bp); err != nil {
		mylogger.MainLogger.Errorf("Error writing to InfluxDB: %s", err)
		return
	}
	db.metrics = db.metrics[:0]
}

// LaunchAggregation sets up retention policies and continuous queries.
func (db *InfluxDB) LaunchAggregation() error {
	if err := db.createRetentionPolicy(); err != nil {
		mylogger.MainLogger.Errorf("Failed to create retention policy: %s", err)
		return err
	}

	if err := db.createContinuousQueries(); err != nil {
		mylogger.MainLogger.Errorf("Failed to create continuous queries: %s", err)
		return err
	}

	return nil
}

// createRetentionPolicy creates a retention policy for aggregated data.
func (db *InfluxDB) createRetentionPolicy() error {
	query := fmt.Sprintf(`CREATE RETENTION POLICY "report" ON "%s" DURATION 30d REPLICATION 1`, db.database)
	return db.executeQuery(query)
}

// createContinuousQueries creates continuous queries for data aggregation.
func (db *InfluxDB) createContinuousQueries() error {
	measurements := []string{"certificate_expiration", "connect_time", "response_time", "status"}

	for _, measurement := range measurements {
		var queries []string

		switch measurement {
		case "certificate_expiration":
			queries = db.certificateExpirationQueries(measurement)
		case "connect_time", "response_time":
			queries = db.latencyQueries(measurement)
		case "status":
			queries = db.statusQueries(measurement)
		default:
			mylogger.MainLogger.Errorf("No continuous queries defined for measurement: %s", measurement)
			continue
		}

		for _, q := range queries {
			if err := db.executeQuery(q); err != nil {
				mylogger.MainLogger.Errorf("Failed to execute query for measurement %s: %s", measurement, err)
				return err
			}
		}
	}

	return nil
}

// executeQuery runs a query against InfluxDB.
func (db *InfluxDB) executeQuery(query string) error {
	q := influxdb_client.Query{
		Command:  query,
		Database: db.database,
	}
	response, err := db.client.Query(q)
	if err != nil {
		return err
	}
	if response.Error() != nil {
		return response.Error()
	}
	return nil
}

// certificateExpirationQueries generates continuous queries for certificate expiration.
func (db *InfluxDB) certificateExpirationQueries(measurement string) []string {
	query := fmt.Sprintf(
		`CREATE CONTINUOUS QUERY cq_%s ON "%s" BEGIN
			SELECT LAST(value) AS days_left
			INTO "%s"."report"."%s"
			FROM "%s"."autogen"."%s"
			GROUP BY time(1m), target_id, prober_id, region, host
		END`,
		measurement, db.database, db.database, measurement, db.database, measurement,
	)
	return []string{query}
}

// latencyQueries generates continuous queries for latency measurements.
func (db *InfluxDB) latencyQueries(measurement string) []string {
	query := fmt.Sprintf(
		`CREATE CONTINUOUS QUERY cq_%s ON "%s" BEGIN
			SELECT 
				ROUND(MEAN(value) * 100) / 100 AS avg_time,
				ROUND(MIN(value) * 100) / 100 AS min_time,
				ROUND(MAX(value) * 100) / 100 AS max_time,
				ROUND(STDDEV(value) * 100) / 100 AS stddev_time,
				ROUND(MEDIAN(value) * 100) / 100 AS median,
				ROUND(PERCENTILE(value, 90) * 100) / 100 AS p90_time,
				ROUND(PERCENTILE(value, 95) * 100) / 100 AS p95_time,
				ROUND(PERCENTILE(value, 99) * 100) / 100 AS p99_time
			INTO "%s"."report"."%s"
			FROM "%s"."autogen"."%s"
			GROUP BY time(1m), target_id, prober_id, region, host
		END`,
		measurement, db.database, db.database, measurement, db.database, measurement,
	)
	return []string{query}
}

// statusQueries generates continuous queries for status-related measurements.
func (db *InfluxDB) statusQueries(measurement string) []string {
	baseQuery := func(cqName, condition, fieldAlias string) string {
		return fmt.Sprintf(
			`CREATE CONTINUOUS QUERY %s ON "%s" BEGIN
				SELECT COUNT(value) AS %s
				INTO "%s"."report"."%s"
				FROM "%s"."autogen"."%s"
				WHERE %s
				GROUP BY time(1m), target_id, prober_id, region, host
			END`,
			cqName,
			db.database,
			fieldAlias,
			db.database,
			measurement,
			db.database,
			measurement,
			condition,
		)
	}
	
	successQuery := baseQuery(
		"cq_status_success",
		`value >= 200 AND value < 300`,
		"success_requests",
	)
	failQuery := baseQuery(
		"cq_status_fail",
		`value < 200 OR value >= 300`,
		"failed_requests",
	)
	totalQuery := baseQuery(
		"cq_status_total",
		`TRUE`,
		"total_requests",
	)
	
	return []string{successQuery, failQuery, totalQuery}
}