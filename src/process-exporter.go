// process-exporter is a metrics exporter for Prometheus environments.
// It is intended to provide cpu stats for processes and threads that match a given filter
package main

import (
	"flag"
	"fmt"
	"os"

	// "log"
	"net/http"

	"github.com/pcuzner/process-exporter/collector"
	"github.com/pcuzner/process-exporter/defaults"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
	})
}

func main() {
	var filter *string
	var withThreads *string
	port := flag.Int("port", defaults.DefaultPort, "port for the exporter to bind to")
	filter = flag.String("filter", "", "command of the process to search for (can be a comma separated list)")
	withThreads = flag.String("with-threads", "", "process names that should include per thread statistics (can be a comma separated list)")
	debug := flag.Bool("debug", true, "run in debug mode")
	noMatchAbort := flag.Bool("nomatch-abort", false, "shutdown if the filter doesn't match any active process")

	metricPrefix := flag.String("prefix", "proc", "prefix to use for the metric names returned to Prometheus")

	flag.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if envFilter, ok := os.LookupEnv("FILTER"); ok {
		log.Debug("Overriding -filter flag with FILTER environment variable: ", envFilter)
		*filter = envFilter
	}

	config := defaults.Config{
		Filter:       *filter,
		NoMatchAbort: *noMatchAbort,
		WithThreads:  *withThreads,
		MetricPrefix: *metricPrefix,
	}

	log.Info("Starting process-exporter")

	threadCollector := collector.NewThreadCollector(&config)
	prometheus.MustRegister(threadCollector)

	log.Infof("Binding to port %d", *port)
	http.Handle("/metrics", promhttp.Handler())

	listenAddr := fmt.Sprintf(":%d", *port)
	log.Fatal(http.ListenAndServe(listenAddr, nil))

}
