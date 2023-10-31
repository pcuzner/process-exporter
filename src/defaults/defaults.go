// Package defaults defines the defaults settings consumed by other packages in the
// process-exporter module
package defaults

// "fmt"

// DefaultPort sets the defautl port to bind to
var DefaultPort int = 9200

// SystemPageSize is the multiplier used when calculating the RSS used. Procfs
// exposes this value as a number of pages, so this multiplier converts the pages to bytes.
var SystemPageSize int = 4096

// Config holds the runtime options to governhow the process-exporter will run
type Config struct {
	Filter       string
	NoMatchAbort bool
	WithThreads  bool
	MetricPrefix string
}

// ProcInfo struct describes all the attributes of a proc or thread that will be used
// to assemble the prometheus metrics
type ProcInfo struct {
	Pid         int
	Tid         int
	Daemon      string
	Comm        string
	CommandLine []string
	NumThreads  int
	STime       uint
	UTime       uint
	SyscR       uint64
	SyscW       uint64
	ReadBytes   uint64
	WriteBytes  uint64
	RSSbytes    int
	VSizeBytes  uint
}
