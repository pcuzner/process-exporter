// Package utils provides miscellaneous functions that may be called by other packages
package utils

// Contains returns a bool indicating true or false for the existance of a value in a slice
func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}

// IndexOf returns the position of an element within a slice
func IndexOf[T comparable](list []T, searchItem T) int {
	for idx, item := range list {
		if item == searchItem {
			return idx
		}
	}
	return -1
}

// GetDaemonName examines the process commandline and returns a shortened version,
// so it can be used within a metric label. At this point it focuses only on
// ceph processes that have been defined through the cephadm orchestrator.
func GetDaemonName(commandLine []string) string {
	daemonName := ""

	// Ceph containers running under podman (cephadm)
	pos := IndexOf(commandLine, "-n")
	if pos >= 0 {
		daemonName = commandLine[pos+1]
	}
	return daemonName
}
