package k8scmprovider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

const schemeName = "k8scm"

// NewFactory returns a new ConfigMap provider factory
func NewFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(newProvider)
}

type provider struct {
	clientset                   *kubernetes.Clientset
	namespace                   string
	cmName                      string
	key                         string
	informer                    cache.SharedIndexInformer
	informerStopCh              chan struct{}
	providerStopCh              chan struct{}
	running                     bool
	logger                      *zap.Logger
	lastReportedResourceVersion string
}

func newProvider(set confmap.ProviderSettings) confmap.Provider {
	return &provider{
		logger: set.Logger,
	}
}

func (p *provider) Retrieve(ctx context.Context, uri string, wf confmap.WatcherFunc) (*confmap.Retrieved, error) {
	// Parse URI: k8scm:namespace/name/key
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	path := strings.TrimPrefix(uri, schemeName+":")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid k8scm uri format, expected k8scm:namespace/name/key, got: %s", uri)
	}

	p.namespace = parts[0]
	p.cmName = parts[1]
	p.key = parts[2]

	// Initialize k8s client if not already done
	if p.clientset == nil {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
		p.clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create k8s client: %w", err)
		}
	}

	var cm *corev1.ConfigMap
	retryErr := retry.OnError(retry.DefaultRetry, func(err error) bool {
		return true
	}, func() error {
		var err error
		cm, err = p.clientset.CoreV1().ConfigMaps(p.namespace).Get(ctx, p.cmName, metav1.GetOptions{})
		return err
	})

	if retryErr != nil {
		return nil, fmt.Errorf("failed to get configmap: %w", retryErr)
	}

	if cm == nil {
		return nil, errors.New("nil configmap returned")
	}

	// Start informer if not running
	if !p.running && wf != nil {
		p.running = true
		p.informerStopCh = make(chan struct{})
		p.providerStopCh = make(chan struct{})
		go p.runInformer(wf)
	}

	content, ok := cm.Data[p.key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in configmap", p.key)
	}

	p.logger.Info("configuration retrieved from ConfigMap", zap.String("name", cm.Name), zap.String("namespace", cm.Namespace))

	return confmap.NewRetrievedFromYAML([]byte(content))
}

func (p *provider) runInformer(wf confmap.WatcherFunc) {
	fieldSelector := fields.OneTermEqualSelector("metadata.name", p.cmName).String()

	p.informer = cache.NewSharedIndexInformerWithOptions(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fieldSelector
				return p.clientset.CoreV1().ConfigMaps(p.namespace).List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fieldSelector
				return p.clientset.CoreV1().ConfigMaps(p.namespace).Watch(context.Background(), options)
			},
		},
		&corev1.ConfigMap{},
		cache.SharedIndexInformerOptions{},
	)

	p.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			cm, ok := newObj.(*corev1.ConfigMap)
			if !ok {
				wf(&confmap.ChangeEvent{
					Error: fmt.Errorf("expected *corev1.ConfigMap, got %T", newObj),
				})
				return
			}

			if cm.Name != p.cmName {
				wf(&confmap.ChangeEvent{
					Error: fmt.Errorf("expected configmap name %q, got %q", p.cmName, cm.Name),
				})
				return
			}

			if cm.ResourceVersion != p.lastReportedResourceVersion {
				p.lastReportedResourceVersion = cm.ResourceVersion
				wf(&confmap.ChangeEvent{})
			}
		},
	})

	p.informer.Run(p.informerStopCh)
	close(p.providerStopCh)
}

func (p *provider) Scheme() string {
	return schemeName
}

func (p *provider) Shutdown(ctx context.Context) error {
	p.logger.Info("shutting down k8s config map provider")
	if p.running {
		close(p.informerStopCh)
		<-p.providerStopCh
		p.running = false
	}
	p.logger.Info("k8s config map provider shut down successfully")
	return nil
}
