package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
)

type args struct {
	Binary    string
	Command   string
	Arguments []string
}

func main() {
	arguments := parseArgs()
	err := requestAllocation(arguments)
	if err != nil {
		log.Fatalf("failed to request allocation: %v", err)
	}

	// Execute entrypoint
	argsWithCmd := append([]string{arguments.Command}, arguments.Arguments...)
	err = syscall.Exec(arguments.Command, argsWithCmd, os.Environ())
	if err != nil {
		log.Printf("Error exec: %s\n", err)
		return
	}
}

func parseArgs() *args {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <binary> <command (default: binary)> <args (optional)>\n", os.Args[0])
	}

	binary := os.Args[1]
	command := binary
	if len(os.Args) > 2 {
		command = os.Args[2]
	}

	var arguments []string
	if len(os.Args) > 3 {
		arguments = os.Args[3:]
	}

	return &args{
		Binary:    binary,
		Command:   command,
		Arguments: arguments,
	}
}

type allocationRequest struct {
	ExePath      string `json:"exe_path"`
	PodName      string `json:"pod_name"`
	PodNamespace string `json:"pod_namespace"`
}

func requestAllocation(arguments *args) error {
	hostIP, exists := os.LookupEnv("HOST_IP")
	if !exists {
		return fmt.Errorf("HOST_IP environment variable not set")
	}

	podName, exists := os.LookupEnv("POD_NAME")
	if !exists {
		return fmt.Errorf("POD_NAME environment variable not set")
	}

	podNamespace, exists := os.LookupEnv("POD_NAMESPACE")
	if !exists {
		return fmt.Errorf("POD_NAMESPACE environment variable not set")
	}

	req := allocationRequest{
		ExePath:      arguments.Binary,
		PodName:      podName,
		PodNamespace: podNamespace,
	}
	payload := new(bytes.Buffer)
	err := json.NewEncoder(payload).Encode(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(fmt.Sprintf("http://%s:8080/launch", hostIP), "application/json", payload)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	return nil
}
