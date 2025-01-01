package deviceid

type K8sResourceAttributes struct {
	Namespace     string
	WorkloadKind  string
	WorkloadName  string
	PodName       string
	ContainerName string
}
