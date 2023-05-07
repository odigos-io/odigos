package kube

import (
	"context"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/inspectors"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/keyval-dev/odigos/odiglet/pkg/process"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

func runtimeInspection(pods []corev1.Pod) ([]common.LanguageByContainer, error) {
	resultsMap := make(map[string]common.LanguageByContainer)
	for _, pod := range pods {
		for _, c := range pod.Spec.Containers {
			processes, err := process.FindAllInContainer(string(pod.UID), c.Name)
			if err != nil {
				log.Logger.Error(err, "Failed to find processes")
				return nil, err
			}

			processResults, processName := inspectors.DetectLanguage(processes)
			if len(processResults) > 0 {
				resultsMap[c.Name] = common.LanguageByContainer{
					ContainerName: c.Name,
					Language:      processResults[0],
					ProcessName:   processName,
				}
			}
		}
	}

	results := make([]common.LanguageByContainer, 0, len(resultsMap))
	for _, value := range resultsMap {
		results = append(results, value)
	}

	return results, nil
}

func getRuntimeObjectName(name string, kind string) string {
	return strings.ToLower(kind + "-" + name)
}

func persistRuntimeResults(ctx context.Context, results []common.LanguageByContainer, owner client.Object, kubeClient client.Client, scheme *runtime.Scheme) error {
	updatedIa := &odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getRuntimeObjectName(owner.GetName(), owner.GetObjectKind().GroupVersionKind().Kind),
			Namespace: owner.GetNamespace(),
		},
	}

	err := controllerutil.SetOwnerReference(owner, updatedIa, scheme)
	if err != nil {
		log.Logger.Error(err, "Failed to set owner reference")
		return err
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, kubeClient, updatedIa, func() error {
		updatedIa.Spec.Languages = results
		return nil
	})

	log.Logger.V(0).Info("updated runtime info", "result", operationResult)
	return nil
}
