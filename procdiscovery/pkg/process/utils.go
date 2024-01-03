package process

import "strconv"

// The content of '/proc' can be categorized into two types:
// 1. Process directories, which are named with process IDs.
// 2. Textual or named entries, which are none-process-related system information,
// and provide global kernel information and parameters that are not specific to any single process.
// for example, '/proc/cpuinfo' and '/proc/meminfo'.
//
// givin a directory name in '/proc', this function returns if it is a process directory,
// and if so, the process ID.
func isDirectoryPid(procDirectoryName string) (int, bool) {
	if procDirectoryName[0] < '0' || procDirectoryName[0] > '9' {
		return 0, false
	}

	pid, err := strconv.Atoi(procDirectoryName)
	if err != nil {
		return 0, false
	}

	return pid, true
}
