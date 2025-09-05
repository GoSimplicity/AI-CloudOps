package apply

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/openapi/cached"
	"k8s.io/kubectl/pkg/cmd/util"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	diskcached "k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	openapi2 "k8s.io/client-go/openapi"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubectl/pkg/util/openapi"
	"k8s.io/kubectl/pkg/validation"
)

type BuilderOptions struct {
	//Unstructured  bool
	Validate      string
	Namespace     string
	LabelSelector string
	FieldSelector string
	All           bool
	AllNamespaces bool
}

func NewBuilderOptions() *BuilderOptions {
	return &BuilderOptions{
		//Unstructured: true,
		Validate: metav1.FieldValidationWarn,
	}
}

//var addToSchemeOnce sync.Once

type GetterFactory struct {
	config         *rest.Config
	ctx            context.Context
	warningPrinter *printers.WarningPrinter
}

func NewGetterFactory(ctx context.Context, config *rest.Config) *GetterFactory {
	factory := &GetterFactory{
		ctx:            ctx,
		config:         config,
		warningPrinter: printers.NewWarningPrinter(os.Stderr, printers.WarningPrinterOptions{Color: true}),
	}

	return factory
}

var overlyCautiousIllegalFileCharacters = regexp.MustCompile(`[^(\w/\.)]`)

func (f *GetterFactory) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	config.Burst = 100
	defaultHTTPCacheDir := filepath.Join(homedir.HomeDir(), ".kube", "http-cache")

	// 构造缓存目录
	parentDir := filepath.Join(homedir.HomeDir(), ".kube", "cache", "discovery")
	schemeHost := strings.Replace(strings.Replace(config.Host, "https://", "", 1), "http://", "", 1)
	safeHost := overlyCautiousIllegalFileCharacters.ReplaceAllString(schemeHost, "_")
	discoveryCacheDir := filepath.Join(parentDir, safeHost)

	return diskcached.NewCachedDiscoveryClientForConfig(config, discoveryCacheDir, defaultHTTPCacheDir, 10*time.Minute)
}

func (f *GetterFactory) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := f.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient, func(a string) {
		if f.warningPrinter != nil {
			f.warningPrinter.Print(a)
		}
	})
	return expander, nil
}

func (f *GetterFactory) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewDefaultClientConfig(
		clientcmdapi.Config{},
		&clientcmd.ConfigOverrides{},
	)
}

func (f *GetterFactory) ToRESTConfig() (*rest.Config, error) {
	if f.config == nil {
		return nil, errors.New("rest config is nil")
	}
	if f.config.GroupVersion == nil {
		f.config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	}
	if f.config.APIPath == "" {
		f.config.APIPath = "/api"
	}
	if f.config.NegotiatedSerializer == nil {
		f.config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	}

	rest.SetKubernetesDefaults(f.config)
	return f.config, nil
}

func (f *GetterFactory) DynamicClient() (dynamic.Interface, error) {
	config, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}

func (f *GetterFactory) KubernetesClientSet() (*kubernetes.Clientset, error) {
	config, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func (f *GetterFactory) NewBuilder() *resource.Builder {
	return resource.NewBuilder(f)
}

// OpenAPISchema 实现一次性初始化并缓存 openapi.Resources。
func (f *GetterFactory) OpenAPISchema() (openapi.Resources, error) {
	restCfg, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	openAPIGetter := openapi.NewOpenAPIGetter(discoveryClient)
	doc, err := openAPIGetter.OpenAPISchema()
	if err != nil {
		return nil, err
	}
	return openapi.NewOpenAPIData(doc)
}
func (f *GetterFactory) configForMapping(mapping *meta.RESTMapping) (*rest.Config, error) {
	config, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	cfg := rest.CopyConfig(config)

	gvk := mapping.GroupVersionKind
	if gvk.Group == corev1.GroupName {
		cfg.APIPath = "/api"
	} else {
		cfg.APIPath = "/apis"
	}
	gv := gvk.GroupVersion()
	cfg.GroupVersion = &gv
	return cfg, nil
}

func (f *GetterFactory) ClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	factory, err := f.configForMapping(mapping)
	if err != nil {
		return nil, err
	}
	return rest.RESTClientFor(factory)
}

func (f *GetterFactory) UnstructuredClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	cfg, err := f.configForMapping(mapping)
	if err != nil {
		return nil, err
	}
	cfg.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	return rest.RESTClientFor(cfg)
}

func (f *GetterFactory) RESTClient() (*rest.RESTClient, error) {
	config, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return rest.RESTClientFor(config)
}

// Validator  validationDirective "strict"/"warn"/"ignore"
func (f *GetterFactory) Validator(validationDirective string) (validation.Schema, error) {
	if validationDirective == "ignore" {
		return validation.NullSchema{}, nil
	}
	return validation.ConjunctiveSchema{
		validation.NewSchemaValidation(f),
		validation.NoDoubleKeySchema{},
	}, nil
}

func (f *GetterFactory) OpenAPIV3Client() (openapi2.Client, error) {
	d, err := f.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return cached.NewClient(d.OpenAPIV3()), nil
}

var _ genericclioptions.RESTClientGetter = &GetterFactory{}
var _ util.Factory = &GetterFactory{}
