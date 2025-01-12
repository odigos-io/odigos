package profile

import (
	"github.com/odigos-io/odigos/common"
	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Profile struct {
	ProfileName      common.ProfileName
	MinimumTier      common.OdigosTier
	ShortDescription string
	KubeObject       K8sObject                         // used to read it from the embedded YAML file
	Dependencies     []common.ProfileName              // other profiles that are applied by the current profile
	ModifyConfigFunc func(*common.OdigosConfiguration) // function to update the configuration based on the profile
}

type K8sObject interface {
	metav1.Object
	runtime.Object
}
