package controllers

import (
	"context"
	"errors"
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	"github.com/keyval-dev/odigos/common/consts"
	"github.com/keyval-dev/odigos/instrumentor/patch"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	IgnoredNamespaces = []string{"kube-system", "local-path-storage", consts.DefaultNamespace}
	SkipAnnotation    = "odigos.io/skip"
)

func shouldSkip(annotations map[string]string, namespace string) bool {
	for k, v := range annotations {
		if k == SkipAnnotation && v == "true" {
			return true
		}
	}

	for _, ns := range IgnoredNamespaces {
		if namespace == ns {
			return true
		}
	}

	return false
}

func syncInstrumentedApps(ctx context.Context, req *ctrl.Request, c client.Client, scheme *runtime.Scheme,
	readyReplicas int32, object client.Object, podTemplateSpec *v1.PodTemplateSpec, ownerKey string) error {

	logger := log.FromContext(ctx)
	instApps, err := getInstrumentedApps(ctx, req, c, ownerKey)
	if err != nil {
		logger.Error(err, "error finding InstrumentedApp objects")
		return err
	}

	if len(instApps.Items) == 0 {
		if readyReplicas == 0 {
			logger.V(0).Info("not enough ready replicas, waiting for pods to be ready")
			return nil
		}

		instrumentedApp := odigosv1.InstrumentedApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
			},
		}

		err = ctrl.SetControllerReference(object, &instrumentedApp, scheme)
		if err != nil {
			logger.Error(err, "error creating InstrumentedApp object")
			return err
		}

		err = c.Create(ctx, &instrumentedApp)
		if err != nil {
			logger.Error(err, "error creating InstrumentedApp object")
			return err
		}

		instrumentedApp.Status = odigosv1.InstrumentedApplicationStatus{
			LangDetection: odigosv1.LangDetectionStatus{
				Phase: odigosv1.PendingLangDetectionPhase,
			},
		}
		err = c.Status().Update(ctx, &instrumentedApp)
		if err != nil {
			logger.Error(err, "error creating InstrumentedApp object")
		}

		return nil
	}

	if len(instApps.Items) > 1 {
		return errors.New("found more than one InstrumentedApp")
	}

	// If lang not detected yet - nothing to do
	instApp := instApps.Items[0]
	if len(instApp.Spec.Languages) == 0 || instApp.Status.LangDetection.Phase != odigosv1.CompletedLangDetectionPhase {
		return nil
	}

	// if scheduled
	if instApp.Spec.CollectorAddr != "" {
		// Compute .status.instrumented field
		instrumneted, err := patch.IsInstrumented(podTemplateSpec, &instApp)
		if err != nil {
			logger.Error(err, "error computing instrumented status")
			return err
		}
		if instrumneted != instApp.Status.Instrumented {
			logger.V(0).Info("updating .status.instrumented", "instrumented", instrumneted)
			instApp.Status.Instrumented = instrumneted
			err = c.Status().Update(ctx, &instApp)
			if err != nil {
				logger.Error(err, "error computing instrumented status")
				return err
			}
		}

		// If not instrumented - patch deployment
		if !instrumneted {
			err = patch.ModifyObject(podTemplateSpec, &instApp)
			if err != nil {
				logger.Error(err, "error patching deployment / statefulset")
				return err
			}

			err = c.Update(ctx, object)
			if err != nil {
				logger.Error(err, "error instrumenting application")
				return err
			}
		}
	}

	return nil
}

func getInstrumentedApps(ctx context.Context, req *ctrl.Request, c client.Client, ownerKey string) (*odigosv1.InstrumentedApplicationList, error) {
	var instrumentedApps odigosv1.InstrumentedApplicationList
	err := c.List(ctx, &instrumentedApps, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: req.Name})
	if err != nil {
		return nil, err
	}

	return &instrumentedApps, nil
}
