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
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/odigos"
)

func DescribeOdigosEndpoint() string {
	return fmt.Sprintf("http://localhost:%s/describe/odigos", DefaultLocalPort)
}

func GetNumberOfDestinations(ctx context.Context) (int, error) {
	url, err := url.Parse(DescribeOdigosEndpoint())
	if err != nil {
		return 0, err
	}

	req := http.Request{
		Method: http.MethodGet,
		URL:    url,
		Header: http.Header{"Accept": []string{"application/json"}},
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

	// Parse to odigos.OdigosAnalyze
	var odigosAnalyze odigos.OdigosAnalyze
	if err := json.Unmarshal(respBody, &odigosAnalyze); err != nil {
		return 0, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return odigosAnalyze.NumberOfDestinations, nil
}

func DescribeOdigos(ctx context.Context) (*odigos.OdigosAnalyze, error) {
	url, err := url.Parse(DescribeOdigosEndpoint())
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

	var odigosAnalyze odigos.OdigosAnalyze
	if err := json.Unmarshal(respBody, &odigosAnalyze); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &odigosAnalyze, nil
}

func GetDescribeSourceEndpoint(workloadKind string, workloadNs string, workloadName string) string {
	return fmt.Sprintf("http://localhost:%s/describe/source/namespace/%s/kind/%s/name/%s", DefaultLocalPort, workloadNs, strings.ToLower(workloadKind), workloadName)
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

func GetSourceEndpoint(workloadKind string, workloadNs string, workloadName string) string {
	return fmt.Sprintf("http://localhost:%s/source/namespace/%s/kind/%s/name/%s", DefaultLocalPort, workloadNs, strings.ToLower(workloadKind), workloadName)
}

func CreateSource(ctx context.Context, client *kube.Client, odigosNs string, workloadKind string, workloadNs string, workloadName string) error {
	url, err := url.Parse(GetSourceEndpoint(workloadKind, workloadNs, workloadName))
	if err != nil {
		return err
	}
	req := http.Request{
		Method: http.MethodPost,
		URL:    url,
		Header: http.Header{"Accept": []string{"application/json"}},
	}

	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}
