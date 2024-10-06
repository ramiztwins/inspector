// influxdb_cq.go
package metrics

import (
	"fmt"
)

/* 
* CreateRetentionPolicyQuery returns a query to create a retention policy named "report" with a duration of 30 days.
* Measurements with aggregated data (i call it 'reports') will be saved in this retention policy.
* For example, reports related to 'response_time' will be stored in inspector.report.response_time where
* 'inspector' - database, 'report' - retention policy, 'response_time' - measurement
*/
func CreateRetentionPolicyQuery(database string) string {
	return fmt.Sprintf(
		`CREATE RETENTION POLICY "report" ON "%s" DURATION 30d REPLICATION 1`,
		database,
	)
}

// CertificateExpirationQueries generates continuous queries for the certificate_expiration measurement.
func CertificateExpirationQueries(database string, measurement string) []string {
	query := fmt.Sprintf(
		`CREATE CONTINUOUS QUERY cq_%s ON "%s" BEGIN
			SELECT
				LAST(value) AS days_left
			INTO "%s"."report"."%s"
			FROM "%s"."autogen"."%s"
			GROUP BY time(1m), target_id, prober_id
		END`,
		measurement,
		database,
		database,
		measurement,
		database,
		measurement,
	)
	return []string{query}
}

// LatencyQueries generates continuous queries for latency-related measurements.
func LatencyQueries(database string, measurement string) []string {
	latencyQuery := fmt.Sprintf(
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
			GROUP BY time(1m), target_id, prober_id
		END`,
		measurement,
		database,
		database,
		measurement,
		database,
		measurement,
	)
	return []string{latencyQuery}
}

// StatusQueries generates continuous queries for status-related measurements.
func StatusQueries(database string, measurement string) []string {
	baseQuery := func(name, condition, alias string) string {
		return fmt.Sprintf(
			`CREATE CONTINUOUS QUERY %s ON "%s" BEGIN
				SELECT COUNT(value) AS %s
				INTO "%s"."report"."%s"
				FROM "%s"."autogen"."%s"
				WHERE %s
				GROUP BY time(1m), target_id, prober_id
			END`,
			name,
			database,
			alias,
			database,
			measurement, 
			database,
			measurement,
			condition,
		)
	}

	successQuery := baseQuery(
		"cq_success_counts",
		`value::integer >= 200 AND value::integer < 300`,
		"success_requests",
	)

	failQuery := baseQuery(
		"cq_fail_counts",
		`value::integer < 200 OR value::integer >= 300`,
		"failed_requests",
	)

	totalQuery := fmt.Sprintf(
		`CREATE CONTINUOUS QUERY cq_total_counts ON "%s" BEGIN
			SELECT COUNT(value) AS total_requests
			INTO "%s"."report"."%s"
			FROM "%s"."autogen"."%s"
			GROUP BY time(1m), target_id, prober_id
		END`,
		database,
		database,
		measurement,
		database,
		measurement,
	)
	return []string{successQuery, failQuery, totalQuery}
}