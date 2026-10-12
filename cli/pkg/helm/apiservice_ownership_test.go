package helm

import (
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"
)

// v1beta1.custom.metrics.k8s.io is a cluster-scoped singleton that Prometheus Adapter and KEDA
// also register. helm/odigos/templates/instrumentor/apiservice.yaml decides whether Odigos may
// claim that name, and it is the only one of the four code paths over this object that creates
// it; the other three (metricshandler.RegisterCustomMetricsAPI,
// metricshandler.APIServiceDeleteMigration and the CLI uninstall) only read or delete it, and all
// three recognise Odigos ownership through the service the APIService points at.
//
// The two halves of the handover have to stay complementary: the chart renders over an existing
// object only when it is Odigos-owned AND already Helm-managed, and APIServiceDeleteMigration
// deletes exactly the Odigos-owned objects that are NOT yet Helm-managed so the next upgrade can
// create them. Anything else belongs to a third party, and claiming it fails `helm install` with
// an ownership error and `helm upgrade` with "no APIService ... found".

const (
	hcmChartDir          = "../../../helm/odigos"
	hcmReleaseName       = "odigos"
	hcmReleaseNamespace  = "odigos-system"
	hcmAPIServicePath    = "odigos/templates/instrumentor/apiservice.yaml"
	hcmInstrumentorSvc   = "odigos/templates/instrumentor/service.yaml"
	hcmWebhooksPath      = "odigos/templates/instrumentor/webhooks.yaml"
	hcmAPIServiceKind    = "APIService"
	hcmAPIServiceVersion = "apiregistration.k8s.io/v1"
	// Spelled out rather than taken from k8sconsts so that renaming the constant cannot quietly
	// move both sides of the assertions below onto a different object.
	hcmManagedByLabel = "app.kubernetes.io/managed-by"
	hcmHelmManaged    = "Helm"
)

var hcmAPIServiceResource = schema.GroupVersionResource{
	Group:    "apiregistration.k8s.io",
	Version:  "v1",
	Resource: "apiservices",
}

// hcmExisting describes the v1beta1.custom.metrics.k8s.io APIService the chart's `lookup` finds.
// Every field is a pointer so that "absent" and "present but empty" stay distinguishable - the
// gate reads both through `dig`, which collapses them, and that is worth pinning.
type hcmExisting struct {
	serviceName *string
	managedBy   *string
	caBundle    *string
}

func hcmStr(s string) *string { return &s }

func (e hcmExisting) object() *unstructured.Unstructured {
	metadata := map[string]interface{}{"name": k8sconsts.CustomMetricsAPIServiceName}
	if e.managedBy != nil {
		metadata["labels"] = map[string]interface{}{hcmManagedByLabel: *e.managedBy}
	}
	spec := map[string]interface{}{
		"group":   "custom.metrics.k8s.io",
		"version": "v1beta1",
	}
	if e.serviceName != nil {
		spec["service"] = map[string]interface{}{
			"name":      *e.serviceName,
			"namespace": hcmReleaseNamespace,
			"port":      int64(443),
		}
	}
	if e.caBundle != nil {
		spec["caBundle"] = *e.caBundle
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": hcmAPIServiceVersion,
		"kind":       hcmAPIServiceKind,
		"metadata":   metadata,
		"spec":       spec,
	}}
}

// hcmLookupClientProvider answers the chart's `lookup` calls from a fake dynamic client and
// records what was asked for. A client is built per request so that a template looking up a kind
// this test does not care about still gets a well-formed empty result.
type hcmLookupClientProvider struct {
	existing *unstructured.Unstructured
	clients  []*dynamicfake.FakeDynamicClient
}

func (p *hcmLookupClientProvider) GetClientFor(apiVersion, kind string) (dynamic.NamespaceableResourceInterface, bool, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, false, err
	}
	gvr := gv.WithResource(strings.ToLower(kind) + "s")

	var seeded []runtime.Object
	if p.existing != nil && p.existing.GetAPIVersion() == apiVersion && p.existing.GetKind() == kind {
		seeded = append(seeded, p.existing.DeepCopy())
	}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: kind + "List"}, seeded...)
	p.clients = append(p.clients, client)

	namespaced := kind != hcmAPIServiceKind
	return client.Resource(gvr), namespaced, nil
}

type hcmLookup struct {
	resource schema.GroupVersionResource
	name     string
}

func (p *hcmLookupClientProvider) recordedLookups() []hcmLookup {
	var out []hcmLookup
	for _, client := range p.clients {
		for _, action := range client.Actions() {
			get, ok := action.(k8stesting.GetAction)
			if !ok {
				continue
			}
			out = append(out, hcmLookup{resource: action.GetResource(), name: get.GetName()})
		}
	}
	return out
}

func hcmRenderChart(t *testing.T, namespace string, existing *unstructured.Unstructured) (map[string]string, *hcmLookupClientProvider) {
	t.Helper()

	chrt, err := loader.Load(hcmChartDir)
	if err != nil {
		t.Fatalf("load chart %s: %v", hcmChartDir, err)
	}
	values, err := chartutil.ToRenderValues(chrt, map[string]interface{}{}, chartutil.ReleaseOptions{
		Name:      hcmReleaseName,
		Namespace: namespace,
		IsInstall: existing == nil,
		IsUpgrade: existing != nil,
	}, chartutil.DefaultCapabilities)
	if err != nil {
		t.Fatalf("build render values: %v", err)
	}

	provider := &hcmLookupClientProvider{existing: existing}
	files, err := engine.RenderWithClientProvider(chrt, values, provider)
	if err != nil {
		t.Fatalf("render chart: %v", err)
	}
	if _, ok := files[hcmAPIServicePath]; !ok {
		t.Fatalf("chart did not render %s at all; rendered files: %d", hcmAPIServicePath, len(files))
	}
	return files, provider
}

// hcmRenderAPIService returns the rendered custom-metrics APIService, or "" when the chart
// decided not to claim the name.
func hcmRenderAPIService(t *testing.T, existing *unstructured.Unstructured) string {
	t.Helper()
	files, _ := hcmRenderChart(t, hcmReleaseNamespace, existing)
	return strings.TrimSpace(files[hcmAPIServicePath])
}

type hcmAPIServiceDoc struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Service  *hcmServiceRef `json:"service"`
		Group    string         `json:"group"`
		Version  string         `json:"version"`
		CABundle *string        `json:"caBundle"`
	} `json:"spec"`
}

// hcmServiceRef is the backend reference shape shared by an APIService's spec.service and a
// webhook's clientConfig.service.
type hcmServiceRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Port      int32  `json:"port"`
}

func hcmParseAPIService(t *testing.T, rendered string) hcmAPIServiceDoc {
	t.Helper()
	if rendered == "" {
		t.Fatal("expected a rendered APIService, got nothing")
	}
	var doc hcmAPIServiceDoc
	if err := yaml.Unmarshal([]byte(rendered), &doc); err != nil {
		t.Fatalf("parse rendered APIService: %v\n%s", err, rendered)
	}
	return doc
}

func hcmParseUnstructured(t *testing.T, rendered string) *unstructured.Unstructured {
	t.Helper()
	obj := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(rendered), &obj); err != nil {
		t.Fatalf("parse rendered object: %v\n%s", err, rendered)
	}
	return &unstructured.Unstructured{Object: obj}
}

// TestCustomMetricsAPIServiceIsClaimedOnlyWhenOdigosAlreadyOwnsIt pins the decision the chart
// makes about an existing v1beta1.custom.metrics.k8s.io. The prometheus-adapter-installed-by-Helm
// row is the one that regressed: app.kubernetes.io/managed-by=Helm is a label Helm puts on every
// resource of every release, so it alone says nothing about whose release owns the object.
func TestCustomMetricsAPIServiceIsClaimedOnlyWhenOdigosAlreadyOwnsIt(t *testing.T) {
	tests := []struct {
		name         string
		existing     *hcmExisting
		wantRendered bool
	}{
		{
			name:         "fresh install, nothing in the cluster",
			existing:     nil,
			wantRendered: true,
		},
		{
			name:         "odigos instrumentor service, already owned by this helm release",
			existing:     &hcmExisting{serviceName: hcmStr("odigos-instrumentor"), managedBy: hcmStr(hcmHelmManaged)},
			wantRendered: true,
		},
		{
			name:         "odigos autoscaler service, already owned by this helm release",
			existing:     &hcmExisting{serviceName: hcmStr("odigos-autoscaler"), managedBy: hcmStr(hcmHelmManaged)},
			wantRendered: true,
		},
		{
			name:         "leftover odigos instrumentor service created by the go controller",
			existing:     &hcmExisting{serviceName: hcmStr("odigos-instrumentor")},
			wantRendered: false,
		},
		{
			name:         "leftover odigos autoscaler service created by the go controller",
			existing:     &hcmExisting{serviceName: hcmStr("odigos-autoscaler")},
			wantRendered: false,
		},
		{
			name:         "odigos service with an empty managed-by label",
			existing:     &hcmExisting{serviceName: hcmStr("odigos-instrumentor"), managedBy: hcmStr("")},
			wantRendered: false,
		},
		{
			name:         "prometheus adapter installed by its own helm chart",
			existing:     &hcmExisting{serviceName: hcmStr("prometheus-adapter"), managedBy: hcmStr(hcmHelmManaged)},
			wantRendered: false,
		},
		{
			name:         "prometheus adapter installed from plain manifests",
			existing:     &hcmExisting{serviceName: hcmStr("prometheus-adapter")},
			wantRendered: false,
		},
		{
			name:         "keda metrics apiserver installed by its own helm chart",
			existing:     &hcmExisting{serviceName: hcmStr("keda-operator-metrics-apiserver"), managedBy: hcmStr(hcmHelmManaged)},
			wantRendered: false,
		},
		{
			name:         "an apiservice with no spec.service at all",
			existing:     &hcmExisting{managedBy: hcmStr(hcmHelmManaged)},
			wantRendered: false,
		},
		{
			name:         "an apiservice whose spec.service.name is empty",
			existing:     &hcmExisting{serviceName: hcmStr(""), managedBy: hcmStr(hcmHelmManaged)},
			wantRendered: false,
		},
		{
			name:         "managed-by label in the wrong case",
			existing:     &hcmExisting{serviceName: hcmStr("odigos-instrumentor"), managedBy: hcmStr("helm")},
			wantRendered: false,
		},
		{
			name:         "managed-by label with a trailing space",
			existing:     &hcmExisting{serviceName: hcmStr("odigos-instrumentor"), managedBy: hcmStr("Helm ")},
			wantRendered: false,
		},
		{
			name:         "a service name that only starts with an odigos service name",
			existing:     &hcmExisting{serviceName: hcmStr("odigos-instrumentor-webhook"), managedBy: hcmStr(hcmHelmManaged)},
			wantRendered: false,
		},
		{
			name:         "an odigos service name in the wrong case",
			existing:     &hcmExisting{serviceName: hcmStr("Odigos-Instrumentor"), managedBy: hcmStr(hcmHelmManaged)},
			wantRendered: false,
		},
		{
			name:         "a different real odigos service",
			existing:     &hcmExisting{serviceName: hcmStr("odigos-gateway"), managedBy: hcmStr(hcmHelmManaged)},
			wantRendered: false,
		},
	}

	claimed, skipped := 0, 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var existing *unstructured.Unstructured
			if tt.existing != nil {
				existing = tt.existing.object()
			}
			rendered := hcmRenderAPIService(t, existing)

			if !tt.wantRendered {
				if rendered != "" {
					t.Fatalf("chart claimed v1beta1.custom.metrics.k8s.io it does not own:\n%s", rendered)
				}
				return
			}
			doc := hcmParseAPIService(t, rendered)
			if doc.Kind != hcmAPIServiceKind || doc.Metadata.Name != k8sconsts.CustomMetricsAPIServiceName {
				t.Fatalf("rendered %s/%s, want %s/%s", doc.Kind, doc.Metadata.Name,
					hcmAPIServiceKind, k8sconsts.CustomMetricsAPIServiceName)
			}
		})
		if tt.wantRendered {
			claimed++
		} else {
			skipped++
		}
	}

	// A gate that is stuck open or stuck shut must not be able to pass this table.
	if claimed == 0 || skipped == 0 {
		t.Fatalf("the table must exercise both outcomes, got %d claimed and %d skipped", claimed, skipped)
	}
}

// TestCustomMetricsAPIServiceOwnershipAcceptsExactlyTheOdigosServiceNames derives the set of
// spec.service.name values the chart treats as Odigos-owned instead of restating the template's
// literals, so it fails both when the template stops accepting a name the Go side still writes
// and when it starts accepting one the Go side would call foreign.
// The Go side is IsOwnedByOdigos in instrumentor/controllers/metricshandler/helpers.go, which
// switches on exactly these two k8sconsts.
func TestCustomMetricsAPIServiceOwnershipAcceptsExactlyTheOdigosServiceNames(t *testing.T) {
	want := map[string]bool{
		k8sconsts.AutoScalerWebhookServiceName: true,
		k8sconsts.InstrumentorServiceName:      true,
	}
	candidates := []string{
		k8sconsts.AutoScalerWebhookServiceName,
		k8sconsts.InstrumentorServiceName,
		k8sconsts.OdigosClusterCollectorServiceName,
		k8sconsts.OdigletLocalTrafficServiceName,
		k8sconsts.OdigosCacheServiceName,
		"prometheus-adapter",
		"keda-operator-metrics-apiserver",
		"custom-metrics-apiserver",
	}

	// Several k8sconsts names alias onto the same string; a collision here would silently shrink
	// the probe, and an accepted name missing from it would make the comparison vacuous.
	seen := map[string]bool{}
	for _, name := range candidates {
		if name == "" || seen[name] {
			t.Fatalf("candidate service names must be distinct and non-empty, %q is not", name)
		}
		seen[name] = true
	}
	for name := range want {
		if !seen[name] {
			t.Fatalf("candidate list must probe %q", name)
		}
	}
	if len(candidates) <= len(want) {
		t.Fatalf("candidate list must contain names that are expected to be rejected")
	}

	accepted := map[string]bool{}
	for _, name := range candidates {
		existing := hcmExisting{serviceName: hcmStr(name), managedBy: hcmStr(hcmHelmManaged)}.object()
		if hcmRenderAPIService(t, existing) != "" {
			accepted[name] = true
		}
	}

	for name := range want {
		if !accepted[name] {
			t.Errorf("chart refuses to upgrade its own APIService backed by service %q", name)
		}
	}
	for name := range accepted {
		if !want[name] {
			t.Errorf("chart claims an APIService backed by foreign service %q", name)
		}
	}
}

// TestCustomMetricsAPIServiceTargetsAServiceTheChartRenders pins the rendered object against the
// rest of the chart rather than against literals: the APIService must point at the instrumentor
// Service the same release installs, on a port that Service actually exposes, and its name must
// be the <version>.<group> the aggregation layer requires. Any of those drifting leaves the
// custom metrics API registered but unreachable, which silently stops gateway autoscaling.
func TestCustomMetricsAPIServiceTargetsAServiceTheChartRenders(t *testing.T) {
	const namespace = "odigos-custom-namespace"

	files, _ := hcmRenderChart(t, namespace, nil)
	doc := hcmParseAPIService(t, strings.TrimSpace(files[hcmAPIServicePath]))

	if doc.Spec.Service == nil {
		t.Fatal("rendered APIService has no spec.service")
	}
	if doc.Metadata.Name != doc.Spec.Version+"."+doc.Spec.Group {
		t.Errorf("APIService name %q must be <spec.version>.<spec.group> = %q.%q",
			doc.Metadata.Name, doc.Spec.Version, doc.Spec.Group)
	}
	if doc.Metadata.Name != k8sconsts.CustomMetricsAPIServiceName {
		t.Errorf("APIService name = %q, want %q", doc.Metadata.Name, k8sconsts.CustomMetricsAPIServiceName)
	}

	var service struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Ports []struct {
				Name string `json:"name"`
				Port int32  `json:"port"`
			} `json:"ports"`
		} `json:"spec"`
	}
	if err := yaml.Unmarshal([]byte(files[hcmInstrumentorSvc]), &service); err != nil {
		t.Fatalf("parse rendered instrumentor Service: %v", err)
	}

	if doc.Spec.Service.Name != service.Metadata.Name {
		t.Errorf("APIService points at service %q, chart renders %q",
			doc.Spec.Service.Name, service.Metadata.Name)
	}
	if doc.Spec.Service.Namespace != namespace || service.Metadata.Namespace != namespace {
		t.Errorf("APIService/Service namespaces are %q/%q, want the release namespace %q",
			doc.Spec.Service.Namespace, service.Metadata.Namespace, namespace)
	}

	// Picking the one serving port out of several is only a real assertion while the Service
	// exposes more than one.
	if len(service.Spec.Ports) < 2 {
		t.Fatalf("instrumentor Service exposes %d port(s); the port assertion below needs a choice",
			len(service.Spec.Ports))
	}
	matched := false
	for _, port := range service.Spec.Ports {
		if port.Port == doc.Spec.Service.Port {
			matched = true
		}
	}
	if !matched {
		t.Errorf("APIService targets port %d, which the instrumentor Service does not expose: %+v",
			doc.Spec.Service.Port, service.Spec.Ports)
	}

	// The aggregated metrics API and the admission webhooks are the same TLS server inside the
	// instrumentor, and insecureSkipTLSVerify is false, so they have to agree on which port of
	// that Service terminates TLS. Only one of the Service's ports does.
	webhooks := hcmWebhookBackends(t, files[hcmWebhooksPath])
	if len(webhooks) < 2 {
		t.Fatalf("expected the chart to render several webhooks pointing at the instrumentor, got %d", len(webhooks))
	}
	for name, backend := range webhooks {
		if backend != *doc.Spec.Service {
			t.Errorf("webhook %q is served by %+v but the APIService targets %+v", name, backend, *doc.Spec.Service)
		}
	}
}

// hcmWebhookBackends collects the clientConfig.service of every webhook the chart renders,
// keyed by webhook name.
func hcmWebhookBackends(t *testing.T, rendered string) map[string]hcmServiceRef {
	t.Helper()

	backends := map[string]hcmServiceRef{}
	for _, document := range strings.Split(rendered, "\n---") {
		if strings.TrimSpace(document) == "" {
			continue
		}
		var doc struct {
			Webhooks []struct {
				Name         string `json:"name"`
				ClientConfig struct {
					Service hcmServiceRef `json:"service"`
				} `json:"clientConfig"`
			} `json:"webhooks"`
		}
		if err := yaml.Unmarshal([]byte(document), &doc); err != nil {
			t.Fatalf("parse rendered webhook configuration: %v\n%s", err, document)
		}
		for _, webhook := range doc.Webhooks {
			backends[webhook.Name] = webhook.ClientConfig.Service
		}
	}
	return backends
}

// TestCustomMetricsAPIServiceUpgradeIsStableAndKeepsTheCABundle feeds the chart's own output back
// in as the live object. A helm upgrade re-reads what the previous release created, so the chart
// has to recognise its own APIService as Odigos-owned and reproduce it unchanged - otherwise
// upgrades stop renewing it and metricshandler.APIServiceDeleteMigration, which uses the same
// ownership rule, treats it as foreign.
func TestCustomMetricsAPIServiceUpgradeIsStableAndKeepsTheCABundle(t *testing.T) {
	install := hcmRenderAPIService(t, nil)
	installed := hcmParseUnstructured(t, install)

	if doc := hcmParseAPIService(t, install); doc.Spec.CABundle != nil {
		t.Errorf("a fresh install must not render a caBundle, got %q", *doc.Spec.CABundle)
	}

	// Helm labels everything it installs, so this is what the next upgrade's lookup returns.
	installed.SetLabels(map[string]string{hcmManagedByLabel: hcmHelmManaged})
	if upgrade := hcmRenderAPIService(t, installed); upgrade != install {
		t.Errorf("upgrading over the chart's own APIService changed it:\n--- install ---\n%s\n--- upgrade ---\n%s",
			install, upgrade)
	}

	// The CA bundle is written onto the live object by metricshandler's CA sync controller, not by
	// the chart. Dropping it on upgrade breaks the aggregation layer's TLS until it is rewritten.
	const caBundle = "dGhlLW9kaWdvcy1jYS1idW5kbGU="
	withCA := hcmExisting{
		serviceName: hcmStr("odigos-instrumentor"),
		managedBy:   hcmStr(hcmHelmManaged),
		caBundle:    hcmStr(caBundle),
	}.object()
	doc := hcmParseAPIService(t, hcmRenderAPIService(t, withCA))
	if doc.Spec.CABundle == nil || *doc.Spec.CABundle != caBundle {
		t.Errorf("caBundle = %v, want it carried over from the live object (%q)", doc.Spec.CABundle, caBundle)
	}

	// An empty caBundle must stay absent rather than be rendered as "", which the API server
	// rejects as an invalid base64 PEM bundle.
	emptyCA := hcmExisting{
		serviceName: hcmStr("odigos-instrumentor"),
		managedBy:   hcmStr(hcmHelmManaged),
		caBundle:    hcmStr(""),
	}.object()
	if doc := hcmParseAPIService(t, hcmRenderAPIService(t, emptyCA)); doc.Spec.CABundle != nil {
		t.Errorf("an empty live caBundle must not be rendered, got %q", *doc.Spec.CABundle)
	}
}

// TestCustomMetricsAPIServiceLooksUpTheObjectItRenders pins the `lookup` arguments. Asking for a
// different name or a different group/version makes every cluster look like a fresh install, so
// the ownership gate above would still pass its own table while never firing in production.
func TestCustomMetricsAPIServiceLooksUpTheObjectItRenders(t *testing.T) {
	files, provider := hcmRenderChart(t, hcmReleaseNamespace, nil)
	rendered := hcmParseAPIService(t, strings.TrimSpace(files[hcmAPIServicePath]))

	want := hcmLookup{resource: hcmAPIServiceResource, name: rendered.Metadata.Name}
	lookups := provider.recordedLookups()
	found := false
	for _, lookup := range lookups {
		if lookup == want {
			found = true
		}
	}
	if !found {
		t.Errorf("chart never looked up %v before claiming it; lookups were %+v", want, lookups)
	}
	if rendered.Metadata.Name != k8sconsts.CustomMetricsAPIServiceName {
		t.Errorf("rendered APIService name = %q, want %q",
			rendered.Metadata.Name, k8sconsts.CustomMetricsAPIServiceName)
	}
}
