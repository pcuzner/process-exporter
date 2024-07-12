package collector

import (
	// "fmt"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/pcuzner/process-exporter/defaults"
	"github.com/pcuzner/process-exporter/utils"
	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

// FS mount point for procfs
var FS procfs.FS
var procPath string

type ProcMatch struct {
	procInfo    procfs.Proc
	withThreads bool
	comm        string
}

func init() {

	if _, err := os.Stat("/host/proc"); err == nil {
		// running as a container with proc mounted at /host
		procPath = "/host/proc"
	} else {
		procPath = "/proc"
	}
	FS, _ = procfs.NewFS(procPath)
	fmt.Println("Using proc filesystem at ", FS)
}

// GetProcs returns a slice of Proc structs (pids) that match a given name
func GetProcs(filter string, withThreads string) []*ProcMatch {
	var procData []*ProcMatch

	targets := strings.Split(filter, ",")
	threadProcesses := strings.Split(withThreads, ",")

	procs, err := FS.AllProcs()

	if err != nil {
		panic("Aborting. Call to procfs.AllProcs() failed")
	}
	log.Debugf("procfs has %d pids", len(procs))
	for _, proc := range procs {
		c, _ := proc.Comm()
		if utils.Contains(targets, c) {
			withThreads := utils.Contains(threadProcesses, c)
			procData = append(procData, &ProcMatch{procInfo: proc, withThreads: withThreads, comm: c})
		}

	}
	return procData
}

// GetProcInfo runs as a goroutine to gather the proc information for a given proc
func GetProcInfo(proc procfs.Proc, withThreads bool, ch chan []defaults.ProcInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	// TODO catch the potential error
	stats, _ := getProc(proc, withThreads)
	ch <- stats
}

// getProc extracts the proc data of interest and returns a slice of ProcInfo structs to the caller
func getProc(proc procfs.Proc, withThreads bool) ([]defaults.ProcInfo, error) {
	var err error
	stats := []defaults.ProcInfo{}

	pid := proc.PID // int

	// TODO: tidy this up to catch potential errors in the calls
	comm, _ := proc.Comm()
	cmdLine, _ := proc.CmdLine()
	daemonName := utils.GetDaemonName(cmdLine)
	procStats, _ := proc.Stat()
	ioStats, _ := proc.IO()
	procStatus, _ := proc.NewStatus()

	info := defaults.ProcInfo{
		Pid:            pid,
		Tid:            0,
		Daemon:         daemonName,
		Comm:           comm,
		CommandLine:    cmdLine,
		NumThreads:     procStats.NumThreads,
		STime:          procStats.STime,
		UTime:          procStats.UTime,
		SyscR:          ioStats.SyscR,
		SyscW:          ioStats.SyscW,
		ReadBytes:      ioStats.ReadBytes,
		WriteBytes:     ioStats.WriteBytes,
		RSSbytes:       (procStats.RSS * defaults.SystemPageSize),
		VSizeBytes:     procStats.VSize,
		HugePagesBytes: procStatus.HugetlbPages,
	}
	stats = append(stats, info)

	if withThreads {
		threadData, err := GetThreadData(pid, daemonName)
		if err != nil {
			log.Error("unable to fetch threads for PID", pid)
		}
		log.Debugf("Proc %d has %d threads", pid, len(threadData))
		// merge the threads to the main slice
		stats = append(stats, threadData...)
	}

	return stats, err
}

// GetThreadData uses the same proc interface to get thread level CPU usage of a given process
func GetThreadData(pid int, daemonName string) ([]defaults.ProcInfo, error) {
	threadStats := []defaults.ProcInfo{}
	threadPIDs, err := FS.AllThreads(pid)
	if err != nil {
		return threadStats, err
	}
	for _, proc := range threadPIDs {
		tid := proc.PID // int

		// TODO: tody this up to catch potential errors in the calls
		comm, _ := proc.Comm()
		cmdLine, _ := proc.CmdLine()
		procStats, _ := proc.Stat()

		threadStats = append(threadStats, defaults.ProcInfo{
			Pid:         pid,
			Tid:         tid,
			Daemon:      daemonName,
			Comm:        comm,
			CommandLine: cmdLine,
			STime:       procStats.STime,
			UTime:       procStats.UTime,
		})

	}
	return threadStats, nil
}
