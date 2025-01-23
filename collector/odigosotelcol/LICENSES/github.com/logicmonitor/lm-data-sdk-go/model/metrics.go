package model

type MetricsInput struct {
	Resource   ResourceInput
	Datasource DatasourceInput
	Instance   InstanceInput
	DataPoint  DataPointInput
}

type MetricPayload struct {
	ResourceName          string            `json:"resourceName"`
	ResourceDescription   string            `json:"resourceDescription,omitempty"`
	ResourceID            map[string]string `json:"resourceIds"`
	ResourceProperties    map[string]string `json:"resourceProperties,omitempty"`
	DataSourceName        string            `json:"dataSource"`
	DataSourceDisplayName string            `json:"dataSourceDisplayName,omitempty"`
	DataSourceGroup       string            `json:"dataSourceGroup,omitempty"`
	DataSourceID          int               `json:"dataSourceId"`
	Instances             []Instance        `json:"instances"`

	IsCreate bool `json:"-"`
}

type Instance struct {
	InstanceName        string            `json:"instanceName"`
	InstanceID          int               `json:"instanceId"`
	InstanceDisplayName string            `json:"instanceDisplayName,omitempty"`
	InstanceGroup       string            `json:"instanceGroup,omitempty"`
	InstanceProperties  map[string]string `json:"instanceProperties,omitempty"`
	DataPoints          []DataPoint       `json:"dataPoints"`
}

type DataPoint struct {
	DataPointName            string            `json:"dataPointName"`
	DataPointType            string            `json:"dataPointType"`
	DataPointDescription     string            `json:"dataPointDescription,omitempty"`
	DataPointAggregationType string            `json:"dataPointAggregationType"`
	Value                    map[string]string `json:"values"`
}

type ResourceInput struct {
	ResourceName        string
	ResourceDescription string
	ResourceID          map[string]string
	ResourceProperties  map[string]string
	IsCreate            bool
}

type DatasourceInput struct {
	DataSourceName        string
	DataSourceDisplayName string
	DataSourceGroup       string
	DataSourceID          int
}

type InstanceInput struct {
	InstanceName        string
	InstanceID          int
	InstanceDisplayName string
	InstanceGroup       string
	InstanceProperties  map[string]string
}

type DataPointInput struct {
	DataPointName            string
	DataPointType            string
	DataPointDescription     string
	DataPointAggregationType string
	Value                    map[string]string
}
