package consts

import "errors"

const (
	LangDetectionContainerAnnotationKey = "keyval.dev/lang-detection-pod"
	LangDetectorContainer               = "keyval/lang-detector"
	LangDetectionEnvVar                 = "LANG_DETECTION_VERSION"
	DefaultLangDetectionVersion         = "v0.0.249"
	CurrentNamespaceEnvVar              = "CURRENT_NS"
	DefaultNamespace                    = "odigos-system"
	LangDetectorServiceAccount          = "odigos-lang-detector"
	OTLPPort                            = 4317
)

var (
	PodsNotFoundErr = errors.New("could not find a ready pod")
)
