package model

type UpdatePropertiesPayload struct {
	ResourceName          string            `json:"resourceName"`
	ResourceID            map[string]string `json:"resourceIds"`
	ResourceProperties    map[string]string `json:"resourceProperties"`
	DataSourceName        string            `json:"dataSource"`
	DataSourceDisplayName string            `json:"dataSourceDisplayName,omitempty"`
	InstanceName          string            `json:"instanceName"`
	InstanceProperties    map[string]string `json:"instanceProperties"`
}
