package process

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	NodeVersionConst   = "NODE_VERSION"
	PythonVersionConst = "PYTHON_VERSION"
	JavaVersionConst   = "JAVA_VERSION"
	JavaHomeConst      = "JAVA_HOME"
	PhpVersionConst    = "PHP_VERSION"
	RubyVersionConst   = "RUBY_VERSION"
)

const (
	// https://elixir.bootlin.com/linux/v6.5.5/source/include/uapi/linux/auxvec.h
	AT_SECURE = 23

	defaultProcDir = "/proc"
)

var procDir = func() string {
	dir, ok := os.LookupEnv("ODIGOS_PROC_DIR")
	if !ok || dir == "" {
		return defaultProcDir
	}
	return dir
}()

func HostProcDir() string {
	return procDir
}

// LangsVersionEnvs is a map of environment variables used for detecting the versions of different languages
var LangsVersionEnvs = map[string]struct{}{
	NodeVersionConst:   {},
	PythonVersionConst: {},
	JavaVersionConst:   {},
	JavaHomeConst:      {},
	PhpVersionConst:    {},
	RubyVersionConst:   {},
}

const (
	NewRelicAgentName  = "New Relic Agent"
	DynatraceAgentName = "Dynatrace Agent"
	DataDogAgentName   = "Datadog Agent"
)

const (
	NewRelicAgentEnv                 = "NEW_RELIC_CONFIG_FILE"
	DynatraceDynamizerEnv            = "DT_DYNAMIZER_TARGET_EXE"
	DynatraceDynamizerExeSubString   = "oneagentdynamizer"
	DynatraceFullStackEnvValuePrefix = "/dynatrace/"
	DataDogAgentEnv                  = "DD_TRACE_AGENT_URL"
)

var OtherAgentEnvs = map[string]string{
	NewRelicAgentEnv:      NewRelicAgentName,
	DynatraceDynamizerEnv: DynatraceAgentName,
	DataDogAgentEnv:       DataDogAgentName,
}

var OtherAgentCmdSubString = map[string]string{
	"newrelic.jar": NewRelicAgentName,
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

// ProcFilePath constructs the file path for a given process ID and file name in the /proc filesystem.
func ProcFilePath(pid int, fileName string) string {
	return fmt.Sprintf("%s/%d/%s", procDir, pid, fileName)
}

func (pcx *ProcessContext) GetExeFile() (ProcessFile, error) {
	if pcx.exeFile == nil {
		path := ProcFilePath(pcx.ProcessID, "exe")
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
		mapsPath := ProcFilePath(pcx.ProcessID, "maps")
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
func FindAllProcesses(predicate func(int) bool) ([]int, error) {
	dirs, err := os.ReadDir(procDir)
	if err != nil {
		return nil, err
	}

	var result []int
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
		if predicate != nil && !predicate(pid) {
			continue
		}

		result = append(result, pid)
	}
	return result, nil
}

// Group processes by a key returned by the predicate function.
// The predicate function should return the key and a boolean indicating whether to include the process in the result.
// If the boolean is false, the process will be skipped.
// The function returns a map where the keys are the keys returned by the predicate function
// and the values are maps of process IDs that belong to each key.
func Group[K comparable](predicate func(int) (K, bool)) (map[K]map[int]struct{}, error) {
	if predicate == nil {
		return nil, errors.New("predicate must be provided for grouping")
	}

	dirs, err := os.ReadDir(procDir)
	if err != nil {
		return nil, err
	}

	result := make(map[K]map[int]struct{})
	for _, di := range dirs {
		if !di.IsDir() {
			continue
		}

		dirName := di.Name()

		pid, isProcessDirectory := isDirectoryPid(dirName)
		if !isProcessDirectory {
			continue
		}

		k, ok := predicate(pid)
		if !ok {
			continue
		}

		_, keyExists := result[k]
		if !keyExists {
			result[k] = make(map[int]struct{})
		}

		result[k][pid] = struct{}{}
	}

	return result, nil
}

func GetPidDetails(pid int, runtimeDetectionEnvs map[string]struct{}) Details {
	exePath := getExePath(pid)
	cmdLine := getCommandLine(pid)
	envVars := getRelevantEnvVars(pid, runtimeDetectionEnvs)
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
	exeFileName := ProcFilePath(pid, "exe")
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
	cmdLineFileName := ProcFilePath(pid, "cmdline")
	fileContent, err := os.ReadFile(cmdLineFileName)
	if err != nil {
		// Ignore errors
		return ""
	} else {
		return string(fileContent)
	}
}

func getRelevantEnvVars(pid int, runtimeDetectionEnvs map[string]struct{}) ProcessEnvs {
	envFileName := ProcFilePath(pid, "environ")
	fileContent, err := os.ReadFile(envFileName)
	if err != nil {
		// TODO: if we fail to read the environment variables, we should probably return an error
		// which will cause the process to be skipped and not instrumented?
		return ProcessEnvs{}
	}

	r := bufio.NewReader(strings.NewReader(string(fileContent)))

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

		envName := envParts[0]
		envDetectionValue := envParts[1]

		if runtimeDetectionEnvs != nil {
			if _, ok := runtimeDetectionEnvs[envName]; ok {
				overWriteEnvsResult[envName] = envDetectionValue
			}
		}

		if _, ok := LangsVersionEnvs[envName]; ok {
			detailedEnvsResult[envName] = envDetectionValue
		}

		if _, ok := OtherAgentEnvs[envName]; ok {
			detailedEnvsResult[envName] = envDetectionValue
		}
	}

	envs := ProcessEnvs{
		OverwriteEnvs: overWriteEnvsResult,
		DetailedEnvs:  detailedEnvsResult,
	}

	return envs
}

func isSecureExecutionMode(pid int) (bool, error) {
	path := ProcFilePath(pid, "auxv")
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
