package pkg

type LaunchRequest struct {
	ExePath      string `json:"exe_path"`
	PodName      string `json:"pod_name"`
	PodNamespace string `json:"pod_namespace"`
}
