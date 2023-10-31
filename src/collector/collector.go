// Package collector controls the prometheus metrics collection logic
package collector

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pcuzner/process-exporter/defaults"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
)

// threadCollector defines the structure of the custom collector we're using to
// assemble the procfs related metrics that we're interested in
type threadCollector struct {
	config              defaults.Config
	procKernelCPUTime   *prometheus.Desc
	procUserCPUTime     *prometheus.Desc
	procRSS             *prometheus.Desc
	procSyscR           *prometheus.Desc
	procSyscW           *prometheus.Desc
	procReadBytes       *prometheus.Desc
	procWriteBytes      *prometheus.Desc
	procNumThreads      *prometheus.Desc
	threadKernelCPUTime *prometheus.Desc
	threadUserCPUTime   *prometheus.Desc
	procVSizeBytes      *prometheus.Desc
}

// NewThreadCollector returns a threadcollector instance
func NewThreadCollector(config *defaults.Config) *threadCollector {
	prefix := config.MetricPrefix
	return &threadCollector{
		config: *config,
		procKernelCPUTime: prometheus.NewDesc(
			fmt.Sprint(prefix, "_process_cpu_kernel_seconds_total"),
			"Kernel CPU usage of process",
			[]string{"pid", "pname", "daemon"}, nil,
		),
		procUserCPUTime: prometheus.NewDesc(
			fmt.Sprint(prefix, "_process_cpu_user_seconds_total"),
			"User CPU usage of process",
			[]string{"pid", "pname", "daemon"}, nil,
		),
		procRSS: prometheus.NewDesc(
			fmt.Sprint(prefix, "_process_rss_size_bytes"),
			"Memory consumed by the process (Resident Set Size)",
			[]string{"pid", "pname", "daemon"}, nil,
		),
		procSyscR: prometheus.NewDesc(
			fmt.Sprint(prefix, "_process_syscall_reads_total"),
			"Total of read syscalls issued by the process",
			[]string{"pid", "pname", "daemon"}, nil,
		),
		procSyscW: prometheus.NewDesc(
			fmt.Sprint(prefix, "_process_syscall_writes_total"),
			"Total of write syscalls issued by the process",
			[]string{"pid", "pname", "daemon"}, nil,
		),
		procReadBytes: prometheus.NewDesc(
			fmt.Sprint(prefix, "_process_read_bytes_total"),
			"Process read bytes completed",
			[]string{"pid", "pname", "daemon"}, nil,
		),
		procWriteBytes: prometheus.NewDesc(
			fmt.Sprint(prefix, "_process_written_bytes_total"),
			"Process write bytes completed",
			[]string{"pid", "pname", "daemon"}, nil,
		),
		procNumThreads: prometheus.NewDesc(
			fmt.Sprint(prefix, "_process_thread_total"),
			"Total threads associated with the process",
			[]string{"pid", "pname", "daemon"}, nil,
		),
		threadKernelCPUTime: prometheus.NewDesc(
			fmt.Sprint(prefix, "_thread_cpu_kernel_seconds_total"),
			"Kernel CPU usage of thread",
			[]string{"pid", "tid", "tname", "daemon"}, nil,
		),
		threadUserCPUTime: prometheus.NewDesc(
			fmt.Sprint(prefix, "_thread_cpu_user_seconds_total"),
			"User CPU usage of thread",
			[]string{"pid", "tid", "tname", "daemon"}, nil,
		),
		procVSizeBytes: prometheus.NewDesc(
			fmt.Sprint(prefix, "_process_virtual_memory_bytes_total"),
			"Virtual Memory size of the process (bytes)",
			[]string{"pid", "pname", "daemon"}, nil,
		),
	}
}

// Describe returns the metric descriptions
func (tCollector *threadCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- tCollector.procKernelCPUTime
	ch <- tCollector.procUserCPUTime
	ch <- tCollector.procRSS
	ch <- tCollector.procSyscR
	ch <- tCollector.procSyscW
	ch <- tCollector.procReadBytes
	ch <- tCollector.procWriteBytes
	ch <- tCollector.procNumThreads
	ch <- tCollector.threadKernelCPUTime
	ch <- tCollector.threadUserCPUTime
	ch <- tCollector.procVSizeBytes
}

// Collect is called by a GET request, and handles the data gathering and metrics
// assembly returning the metrics to the prometheus client over a channel
func (tCollector *threadCollector) Collect(ch chan<- prometheus.Metric) {

	log.Debug("Collect called")
	start := time.Now()

	matchingProcs := GetProcs(tCollector.config.Filter)
	elapsed := time.Since(start)
	log.Debugf("Looking for matching procs took: %s", elapsed)

	if len(matchingProcs) == 0 {
		if tCollector.config.NoMatchAbort {
			log.Errorf("No processes match filter: '%s'. Aborting", tCollector.config.Filter)
			os.Exit(4)
		} else {
			log.Warning("No processes match filter:", tCollector.config.Filter)
			return
		}
	}

	start = time.Now()
	var wg sync.WaitGroup
	procChannel := make(chan []defaults.ProcInfo, len(matchingProcs))
	wg.Add(len(matchingProcs))

	log.Debugf("Starting %d goroutines to gather the data", len(matchingProcs))
	for _, proc := range matchingProcs {
		GetProcInfo(proc, tCollector.config.WithThreads, procChannel, &wg)
	}
	wg.Wait()
	close(procChannel)

	elapsed = time.Since(start)
	log.Debugf("go-routines complete in : %s", elapsed)

	for procData := range procChannel {
		if len(procData) == 0 {
			continue
		}
		for _, procInfo := range procData {
			pid := procInfo.Pid
			if procInfo.Tid == 0 {
				// parent process metrics
				metric := prometheus.MustNewConstMetric(
					tCollector.procKernelCPUTime,
					prometheus.CounterValue,
					float64(procInfo.STime),
					strconv.Itoa(pid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric

				metric = prometheus.MustNewConstMetric(
					tCollector.procUserCPUTime,
					prometheus.CounterValue,
					float64(procInfo.UTime),
					strconv.Itoa(pid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric

				metric = prometheus.MustNewConstMetric(
					tCollector.procRSS,
					prometheus.GaugeValue,
					float64(procInfo.RSSbytes),
					strconv.Itoa(pid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric

				metric = prometheus.MustNewConstMetric(
					tCollector.procVSizeBytes,
					prometheus.GaugeValue,
					float64(procInfo.VSizeBytes),
					strconv.Itoa(pid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric

				// these are syscalls not IO requests
				metric = prometheus.MustNewConstMetric(
					tCollector.procSyscR,
					prometheus.CounterValue,
					float64(procInfo.SyscR),
					strconv.Itoa(pid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric

				metric = prometheus.MustNewConstMetric(
					tCollector.procSyscW,
					prometheus.CounterValue,
					float64(procInfo.SyscW),
					strconv.Itoa(pid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric

				metric = prometheus.MustNewConstMetric(
					tCollector.procReadBytes,
					prometheus.CounterValue,
					float64(procInfo.ReadBytes),
					strconv.Itoa(pid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric

				metric = prometheus.MustNewConstMetric(
					tCollector.procWriteBytes,
					prometheus.CounterValue,
					float64(procInfo.WriteBytes),
					strconv.Itoa(pid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric

				metric = prometheus.MustNewConstMetric(
					tCollector.procNumThreads,
					prometheus.GaugeValue,
					float64(procInfo.NumThreads),
					strconv.Itoa(pid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric
			} else {
				// thread metrics - only cpu is of interest
				metric := prometheus.MustNewConstMetric(
					tCollector.threadKernelCPUTime,
					prometheus.CounterValue,
					float64(procInfo.STime),
					strconv.Itoa(procInfo.Pid), strconv.Itoa(procInfo.Tid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric
				metric = prometheus.MustNewConstMetric(
					tCollector.threadUserCPUTime,
					prometheus.CounterValue,
					float64(procInfo.UTime),
					strconv.Itoa(procInfo.Pid), strconv.Itoa(procInfo.Tid), procInfo.Comm, procInfo.Daemon,
				)
				ch <- metric
			}

		}

	}

}
