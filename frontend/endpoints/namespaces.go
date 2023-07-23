package endpoints

import (
	"context"
	"log"
	"net/http"

	corev1 "k8s.io/api/core/v1"

	"github.com/keyval-dev/odigos/common/consts"

	"github.com/keyval-dev/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"
)

type GetNamespacesResponse struct {
	Namespaces []GetNamespaceItem `json:"namespaces"`
}

type GetNamespaceItem struct {
	Name      string `json:"name"`
	Selected  bool   `json:"selected"`
	TotalApps int    `json:"totalApps"`
}

func GetNamespaces(c *gin.Context) {
	list, err := kube.DefaultClient.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	appsPerNamespace, err := CountAppsPerNamespace(c)
	if err != nil {
		returnError(c, err)
		return
	}

	var response GetNamespacesResponse
	for _, namespace := range list.Items {

		// check if entire namespace is instrumented
		selected := namespace.Labels[consts.OdigosInstrumentationLabel] == consts.InstrumentationEnabled

		response.Namespaces = append(response.Namespaces, GetNamespaceItem{
			Name:      namespace.Name,
			Selected:  selected,
			TotalApps: appsPerNamespace[namespace.Name],
		})
	}

	c.JSON(http.StatusOK, response)
}

type PersistNamespaceItem struct {
	Name           string                   `json:"name"`
	SelectedAll    bool                     `json:"selected_all"`
	FutureSelected bool                     `json:"future_selected"`
	Objects        []PersistNamespaceObject `json:"objects"`
}

type PersistNamespaceObject struct {
	Name     string          `json:"name"`
	Kind     ApplicationKind `json:"kind"`
	Selected bool            `json:"selected"`
}

func PersistNamespaces(c *gin.Context) {
	request := make(map[string]PersistNamespaceItem)
	if err := c.ShouldBindJSON(&request); err != nil {
		returnError(c, err)
		return
	}

	namespaces, err := kube.DefaultClient.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	for _, ns := range namespaces.Items {
		labeled := isLabeled(&ns)
		userSelection, exists := request[ns.Name]
		if !exists {
			log.Printf("Namespace %s not found in request, skipping\n", ns.Name)
		} else {
			labeledByUser := userSelection.SelectedAll || userSelection.FutureSelected
			changed := false
			if labeledByUser && !labeled {
				ns.Labels[consts.OdigosInstrumentationLabel] = consts.InstrumentationEnabled
				changed = true
			} else if !labeledByUser && labeled {
				delete(ns.Labels, consts.OdigosInstrumentationLabel)
				changed = true
			}

			if changed {
				_, err = kube.DefaultClient.CoreV1().Namespaces().Update(c.Request.Context(), &ns, metav1.UpdateOptions{})
				if err != nil {
					returnError(c, err)
					return
				}
			}

			err = syncObjectsInNamespace(c.Request.Context(), &ns, userSelection.Objects)
		}
	}
}

func isLabeled(obj metav1.Object) bool {
	if val, exists := obj.GetLabels()[consts.OdigosInstrumentationLabel]; exists {
		if val == consts.InstrumentationEnabled {
			return true
		}
	}

	return false
}

func syncObjectsInNamespace(ctx context.Context, ns *corev1.Namespace, objects []PersistNamespaceObject) error {
	deps, err := kube.DefaultClient.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, dep := range deps.Items {
		labeled := isLabeled(&dep)
		userSelection, exists := findObject(objects, dep.Name, ApplicationKindDeployment)
		if !exists {
			log.Printf("Deployment %s not found in request, skipping\n", dep.Name)
		} else {
			labeledByUser := userSelection.Selected
			changed := false
			if labeledByUser && !labeled {
				if dep.Labels == nil {
					dep.Labels = make(map[string]string)
				}
				dep.Labels[consts.OdigosInstrumentationLabel] = consts.InstrumentationEnabled
				changed = true
			} else if !labeledByUser && labeled {
				delete(dep.Labels, consts.OdigosInstrumentationLabel)
				changed = true
			}

			if changed {
				_, err = kube.DefaultClient.AppsV1().Deployments(ns.Name).Update(ctx, &dep, metav1.UpdateOptions{})
				if err != nil {
					return err
				}
			}
		}
	}

	sts, err := kube.DefaultClient.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, st := range sts.Items {
		labeled := isLabeled(&st)
		userSelection, exists := findObject(objects, st.Name, ApplicationKindStatefulSet)
		if !exists {
			log.Printf("StatefulSet %s not found in request, skipping\n", st.Name)
		} else {
			labeledByUser := userSelection.Selected
			changed := false
			if labeledByUser && !labeled {
				st.Labels[consts.OdigosInstrumentationLabel] = consts.InstrumentationEnabled
				changed = true
			} else if !labeledByUser && labeled {
				delete(st.Labels, consts.OdigosInstrumentationLabel)
				changed = true
			}

			if changed {
				_, err = kube.DefaultClient.AppsV1().StatefulSets(ns.Name).Update(ctx, &st, metav1.UpdateOptions{})
				if err != nil {
					return err
				}
			}
		}
	}

	dss, err := kube.DefaultClient.AppsV1().DaemonSets(ns.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ds := range dss.Items {
		labeled := isLabeled(&ds)
		userSelection, exists := findObject(objects, ds.Name, ApplicationKindDaemonSet)
		if !exists {
			log.Printf("DaemonSet %s not found in request, skipping\n", ds.Name)
		} else {
			labeledByUser := userSelection.Selected
			changed := false
			if labeledByUser && !labeled {
				ds.Labels[consts.OdigosInstrumentationLabel] = consts.InstrumentationEnabled
				changed = true
			} else if !labeledByUser && labeled {
				delete(ds.Labels, consts.OdigosInstrumentationLabel)
				changed = true
			}

			if changed {
				_, err = kube.DefaultClient.AppsV1().DaemonSets(ns.Name).Update(ctx, &ds, metav1.UpdateOptions{})
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func findObject(objects []PersistNamespaceObject, name string, kind ApplicationKind) (*PersistNamespaceObject, bool) {
	for _, obj := range objects {
		if obj.Name == name && obj.Kind == kind {
			return &obj, true
		}
	}

	return nil, false
}

// returns a map, where the key is a namespace name and the value is the
// number of apps in this namespace (not necessarily instrumented)
func CountAppsPerNamespace(ctx context.Context) (map[string]int, error) {

	deps, err := kube.DefaultClient.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ss, err := kube.DefaultClient.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ds, err := kube.DefaultClient.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	namespaceToAppsCount := make(map[string]int)
	for _, dep := range deps.Items {
		namespaceToAppsCount[dep.Namespace]++
	}
	for _, st := range ss.Items {
		namespaceToAppsCount[st.Namespace]++
	}
	for _, d := range ds.Items {
		namespaceToAppsCount[d.Namespace]++
	}

	return namespaceToAppsCount, nil
}
