package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"
)

//this file is used to update the token for the odigos pro secret

func createTokenPayload(onpremToken string) ([]byte, error) {
	tokenPayload := pro.TokenPayload{OnpremToken: onpremToken}
	jsonBytes, err := json.Marshal(tokenPayload)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func ExecuteRemoteUpdateToken(ctx context.Context, client *Client, namespace string, onPremToken string) error {
	uiSvcProxyEndpoint := fmt.Sprintf(
		"/api/v1/namespaces/%s/services/%s:%d/proxy/token/update",
		namespace,
		k8sconsts.OdigosUiServiceName,
		k8sconsts.OdigosUiServicePort,
	)

	tokenPayload, err := createTokenPayload(onPremToken)
	if err != nil {
		return fmt.Errorf("failed to create token payload: %w", err)
	}
	body := bytes.NewBuffer(tokenPayload)

	request := client.Clientset.RESTClient().Post().
		AbsPath(uiSvcProxyEndpoint).
		Body(body).
		SetHeader("Content-Type", "application/json")
	// The API server consumes Authorization for Kubernetes credentials; forward
	// the UI identity in its existing cookie instead.
	if token := os.Getenv(consts.UIIDTokenEnvVarName); token != "" {
		request.SetHeader("Cookie", (&http.Cookie{Name: "id_token", Value: token}).String())
	}

	if err := request.Do(ctx).Error(); err != nil {
		return fmt.Errorf("failed to update token: %w", err)
	}

	return nil
}
