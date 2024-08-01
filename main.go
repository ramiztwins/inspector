package main

import (
	"flag"
	glogger "github.com/google/logger"
	"inspector/config"
	"inspector/metrics"
	"inspector/mylogger"
	"inspector/probers"
	"io"
	"math/rand"
	"os"
	"time"
)

var METRIC_CHANNEL_POLL_INTERVAL = 2 * time.Second
var TARGET_LIST_SCAN_WAIT_INTERVAL = 4 * time.Second
var PROBER_RESTART_INTERVAL_JITTER_RANGE = 2
var METRIC_CHANNEL_SIZE = 400

func main() {

	var configPath = flag.String("config_path", "", "Path to the configuration file. Mandatory argument.")
	var logFilePath = flag.String("log_path", "", "A file where to write logs. Optional argument, defaults to stdout")

	flag.Parse()

	if *logFilePath != "" {
		logFile, err := os.OpenFile(*logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
		if err != nil {
			glogger.Fatalf("Failed to open log file path: %s, error: %s", logFilePath, err)
		}
		defer logFile.Close()
		mylogger.MainLogger = glogger.Init("InspectorLogger", false, true, logFile)
	} else {
		mylogger.MainLogger = glogger.Init("InspectorLogger", true, true, io.Discard)
	}

	if *configPath == "" {
		mylogger.MainLogger.Errorf("Missing a mandatory argument: config_path. Try -help option for the list of " +
			"supported arguments")
		os.Exit(1)
	}
	c, err := config.NewConfig(*configPath)
	if err != nil {
		mylogger.MainLogger.Infof("Error reading config: %s", err)
		os.Exit(1)
	}
	mylogger.MainLogger.Infof("Config parsed: %v", c.TimeSeriesDB[0])

	//TODO: enable support for multiple time series databases. For now only the first one is used from config.
	mdb, err := metrics.NewMetricsDB(c.TimeSeriesDB[0])
	if err != nil {
		mylogger.MainLogger.Infof("Failed initializing metrics db client with error: %s", err)
		os.Exit(1)
	}
	mylogger.MainLogger.Infof("Initialized metrics database...")

	// TODO: determine what should the size of the channel be ?
	metricsChannel := make(chan metrics.SingleMetric, METRIC_CHANNEL_SIZE)

	/*
	 * Kick off an async  metrics collection from the metrics channel. Metrics are pushed into the metrics channel
	 * by probers. Collected metrics are pushed out to the currently configured metrics database.
	 */
	go func() {
		for {
			select {
			case m := <-metricsChannel:
				m.Tags["region"] = c.Inspector.Region
				mdb.EmitSingle(m)
			default:
				mylogger.MainLogger.Infof("Metrics channel is empty. Waiting some more...")
				time.Sleep(METRIC_CHANNEL_POLL_INTERVAL)
			}
		}
	}()

	/*
	 * Iterate over every target defined in the config, and per each target asynchronously initialize and run configured
	 * probers. Each prober will inject metrics into the metrics channel. The probers are expected to be implemented
	 * using the Prober interface.
	 * Probers are not reused and run only once per iteration.
	 *
	 * TODO: as a further optimization, the targets can be partitioned and processed asynchronously. This will help
	 *       if the number of targets become extremely large, but for now it's not a priority.
	 */
	for {
		for _, target := range c.Targets {
			for _, proberSubConfig := range target.Probers {
				go func() {
					prober, err := probers.NewProber(proberSubConfig)
					if err != nil {
						mylogger.MainLogger.Errorf("Failed creating new prober: %s for target: %s, error: %s",
							proberSubConfig.Name, target.Name, err)
						return
					}
					err = prober.Initialize(target.Id, proberSubConfig.Id)
					if err != nil {
						mylogger.MainLogger.Errorf("Failed initializing prober: %s for target: %s, error: %s",
							proberSubConfig.Name, target.Name, err)
						return
					}
					mylogger.MainLogger.Infof("Successfully initialized prober: %s for target: %s",
						proberSubConfig.Name, target.Name)

					err = prober.Connect(metricsChannel)
					if err != nil {
						mylogger.MainLogger.Errorf("Failed prober connection: %s for target: %s, error: %s",
							proberSubConfig.Name, target.Name, err)
						return
					}
					mylogger.MainLogger.Infof("Successful prober connection: %s for target: %s",
						proberSubConfig.Name, target.Name)

					err = prober.RunOnce(metricsChannel)
					if err != nil {
						mylogger.MainLogger.Errorf("Failed running prober: %s for target: %s, error: %s",
							proberSubConfig.Name, target.Name, err)
						return
					}
					err = prober.TearDown()
					if err != nil {
						mylogger.MainLogger.Errorf("Failed tearing down prober: %s for target: %s, error: %s",
							proberSubConfig.Name, target.Name, err)
						return
					}
					mylogger.MainLogger.Infof("Successfully torn down prober: %s for target: %s",
						proberSubConfig.Name, target.Name)
				}()

				jitter := rand.Intn(PROBER_RESTART_INTERVAL_JITTER_RANGE)
				time.Sleep(time.Duration(jitter) * time.Second)
			}
		}
		// Wait before scanning through the targets from scratch
		time.Sleep(TARGET_LIST_SCAN_WAIT_INTERVAL)
	}
}
