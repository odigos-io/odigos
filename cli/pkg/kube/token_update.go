package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"
)

func CreateTokenPayload(onpremToken string) (string, error) {
	tokenPayload := pro.TokenPayload{OnpremToken: onpremToken}
	jsonBytes, err := json.Marshal(tokenPayload)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func ExecuteRemoteUpdateToken(ctx context.Context, client *Client, namespace string, onPremToken string) error {
	uiSvcProxyEndpoint := fmt.Sprintf(
		"/api/v1/namespaces/%s/services/%s:%d/proxy/api/token/update",
		namespace,
		k8sconsts.OdigosUiServiceName,
		k8sconsts.OdigosUiServicePort,
	)

	tokenPayload, err := CreateTokenPayload(onPremToken)
	if err != nil {
		return fmt.Errorf("failed to create token payload: %v", err)
	}
	body := bytes.NewBuffer([]byte(tokenPayload))

	request := client.Clientset.RESTClient().Post().
		AbsPath(uiSvcProxyEndpoint).
		Body(body).
		SetHeader("Content-Type", "application/json").
		Do(ctx)

	if err := request.Error(); err != nil {
		return fmt.Errorf("failed to update token: %v", err)
	}

	return nil
}
