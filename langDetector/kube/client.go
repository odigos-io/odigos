package kube

import (
	"context"
	v1 "github.com/keyval-dev/odigos/langDetector/kube/apis/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
)

type V1Interface interface {
	InstrumentedApps(namespace string) InstrumentedAppsInterface
}

type V1Client struct {
	restClient rest.Interface
}

func CreateClient() (*V1Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	err = v1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &v1.GroupVersion
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&crdConfig)
	if err != nil {
		return nil, err
	}

	return &V1Client{restClient: client}, nil
}

func (c *V1Client) InstrumentedApps(namespace string) InstrumentedAppsInterface {
	return &instrumentedAppClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

type InstrumentedAppsInterface interface {
	Get(ctx context.Context, name string, options metav1.GetOptions) (*v1.InstrumentedApplication, error)
	Update(ctx context.Context, instrumentedApp *v1.InstrumentedApplication, opts metav1.UpdateOptions) (*v1.InstrumentedApplication, error)
	UpdateStatus(ctx context.Context, instrumentedApp *v1.InstrumentedApplication, opts metav1.UpdateOptions) (*v1.InstrumentedApplication, error)
}

type instrumentedAppClient struct {
	restClient rest.Interface
	ns         string
}

func (i *instrumentedAppClient) Get(ctx context.Context, name string, options metav1.GetOptions) (*v1.InstrumentedApplication, error) {
	result := v1.InstrumentedApplication{}
	err := i.restClient.
		Get().
		Namespace(i.ns).
		Resource("instrumentedapplications").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (i *instrumentedAppClient) Update(ctx context.Context, instrumentedApp *v1.InstrumentedApplication, opts metav1.UpdateOptions) (*v1.InstrumentedApplication, error) {
	result := v1.InstrumentedApplication{}
	err := i.restClient.
		Put().
		Namespace(i.ns).
		Resource("instrumentedapplications").
		Name(instrumentedApp.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(instrumentedApp).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (i *instrumentedAppClient) UpdateStatus(ctx context.Context, instrumentedApp *v1.InstrumentedApplication, opts metav1.UpdateOptions) (*v1.InstrumentedApplication, error) {
	result := v1.InstrumentedApplication{}
	err := i.restClient.
		Put().
		Namespace(i.ns).
		Resource("instrumentedapplications").
		Name(instrumentedApp.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(instrumentedApp).
		Do(ctx).
		Into(&result)

	return &result, err
}
