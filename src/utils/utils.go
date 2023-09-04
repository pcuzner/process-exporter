// Package utils provides miscellaneous functions that may be called by other packages
package utils

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

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

	// check for commandline not being populated yet (/proc entry being built or destroyed?)
	if len(commandLine) == 0 {
		return ""
	}

	// Ceph containers running under cephadm control
	pos := IndexOf(commandLine, "-n")
	if pos >= 0 {
		return commandLine[pos+1]
	}

	// ganesha et al
	cmdString := strings.Join(commandLine, " ")
	log.Debug("daemon is not a native Ceph daemon/client. cmdline is: ", cmdString)
	hostname, err := os.Hostname()
	var fqdn string = ""
	if err == nil {
		hostParts := strings.Split(hostname, ".")
		fqdn = hostParts[0]
	}
	if commandLine[0] == "/usr/bin/ganesha.nfsd" {
		return "ganesha." + fqdn
	}
	if commandLine[0] == "/usr/local/nvmef_tgt" {
		return "nvmeof_tgt." + fqdn
	}
	if commandLine[0] == "haproxy" {
		return "haproxy." + fqdn
	}

	return ""
}
