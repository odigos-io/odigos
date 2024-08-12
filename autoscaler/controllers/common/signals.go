package common

import (
	"context"
	"encoding/json"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateCollectorGroupReceiverSignals(ctx context.Context, c client.Client, cg *odigosv1.CollectorsGroup, signals []common.ObservabilitySignal) error {
	signalsStr := make([]string, 0, len(signals))
	for _, signal := range signals {
		signalsStr = append(signalsStr, string(signal))
	}

	patch := map[string]interface{}{
		"status": map[string]interface{}{
			"receiverSignals": signalsStr,
		},
	}

	patchData, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	return c.Status().Patch(ctx, cg, client.RawPatch(types.MergePatchType, patchData))
}
