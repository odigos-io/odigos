package main

import (
	"encoding/json"
	"flag"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/langDetector/inspectors"
	"github.com/keyval-dev/odigos/langDetector/process"
	"io/fs"
	"io/ioutil"
	"log"
	"strings"
)

type Args struct {
	PodUID         string
	ContainerNames []string
}

func main() {
	args := parseArgs()
	var containerResults []common.LanguageByContainer
	for _, containerName := range args.ContainerNames {
		processes, err := process.FindAllInContainer(args.PodUID, containerName)
		if err != nil {
			log.Fatalf("could not find processes, error: %s\n", err)
		}

		processResults, processName := inspectors.DetectLanguage(processes)
		log.Printf("detection result: %s\n", processResults)

		if len(processResults) > 0 {
			containerResults = append(containerResults, common.LanguageByContainer{
				ContainerName: containerName,
				Language:      processResults[0],
				ProcessName:   processName,
			})
		}

	}

	err := publishDetectionResult(containerResults)
	if err != nil {
		log.Fatalf("could not publish detection result, error: %s\n", err)
	}
}

func parseArgs() *Args {
	result := Args{}
	var names string
	flag.StringVar(&result.PodUID, "pod-uid", "", "The UID of the target pod")
	flag.StringVar(&names, "container-names", "", "The container names in the target pod")
	flag.Parse()

	result.ContainerNames = strings.Split(names, ",")

	return &result
}

func publishDetectionResult(result []common.LanguageByContainer) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return ioutil.WriteFile("/dev/detection-result", data, fs.ModePerm)
}
