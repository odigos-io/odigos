package main

import (
	"encoding/json"
	"github.com/keyval-dev/odigos/odiglet/pkg"
	"github.com/keyval-dev/odigos/odiglet/pkg/allocator"
	"github.com/keyval-dev/odigos/odiglet/pkg/containers"
	"github.com/keyval-dev/odigos/odiglet/pkg/containers/runtimes"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
	"path"
)

func main() {
	if err := log.Init(); err != nil {
		panic(err)
	}
	log.Logger.V(0).Info("Starting odiglet")

	// Load env
	if err := env.Load(); err != nil {
		log.Logger.Error(err, "Failed to load env")
		os.Exit(1)
	}

	// Init Kubernetes API client
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Logger.Error(err, "Failed to init Kubernetes API client")
		os.Exit(-1)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Logger.Error(err, "Failed to init Kubernetes API client")
		os.Exit(-1)
	}

	server := newHTTPServer(clientset)
	http.Handle("/launch", server)

	log.Logger.V(0).Info("Listening on port 8080")
	log.Logger.V(0).Error(http.ListenAndServe(":8080", nil), "Failed to start http server")
}

type httpServer struct {
	kubeClient kubernetes.Interface
}

func newHTTPServer(kubeClient kubernetes.Interface) *httpServer {
	return &httpServer{
		kubeClient: kubeClient,
	}
}

func (s *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Logger.V(0).Info("Got a new launch request")

	// verify request is POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// verify request type is application/json
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	// verify request body is not empty
	if r.Body == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// verify request body is valid json
	decoder := json.NewDecoder(r.Body)
	var req pkg.LaunchRequest
	err := decoder.Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// verify request body has exe_path and pod_name and pod_namespace
	if req.ExePath == "" || req.PodName == "" || req.PodNamespace == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify pod belongs to current node
	log.Logger.V(0).Info("Getting pod", "pod", req.PodName, "namespace", req.PodNamespace)
	pod, err := s.kubeClient.CoreV1().Pods(req.PodNamespace).Get(r.Context(), req.PodName, metav1.GetOptions{})
	if err != nil {
		log.Logger.Error(err, "Failed to get pod", "pod", req.PodName, "namespace", req.PodNamespace)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if pod.Spec.NodeName != env.Current.NodeName {
		log.Logger.Error(err, "Pod does not belong to current node", "pod", req.PodName, "namespace", req.PodNamespace)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Find container ids
	containerIds, err := containers.FindIDs(pod)
	if err != nil {
		log.Logger.Error(err, "Failed to find target containers")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, containerId := range containerIds {
		cri, err := runtimes.ByName(containerId.Runtime)
		if err != nil {
			log.Logger.Error(err, "Failed to find runtime", "runtime", containerId.Runtime)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fs, err := cri.GetFileSystemPath(containerId.ID)
		if err != nil {
			log.Logger.Error(err, "Failed to get filesystem path", "container", containerId.ID)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Logger.V(0).Info("Got filesystem path", "path", fs)
		exePath := path.Join(fs, req.ExePath)
		err = allocator.Apply(exePath)
		if err != nil {
			log.Logger.Error(err, "Failed to apply allocator", "exe", exePath)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//err = writeFileToContainer(fs, &req)
		//if err != nil {
		//	log.Logger.Error(err, "Failed to write file to container")
		//	w.WriteHeader(http.StatusInternalServerError)
		//	return
		//}
	}
}

func writeFileToContainer(containerFs string, req *pkg.LaunchRequest) error {
	rootDir := path.Join(containerFs, "workspace")
	filePath := path.Join(rootDir, "test.txt")

	//// Create file and directory if not exist
	//err := os.MkdirAll(rootDir, 0755)
	//if err != nil {
	//	return err
	//}
	//
	//// Chown
	//err = os.Chown(rootDir, req.UserID, req.UserID)
	//if err != nil {
	//	return err
	//}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString("Hello World!")
	if err != nil {
		return err
	}

	//err = os.Chown(filePath, req.UserID, req.GroupID)
	return err
}
