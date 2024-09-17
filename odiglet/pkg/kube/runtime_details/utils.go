package runtime_details

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Givin a pod, this function calculates it's workload generation.
// e.g. it will return the generation of the deployment/ds/ss manifest that created the pod.
// This function uses clientset and not the controller-runtime client, so not to pull these objects into the cache,
// potentially increasing the memory usage of the process.
// Avoid calling this function too often, as it will make an API call to the k8s API server.
func GetPodGeneration(ctx context.Context, kubeClient kubernetes.Clientset, pod *corev1.Pod) (int64, error) {

	for _, owner := range pod.ObjectMeta.OwnerReferences {
		switch owner.Kind {
		case "ReplicaSet":
			rs, err := kubeClient.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
			if err != nil {
				return 0, err
			}

			// if the replica set is created by a deployment, it will have the "deployment.kubernetes.io/revision" annotation
			// and we can use that to get the deployment generation
			deploymentRevision, found := rs.Annotations["deployment.kubernetes.io/revision"]
			if !found {
				// if the replica set is not owned by a deployment, this rs is of no interest to us
				return 0, nil
			}

			for _, owner := range rs.ObjectMeta.OwnerReferences {
				switch owner.Kind {
				case "Deployment":
					deployment, err := kubeClient.AppsV1().Deployments(pod.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
					if err != nil {
						return 0, err
					}
					if deployment.Annotations["deployment.kubernetes.io/revision"] != deploymentRevision {
						// the rs is not for the most up to date deployment revision
						// thus the pod is not relevant for runtime details
						return 0, nil
					}
					if deployment.Status.ObservedGeneration == 0 {
						// we have "deployment.kubernetes.io/revision" annotation, but the deployment has no observed generation
						// the observed generation should update along with the revision
						return 0, errors.New("deployment with revision has no observed generation")
					}
					// the observed generation is the generation that created the relevant replica set
					return deployment.Status.ObservedGeneration, nil
				}
			}
			return 0, errors.New("replica set with deployment revision has no deployment owner")
		}
		// case "DaemonSet":
		// 	ds, err := kubeClient.AppsV1().DaemonSets(pod.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
		// 	if err != nil {
		// 		return 0, err
		// 	}
		// 	return *ds.ObjectMeta.Generation, nil

		// case "StatefulSet":
		// 	ss, err := kubeClient.AppsV1().StatefulSets(pod.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
		// 	if err != nil {
		// 		return 0, err
		// 	}
		// 	return *ss.ObjectMeta.Generation, nil
		// }
	}

	return 0, nil
}
