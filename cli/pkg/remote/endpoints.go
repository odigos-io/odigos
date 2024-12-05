package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/k8sutils/pkg/describe/source"

	"github.com/odigos-io/odigos/cli/pkg/kube"
)

func GetDestinationsEndpoint() string {
	return fmt.Sprintf("http://localhost:%s/api/destinations", DefaultLocalPort)
}

func GetNumberOfDestinations(ctx context.Context, client *kube.Client) (int, error) {
	url, err := url.Parse(GetDestinationsEndpoint())
	if err != nil {
		return 0, err
	}

	req := http.Request{
		Method: http.MethodGet,
		URL:    url,
	}

	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Parse to array of strings
	var destinations []interface{}
	if err := json.Unmarshal(respBody, &destinations); err != nil {
		return 0, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return len(destinations), nil
}

func GetDescribeSourceEndpoint(workloadKind string, workloadNs string, workloadName string) string {
	return fmt.Sprintf("http://localhost:%s/api/describe/source/namespace/%s/kind/%s/name/%s", DefaultLocalPort, workloadNs, strings.ToLower(workloadKind), workloadName)
}

func DescribeSource(ctx context.Context, client *kube.Client, odigosNs string, workloadKind string, workloadNs string, workloadName string) (*source.SourceAnalyze, error) {
	url, err := url.Parse(GetDescribeSourceEndpoint(workloadKind, workloadNs, workloadName))
	if err != nil {
		return nil, err
	}

	req := http.Request{
		Method: http.MethodGet,
		URL:    url,
		Header: http.Header{"Accept": []string{"application/json"}},
	}

	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sourceObj source.SourceAnalyze
	if err := json.Unmarshal(respBody, &sourceObj); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &sourceObj, nil
}
