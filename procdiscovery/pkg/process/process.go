package process

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
)

const (
	NodeVersionConst   = "NODE_VERSION"
	PythonVersionConst = "PYTHON_VERSION"
	JavaVersionConst   = "JAVA_VERSION"
	PhpVersionConst    = "PHP_VERSION"
	RubyVersionConst   = "RUBY_VERSION"
)

const (
	// https://elixir.bootlin.com/linux/v6.5.5/source/include/uapi/linux/auxvec.h
	AT_SECURE = 23
)

// LangsVersionEnvs is a map of environment variables used for detecting the versions of different languages
var LangsVersionEnvs = map[string]struct{}{
	NodeVersionConst:   {},
	PythonVersionConst: {},
	JavaVersionConst:   {},
	PhpVersionConst:    {},
	RubyVersionConst:   {},
}

const (
	NewRelicAgentEnv = "NEW_RELIC_CONFIG_FILE"
)

var OtherAgentEnvs = map[string]string{
	NewRelicAgentEnv: "New Relic Agent",
}

var OtherAgentCmdSubString = map[string]string{
	"newrelic.jar": "New Relic Agent",
}

type Details struct {
	ProcessID           int
	ExePath             string
	CmdLine             string
	Environments        ProcessEnvs
	SecureExecutionMode *bool
}

// ProcessFile is a read-only interface that supports reading, seeking, and reading at specific positions.
// Write operations should be avoided.
type ProcessFile interface {
	io.ReadSeekCloser
	io.ReaderAt
}

type ProcessContext struct {
	Details
	exeFile  ProcessFile
	mapsFile ProcessFile
}

func NewProcessContext(details Details) *ProcessContext {
	return &ProcessContext{
		Details: details,
	}
}

// Close method to close any open file handles.
func (pcx *ProcessContext) CloseFiles() error {
	var err error
	if pcx.exeFile != nil {
		err = errors.Join(err, pcx.exeFile.Close())
		pcx.exeFile = nil
	}
	if pcx.mapsFile != nil {
		err = errors.Join(err, pcx.mapsFile.Close())
		pcx.mapsFile = nil
	}
	return err
}

func (pcx *ProcessContext) GetExeFile() (ProcessFile, error) {
	if pcx.exeFile == nil {
		path := fmt.Sprintf("/proc/%d/exe", pcx.ProcessID)
		fileData, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		pcx.exeFile = fileData
	} else {
		if _, err := pcx.exeFile.Seek(0, 0); err != nil {
			return nil, err // Return the seek error if it fails
		}
	}

	return pcx.exeFile, nil
}

func (pcx *ProcessContext) GetMapsFile() (ProcessFile, error) {
	if pcx.mapsFile == nil {
		mapsPath := fmt.Sprintf("/proc/%d/maps", pcx.ProcessID)
		fileData, err := os.Open(mapsPath)
		if err != nil {
			return nil, err
		}
		pcx.mapsFile = fileData
	} else {
		if _, err := pcx.mapsFile.Seek(0, 0); err != nil {
			return nil, err // Return the seek error if it fails
		}
	}
	return pcx.mapsFile, nil
}

type ProcessEnvs struct {
	DetailedEnvs map[string]string
	// OverwriteEnvs only contains environment variables that Odigos is using for auto-instrumentation and may need to be overwritten
	OverwriteEnvs map[string]string
}

func (d *Details) GetDetailedEnvsValue(key string) (string, bool) {
	value, exists := d.Environments.DetailedEnvs[key]
	return value, exists
}

func (d *Details) GetOverwriteEnvsValue(key string) (string, bool) {
	value, exists := d.Environments.OverwriteEnvs[key]
	return value, exists
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
	exePath := getExePath(pid)
	cmdLine := getCommandLine(pid)
	envVars := getRelevantEnvVars(pid)
	secureExecutionMode, err := isSecureExecutionMode(pid)
	secureExecutionModePtr := &secureExecutionMode
	if err != nil {
		secureExecutionModePtr = nil
	}

	return Details{
		ProcessID:           pid,
		ExePath:             exePath,
		CmdLine:             cmdLine,
		Environments:        envVars,
		SecureExecutionMode: secureExecutionModePtr,
	}
}

// The exe Symbolic Link: Inside each process's directory in /proc,
// there is a symbolic link named exe. This link points to the executable
// file that was used to start the process.
// For instance, if a process was started from /usr/bin/python,
// the exe symbolic link in that process's /proc directory will point to /usr/bin/python.
func getExePath(pid int) string {
	exeFileName := fmt.Sprintf("/proc/%d/exe", pid)
	exePath, err := os.Readlink(exeFileName)
	if err != nil {
		// Read link may fail if target process runs not as root
		return ""
	}
	return exePath
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

func getRelevantEnvVars(pid int) ProcessEnvs {
	envFileName := fmt.Sprintf("/proc/%d/environ", pid)
	fileContent, err := os.ReadFile(envFileName)
	if err != nil {
		// TODO: if we fail to read the environment variables, we should probably return an error
		// which will cause the process to be skipped and not instrumented?
		return ProcessEnvs{}
	}

	r := bufio.NewReader(strings.NewReader(string(fileContent)))

	// We only care about the environment variables that we might overwrite
	relevantOverwriteEnvVars := make(map[string]interface{})
	for k := range envOverwrite.EnvValuesMap {
		relevantOverwriteEnvVars[k] = nil
	}

	// Add LD_PRELOAD to the list of relevant environment variables
	relevantOverwriteEnvVars[consts.LdPreloadEnvVarName] = nil

	overWriteEnvsResult := make(map[string]string)
	detailedEnvsResult := make(map[string]string)

	for {
		// The entries are  separated  by
		// null bytes ('\0'), and there may be a null byte at the end.
		str, err := r.ReadString(0)
		if err == io.EOF {
			break
		} else if err != nil {
			return ProcessEnvs{}
		}

		str = strings.TrimRight(str, "\x00")
		envParts := strings.SplitN(str, "=", 2)
		if len(envParts) != 2 {
			continue
		}

		if _, ok := relevantOverwriteEnvVars[envParts[0]]; ok {
			overWriteEnvsResult[envParts[0]] = envParts[1]
		}

		if _, ok := LangsVersionEnvs[envParts[0]]; ok {
			detailedEnvsResult[envParts[0]] = envParts[1]
		}

		if _, ok := OtherAgentEnvs[envParts[0]]; ok {
			detailedEnvsResult[envParts[0]] = envParts[1]
		}
	}

	envs := ProcessEnvs{
		OverwriteEnvs: overWriteEnvsResult,
		DetailedEnvs:  detailedEnvsResult,
	}

	return envs
}

func isSecureExecutionMode(pid int) (bool, error) {
	path := fmt.Sprintf("/proc/%d/auxv", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read auxv: %w", err)
	}

	// https://www.man7.org/linux/man-pages/man5/proc_pid_auxv.5.html
	for i := 0; i+16 <= len(data); i += 16 {
		typ := binary.NativeEndian.Uint64(data[i : i+8])

		if typ == 0 {
			break
		}

		// from the linux man page:
		// A binary is executed in secure-execution mode if the AT_SECURE
		// entry in the auxiliary vector (see getauxval(3)) has a nonzero
		// value.
		if typ == AT_SECURE {
			val := binary.NativeEndian.Uint64(data[i+8 : i+16])
			return val != 0, nil
		}
	}

	return false, fmt.Errorf("AT_SECURE not found")
}
