package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/odigos-io/odigos/common/envOverwrite"
)

type Details struct {
	ProcessID int
	ExeName   string
	CmdLine   string
	// Envs only contains the environment variables that we are interested in
	Envs map[string]string
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

		details := GetPidDetails(pid)
		result = append(result, details)
	}

	return result, nil
}

func GetPidDetails(pid int) Details {
	exeName := getExecName(pid)
	cmdLine := getCommandLine(pid)
	envVars := getRelevantEnvVars(pid)

	return Details{
		ProcessID: pid,
		ExeName:   exeName,
		CmdLine:   cmdLine,
		Envs:      envVars,
	}
}

// The exe Symbolic Link: Inside each process's directory in /proc,
// there is a symbolic link named exe. This link points to the executable
// file that was used to start the process.
// For instance, if a process was started from /usr/bin/python,
// the exe symbolic link in that process's /proc directory will point to /usr/bin/python.
func getExecName(pid int) string {
	exeFileName := fmt.Sprintf("/proc/%d/exe", pid)
	exeName, err := os.Readlink(exeFileName)
	if err != nil {
		// Read link may fail if target process runs not as root
		return ""
	}
	return exeName
}

// reads the command line arguments of a Linux process from
// the cmdline file in the /proc filesystem and converts them into a string
func getCommandLine(pid int) string {
	cmdLineFileName := fmt.Sprintf("/proc/%d/cmdline", pid)
	fileContent, err := os.ReadFile(cmdLineFileName)
	if err != nil {
		// Ignore errors
		return ""
	} else {
		return string(fileContent)
	}
}

func getRelevantEnvVars(pid int) map[string]string {
	envFileName := fmt.Sprintf("/proc/%d/environ", pid)
	fileContent, err := os.ReadFile(envFileName)
	if err != nil {
		// TODO: if we fail to read the environment variables, we should probably return an error
		// which will cause the process to be skipped and not instrumented?
		return nil
	}

	r := bufio.NewReader(strings.NewReader(string(fileContent)))
	result := make(map[string]string)

	// We only care about the environment variables that we might overwrite
	relevantEnvVars := make(map[string]interface{})
	for k := range envOverwrite.EnvValuesMap {
		relevantEnvVars[k] = nil
	}

	for {
		// The entries are  separated  by
		// null bytes ('\0'), and there may be a null byte at the end.
		str, err := r.ReadString(0)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil
		}

		str = strings.TrimRight(str, "\x00")

		envParts := strings.SplitN(str, "=", 2)
		if len(envParts) != 2 {
			continue
		}

		if _, ok := relevantEnvVars[envParts[0]]; ok {
			result[envParts[0]] = envParts[1]
		}
	}

	return result
}
