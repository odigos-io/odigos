package common

import (
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetCollectorsGroupDeployedConditionsPatch(err error) string {

	status := metav1.ConditionTrue
	if err != nil {
		status = metav1.ConditionFalse
	}

	message := "Gateway collector is deployed in the cluster"
	if err != nil {
		message = err.Error()
	}

	reason := "GatewayDeployedCreatedSuccessfully"
	if err != nil {
		// in the future, we can be more specific and break it down to
		// more detailed reasons about what exactly failed
		reason = "GatewayDeployedCreationFailed"
	}

	patch := map[string]interface{}{
		"status": map[string]interface{}{
			"conditions": []metav1.Condition{{
				Type:               "Deployed",
				Status:             status,
				Reason:             reason,
				Message:            message,
				LastTransitionTime: metav1.NewTime(time.Now()),
			}},
		},
	}

	patchData, _ := json.Marshal(patch)
	// marshal error is ignored as it is not expected to happen
	return string(patchData)
}
