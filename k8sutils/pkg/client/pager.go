package client

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DefaultPageSize = 500

type listFunc[T metav1.ListInterface] func(context.Context, metav1.ListOptions) (T, error)

func ListWithPages[T metav1.ListInterface](pageSize int, list listFunc[T], ctx context.Context,
	opts *metav1.ListOptions, handler func(obj T) error) error {
	if opts == nil {
		opts = &metav1.ListOptions{}
	}

	opts.Limit = int64(pageSize)
	opts.Continue = ""
	for {
		obj, err := list(ctx, *opts)
		if err != nil {
			return err
		}
		if err := handler(obj); err != nil {
			return err
		}
		if obj.GetContinue() == "" {
			break
		}
		opts.Continue = obj.GetContinue()
	}
	return nil
}
