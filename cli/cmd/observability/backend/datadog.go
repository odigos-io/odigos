package backend

import (
	"fmt"
	"github.com/spf13/cobra"
	"net/url"
	"strings"
)

type Datadog struct{}

func (d *Datadog) Name() string {
	return "datadog"
}

func (d *Datadog) ValidateFlags(cmd *cobra.Command) error {
	apiKey := cmd.Flag("api-key").Value.String()
	if apiKey == "" {
		return fmt.Errorf("API key required for Datadog backend, please specify --api-key")
	}

	targetUrl := cmd.Flag("url").Value.String()
	_, err := url.Parse(targetUrl)
	if err != nil {
		return fmt.Errorf("invalud url specified: %s", err)
	}

	if !strings.Contains(targetUrl, "datadog.com") {
		return fmt.Errorf("%s is not a valid datadog url", targetUrl)
	}

	return nil
}
