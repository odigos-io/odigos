package diagnose_util

import (
	"context"
	"fmt"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"sync"
)

const (
	CRDName         = "crdName"
	CRDGroup        = "crdGroup"
	actionGroupName = "actions.odigos.io"
	odigosGroupName = "odigos.io"
)

var CRDsList = []map[string]string{
	{
		CRDName:  "addclusterinfos",
		CRDGroup: actionGroupName,
	},
	{
		CRDName:  "deleteattributes",
		CRDGroup: actionGroupName,
	},
	{
		CRDName:  "renameattributes",
		CRDGroup: actionGroupName,
	},
	{
		CRDName:  "probabilisticsamplers",
		CRDGroup: actionGroupName,
	},
	{
		CRDName:  "piimaskings",
		CRDGroup: actionGroupName,
	},
	{
		CRDName:  "latencysamplers",
		CRDGroup: actionGroupName,
	},
	{
		CRDName:  "errorsamplers",
		CRDGroup: actionGroupName,
	},
	{
		CRDName:  "instrumentedapplications",
		CRDGroup: odigosGroupName,
	},
	{
		CRDName:  "instrumentationconfigs",
		CRDGroup: odigosGroupName,
	},
	{
		CRDName:  "instrumentationrules",
		CRDGroup: odigosGroupName,
	},
	{
		CRDName:  "instrumentationinstances",
		CRDGroup: odigosGroupName,
	},
}

func FetchOdigosCRs(ctx context.Context, kubeClient *kube.Client, crdDir string) error {
	var wg sync.WaitGroup

	for _, resourceData := range CRDsList {
		crdDataDirPath := filepath.Join(crdDir, resourceData[CRDName])
		err := os.Mkdir(crdDataDirPath, os.ModePerm) // os.ModePerm gives full permissions (0777)
		if err != nil {
			fmt.Printf("Error creating directory for CRD: %v, because: %v", resourceData, err)
			continue
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			err = fetchSingleResource(ctx, kubeClient, crdDataDirPath, resourceData)
			if err != nil {
				fmt.Printf("Error Getting CRDs of: %v, because: %v\n", resourceData[CRDName], err)
			}
		}()
	}

	wg.Wait()

	return nil
}

func fetchSingleResource(ctx context.Context, kubeClient *kube.Client, crdDataDirPath string, resourceData map[string]string) error {
	fmt.Printf("Fetching Resource: %s\n", resourceData[CRDName])

	gvr := schema.GroupVersionResource{
		Group:    resourceData[CRDGroup], // The API group
		Version:  "v1alpha1",             // The version of the resourceData
		Resource: resourceData[CRDName],  // The resourceData type
	}

	err := client.ListWithPages(client.DefaultPageSize, kubeClient.Dynamic.Resource(gvr).List, ctx, metav1.ListOptions{}, func(crds *unstructured.UnstructuredList) error {
		for _, crd := range crds.Items {
			if err := saveCrdToFile(crd, crdDataDirPath, crd.GetName()); err != nil {
				fmt.Printf("Fetching Resource %s Failed because: %s\n", resourceData[CRDName], err)
			}
		}
		return nil
	},
	)

	if err != nil {
		return err
	}

	return nil
}

func saveCrdToFile(crd interface{}, crdDataDirPath string, crdName string) error {
	crdDirPath := filepath.Join(crdDataDirPath, crdName+".yaml")
	crdFile, err := os.OpenFile(crdDirPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer crdFile.Close()

	crdYAML, err := yaml.Marshal(crd)
	if err != nil {
		return err
	}

	_, err = crdFile.Write(crdYAML)
	if err != nil {
		return err
	}
	if err = crdFile.Sync(); err != nil {
		return err
	}

	return nil
}

func FetchDestinationsCRDs(ctx context.Context, client *kube.Client, CRDsDir string) error {
	odigosNamespace, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return err
	}

	destinations, err := client.OdigosClient.Destinations(odigosNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	crdDestinationPath := filepath.Join(CRDsDir, "destinations")
	err = os.Mkdir(crdDestinationPath, os.ModePerm)

	for _, destination := range destinations.Items {
		if err := saveCrdToFile(destination, crdDestinationPath, destination.Name); err != nil {
			fmt.Printf("Fetching Resource %s Failed because: %s\n", destination.Name, err)
		}

	}

	return nil
}
