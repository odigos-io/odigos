package config

import "testing"

func TestChronosphere_getCompanyNameFromURL(t *testing.T) {
	tt := []struct {
		_       struct{}
		URL     string
		Company string
	}{
		{
			URL:     "demo-sandbox",
			Company: "demo-sandbox",
		},
		{
			URL:     "my-company.chronosphere.io",
			Company: "my-company",
		},
		{
			URL:     "my-company.chronosphere.io:443",
			Company: "my-company",
		},
		{
			URL:     "my-company.chronosphere.io:443/",
			Company: "my-company",
		},
	}

	c := &Chronosphere{}
	for _, tc := range tt {
		t.Run(tc.URL, func(t *testing.T) {
			company := c.getCompanyNameFromURL(tc.URL)
			if company != tc.Company {
				t.Errorf("expected company %s, got %s", tc.Company, company)
			}
		})
	}
}
