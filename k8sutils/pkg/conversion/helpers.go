package conversion

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	actions_v1alpha1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	actions_internal "github.com/odigos-io/odigos/api/hub/actions/v__internal"
	odigos_internal "github.com/odigos-io/odigos/api/hub/odigos/v__internal"
	odigos_v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

// Scheme holds both your external and internal types + conversion funcs.
var Scheme = func() *runtime.Scheme {
	sch := runtime.NewScheme()
	utilruntime.Must(actions_v1alpha1.AddToScheme(sch))
	utilruntime.Must(actions_internal.AddToScheme(sch))
	utilruntime.Must(odigos_v1alpha1.AddToScheme(sch))
	utilruntime.Must(odigos_internal.AddToScheme(sch))
	return sch
}()

// ConvertExternalToInternal converts a versioned (“v1alpha1”) object into its internal counterpart.
//
//	external and internal must be pointers to their struct types.
//
// e.g. ConvertExternalToInternal(&actions_v1alpha1.ErrorSampler{}, &actions_internal.ErrorSampler{})
func ConvertExternalToInternal(external runtime.Object, internal runtime.Object) error {
	return Scheme.Convert(external, internal, nil)
}

// ConvertInternalToExternal converts an internal (“v__internal”) object into its versioned counterpart.
//
//	internal and external must be pointers to their struct types.
//
// e.g. ConvertInternalToExternal(&odigos_internal.Processor{}, &odigos_v1alpha1.Processor{})
func ConvertInternalToExternal(internal runtime.Object, external runtime.Object) error {
	return Scheme.Convert(internal, external, nil)
}
