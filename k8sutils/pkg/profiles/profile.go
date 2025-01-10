package profiles

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Profile struct {
	ProfileName      common.ProfileName
	ShortDescription string
	KubeObject       Object               // used to read it from the embedded YAML file
	Dependencies     []common.ProfileName // other profiles that are applied by the current profile
}

type Object interface {
	metav1.Object
	runtime.Object
}

var (
	// sizing profiles for the collectors resource settings
	SizeSProfile = Profile{
		ProfileName:      common.ProfileName("size_s"),
		ShortDescription: "Small size deployment profile",
	}
	SizeMProfile = Profile{
		ProfileName:      common.ProfileName("size_m"),
		ShortDescription: "Medium size deployment profile",
	}
	SizeLProfile = Profile{
		ProfileName:      common.ProfileName("size_l"),
		ShortDescription: "Large size deployment profile",
	}
	AllowConcurrentAgents = Profile{
		ProfileName:      common.ProfileName("allow_concurrent_agents"),
		ShortDescription: "This profile allows Odigos to run concurrently with other agents",
	}
	FullPayloadCollectionProfile = Profile{
		ProfileName:      common.ProfileName("full-payload-collection"),
		ShortDescription: "Collect any payload from the cluster where supported with default settings",
		KubeObject:       &odigosv1alpha1.InstrumentationRule{},
	}
	DbPayloadCollectionProfile = Profile{
		ProfileName:      common.ProfileName("db-payload-collection"),
		ShortDescription: "Collect db payload from the cluster where supported with default settings",
		KubeObject:       &odigosv1alpha1.InstrumentationRule{},
	}
	QueryOperationDetector = Profile{
		ProfileName:      common.ProfileName("query-operation-detector"),
		ShortDescription: "Detect the SQL operation name from the query text",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	SemconvUpgraderProfile = Profile{
		ProfileName:      common.ProfileName("semconv"),
		ShortDescription: "Upgrade and align some attribute names to a newer version of the OpenTelemetry semantic conventions",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	CategoryAttributesProfile = Profile{
		ProfileName:      common.ProfileName("category-attributes"),
		ShortDescription: "Add category attributes to the spans",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	CopyScopeProfile = Profile{
		ProfileName:      common.ProfileName("copy-scope"),
		ShortDescription: "Copy the scope name into a separate attribute for backends that do not support scopes",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	HostnameAsPodNameProfile = Profile{
		ProfileName:      common.ProfileName("hostname-as-podname"),
		ShortDescription: "Populate the spans resource `host.name` attribute with value of `k8s.pod.name`",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	JavaNativeInstrumentationsProfile = Profile{
		ProfileName:      common.ProfileName("java-native-instrumentations"),
		ShortDescription: "Deprecated, native instrumentations are now enabled by default",
	}
	JavaEbpfInstrumentationsProfile = Profile{
		ProfileName:      common.ProfileName("java-ebpf-instrumentations"),
		ShortDescription: "Instrument Java applications using eBPF instrumentation and eBPF enterprise processing",
		KubeObject:       &odigosv1alpha1.InstrumentationRule{},
	}
	CodeAttributesProfile = Profile{
		ProfileName:      common.ProfileName("code-attributes"),
		ShortDescription: "Record span attributes in 'code' namespace where supported",
	}
	DisableNameProcessorProfile = Profile{
		ProfileName:      common.ProfileName("disable-name-processor"),
		ShortDescription: "If not using dotnet or java native instrumentations, disable the name processor which is not needed",
	}
	SmallBatchesProfile = Profile{
		ProfileName:      common.ProfileName("small-batches"),
		ShortDescription: "Reduce the batch size for exports",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	KratosProfile = Profile{
		ProfileName:      common.ProfileName("kratos"),
		ShortDescription: "Bundle profile that includes db-payload-collection, semconv, category-attributes, copy-scope, hostname-as-podname, code-attributes, query-operation-detector, disableNameProcessorProfile, small-batches, size_m, allow_concurrent_agents",
		Dependencies:     []common.ProfileName{"db-payload-collection", "semconv", "category-attributes", "copy-scope", "hostname-as-podname", "code-attributes", "query-operation-detector", "disableNameProcessorProfile", "small-batches", "size_m", "allow_concurrent_agents"},
	}
	ProfilesMap = map[common.ProfileName]Profile{
		SizeSProfile.ProfileName:                      SizeSProfile,
		SizeMProfile.ProfileName:                      SizeMProfile,
		SizeLProfile.ProfileName:                      SizeLProfile,
		FullPayloadCollectionProfile.ProfileName:      FullPayloadCollectionProfile,
		DbPayloadCollectionProfile.ProfileName:        DbPayloadCollectionProfile,
		QueryOperationDetector.ProfileName:            QueryOperationDetector,
		SemconvUpgraderProfile.ProfileName:            SemconvUpgraderProfile,
		CategoryAttributesProfile.ProfileName:         CategoryAttributesProfile,
		CopyScopeProfile.ProfileName:                  CopyScopeProfile,
		HostnameAsPodNameProfile.ProfileName:          HostnameAsPodNameProfile,
		JavaNativeInstrumentationsProfile.ProfileName: JavaNativeInstrumentationsProfile,
		JavaEbpfInstrumentationsProfile.ProfileName:   JavaEbpfInstrumentationsProfile,
		CodeAttributesProfile.ProfileName:             CodeAttributesProfile,
		DisableNameProcessorProfile.ProfileName:       DisableNameProcessorProfile,
		SmallBatchesProfile.ProfileName:               SmallBatchesProfile,
		KratosProfile.ProfileName:                     KratosProfile,
		AllowConcurrentAgents.ProfileName:             AllowConcurrentAgents,
	}
)
