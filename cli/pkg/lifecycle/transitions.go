package lifecycle

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/cli/pkg/kube"
)

type Transition interface {
	From() State
	To() State
	Execute(ctx context.Context, obj client.Object, templateSpec *v1.PodTemplateSpec) error
	Init(client *kube.Client)
}

type BaseTransition struct {
	client *kube.Client
}

func (b *BaseTransition) Init(client *kube.Client) {
	b.client = client
}

func (b *BaseTransition) log(str string) {
	fmt.Printf("    > %s\n", str)
}

var allTransitions = []Transition{
	&PreflightCheck{},
	&RequestLangDetection{},
	&WaitForLangDetection{},
}
