package testutil

import (
	"context"

	"github.com/keyval-dev/odigos/common/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteNsLabels(ctx context.Context, k8sClient client.Client, nsName string) error {
	return k8sClient.Patch(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}, client.RawPatch(types.MergePatchType, []byte(`{
		"metadata": {
			"labels": null
		}
	}`)))
}

func SetNsInstrumentationEnabled(ctx context.Context, k8sClient client.Client, nsName string) error {
	return k8sClient.Patch(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}, client.RawPatch(types.MergePatchType, []byte(`{
		"metadata": {
			"labels": {
				"`+consts.OdigosInstrumentationLabel+`": "`+consts.InstrumentationEnabled+`"
			}
		}
	}`)))
}

func SetOdigosInstrumentationEnabled[W client.Object](obj W) W {
	copy := obj.DeepCopyObject().(W)
	copy.SetLabels(map[string]string{consts.OdigosInstrumentationLabel: consts.InstrumentationEnabled})
	return copy
}

func SetOdigosInstrumentationDisabled[W client.Object](obj W) W {
	copy := obj.DeepCopyObject().(W)
	copy.SetLabels(map[string]string{consts.OdigosInstrumentationLabel: consts.InstrumentationDisabled})
	return copy
}

func DeleteOdigosInstrumentationLabel[W client.Object](obj W) W {
	copy := obj.DeepCopyObject().(W)
	delete(copy.GetLabels(), consts.OdigosInstrumentationLabel)
	return copy
}

func SetReportedNameAnnotation[W client.Object](obj W, reportedName string) W {
	copy := obj.DeepCopyObject().(W)
	copy.SetAnnotations(map[string]string{consts.OdigosReportedNameAnnotation: reportedName})
	return copy
}
