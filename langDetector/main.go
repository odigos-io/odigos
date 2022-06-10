package main

import (
	"context"
	"flag"
	"github.com/keyval-dev/odigos/langDetector/inspectors"
	"github.com/keyval-dev/odigos/langDetector/kube"
	v1 "github.com/keyval-dev/odigos/langDetector/kube/apis/v1"
	"github.com/keyval-dev/odigos/langDetector/process"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"strings"
)

type Args struct {
	InstrumentedApp string
	Namespace       string
	PodUID          string
	ContainerNames  []string
}

func main() {
	args := parseArgs()
	var containerResults []v1.LanguageByContainer
	for _, containerName := range args.ContainerNames {
		processes, err := process.FindAllInContainer(args.PodUID, containerName)
		if err != nil {
			log.Fatalf("could not find processes, error: %s\n", err)
		}

		processResults, processName := inspectors.DetectLanguage(processes)
		log.Printf("detection result: %s\n", processResults)

		if len(processResults) > 0 {
			containerResults = append(containerResults, v1.LanguageByContainer{
				ContainerName: containerName,
				Language:      processResults[0],
				ProcessName:   processName,
			})
		}

	}

	err := publishDetectionResult(args, containerResults)
	if err != nil {
		log.Fatalf("could not publish detection result, error: %s\n", err)
	}
}

func parseArgs() *Args {
	result := Args{}
	var names string
	flag.StringVar(&result.InstrumentedApp, "instrumented-app", "", "The name of the InstrumentApp object to update with detection result")
	flag.StringVar(&result.PodUID, "pod-uid", "", "The UID of the target pod")
	flag.StringVar(&result.Namespace, "namespace", "", "The current namespace")
	flag.StringVar(&names, "container-names", "", "The container names in the target pod")
	flag.Parse()

	result.ContainerNames = strings.Split(names, ",")

	return &result
}

func publishDetectionResult(args *Args, result []v1.LanguageByContainer) error {
	client, err := kube.CreateClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	app, err := client.InstrumentedApps(args.Namespace).Get(ctx, args.InstrumentedApp, metav1.GetOptions{})
	if err != nil {
		return err
	}

	app.Spec.Languages = result
	app, err = client.InstrumentedApps(args.Namespace).Update(ctx, app, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	app.Status.LangDetection.Phase = v1.CompletedLangDetectionPhase
	_, err = client.InstrumentedApps(args.Namespace).UpdateStatus(ctx, app, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}
