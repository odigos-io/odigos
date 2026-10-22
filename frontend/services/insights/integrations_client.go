package insights

import (
	"context"
	"net/http"
	"strconv"
)

func (c *Client) ListIntegrationCatalog(ctx context.Context) ([]IntegrationDescriptor, error) {
	return doList[IntegrationDescriptor](ctx, c, http.MethodGet, c.apiEndpoint("integrations", "catalog"), nil)
}
func (c *Client) ListIntegrations(ctx context.Context) ([]Integration, error) {
	return doList[Integration](ctx, c, http.MethodGet, c.apiEndpoint("integrations"), nil)
}
func (c *Client) GetIntegration(ctx context.Context, id string) (*Integration, error) {
	var result Integration
	if err := c.do(ctx, http.MethodGet, c.apiEndpoint("integrations", id), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) CreateIntegration(ctx context.Context, input IntegrationWrite) (*Integration, error) {
	var result Integration
	if err := c.do(ctx, http.MethodPost, c.apiEndpoint("integrations"), input, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) UpdateIntegration(ctx context.Context, id string, input IntegrationWrite) (*Integration, error) {
	var result Integration
	if err := c.do(ctx, http.MethodPut, c.apiEndpoint("integrations", id), input, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) DeleteIntegration(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, c.apiEndpoint("integrations", id), nil, nil)
}
func (c *Client) TestIntegration(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, c.apiEndpoint("integrations", id, "test"), nil, nil)
}
func (c *Client) RetryIntegrationDeliveries(ctx context.Context, id string, keys []string) error {
	return c.do(ctx, http.MethodPost, c.apiEndpoint("integrations", id, "deliveries", "retry"), struct {
		FindingKeys []string `json:"finding_keys"`
	}{keys}, nil)
}
func (c *Client) ListIntegrationDeliveries(ctx context.Context, id string, state *string, limit *int) ([]Delivery, error) {
	endpoint := c.apiEndpoint("integrations", id, "deliveries")
	query := endpoint.Query()
	addOptionalString(query, "state", state)
	if limit != nil {
		query.Set("limit", strconv.Itoa(*limit))
	}
	endpoint.RawQuery = query.Encode()
	return doList[Delivery](ctx, c, http.MethodGet, endpoint, nil)
}
func (c *Client) ListIntegrationAttempts(ctx context.Context, id string, limit *int) ([]DeliveryAttempt, error) {
	endpoint := c.apiEndpoint("integrations", id, "attempts")
	query := endpoint.Query()
	if limit != nil {
		query.Set("limit", strconv.Itoa(*limit))
	}
	endpoint.RawQuery = query.Encode()
	return doList[DeliveryAttempt](ctx, c, http.MethodGet, endpoint, nil)
}
func (c *Client) GetFindingDeliveries(ctx context.Context, key string) ([]Delivery, error) {
	endpoint := c.apiEndpoint("findings", "deliveries")
	query := endpoint.Query()
	query.Set("finding_key", key)
	endpoint.RawQuery = query.Encode()
	return doList[Delivery](ctx, c, http.MethodGet, endpoint, nil)
}
