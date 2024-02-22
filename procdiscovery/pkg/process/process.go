package process

import (
	"os"
	"path"
	"strconv"
)

type Details struct {
	ProcessID int
	ExeName   string
	CmdLine   string
}

// Find all processes in the system.
// The function accepts a predicate function that can be used to filter the results.
func FindAllProcesses(predicate func(string) bool) ([]Details, error) {

	dirs, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var result []Details
	for _, di := range dirs {

		if !di.IsDir() {
			continue
		}

		dirName := di.Name()

		pid, isProcessDirectory := isDirectoryPid(dirName)
		if !isProcessDirectory {
			continue
		}

		// predicate is optional, and can be used to filter the results
		// plus avoid doing unnecessary work (e.g. reading the command line and exe name)
		if predicate != nil && !predicate(dirName) {
			continue
		}

		details := getPidDetails(pid, dirName)
		result = append(result, details)
	}

	return result, nil
}

func GetPidDetails(pid int) Details {
	pidStr := strconv.Itoa(pid)
	return getPidDetails(pid, pidStr)
}

func getPidDetails(pid int, pidStr string) Details {
	procDirName := path.Join("/proc", pidStr)
	exeName := getExecName(procDirName)
	cmdLine := getCommandLine(procDirName)

	return Details{
		ProcessID: pid,
		ExeName:   exeName,
		CmdLine:   cmdLine,
	}
}

// The exe Symbolic Link: Inside each process's directory in /proc,
// there is a symbolic link named exe. This link points to the executable
// file that was used to start the process.
// For instance, if a process was started from /usr/bin/python,
// the exe symbolic link in that process's /proc directory will point to /usr/bin/python.
func getExecName(procDirName string) string {
	exeName, err := os.Readlink(path.Join("/proc", procDirName, "exe"))
	if err != nil {
		// Read link may fail if target process runs not as root
		return ""
	}
	return exeName
}

// reads the command line arguments of a Linux process from
// the cmdline file in the /proc filesystem and converts them into a string
func getCommandLine(procDirName string) string {
	fileContent, err := os.ReadFile(path.Join("/proc", procDirName, "cmdline"))
	if err != nil {
		// Ignore errors
		return ""
	} else {
		return string(fileContent)
	}
}
