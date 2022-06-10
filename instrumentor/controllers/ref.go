package controllers

import (
	"context"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReferencedApp struct {
	Deployment  *v1.Deployment
	StatefulSet *v1.StatefulSet
}

func (r *ReferencedApp) IsDeployment() bool {
	return r.Deployment != nil
}

func (r *ReferencedApp) IsStatefulSet() bool {
	return r.StatefulSet != nil
}

func (r *ReferencedApp) PodTemplateSpec() *corev1.PodTemplateSpec {
	if r.IsDeployment() {
		return &r.Deployment.Spec.Template
	}

	return &r.StatefulSet.Spec.Template
}

func (r *ReferencedApp) Update(c client.Client, ctx context.Context) error {
	if r.IsDeployment() {
		return c.Update(ctx, r.Deployment)
	}

	return c.Update(ctx, r.StatefulSet)
}

func ReferenceFromDeployment(dep *v1.Deployment) *ReferencedApp {
	return &ReferencedApp{
		Deployment:  dep,
		StatefulSet: nil,
	}
}

func ReferenceFromStatefulSet(ss *v1.StatefulSet) *ReferencedApp {
	return &ReferencedApp{
		Deployment:  nil,
		StatefulSet: ss,
	}
}
