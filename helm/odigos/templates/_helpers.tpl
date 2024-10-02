{{- define "utils.certManagerApiVersion" -}}
{{- if .Capabilities.APIVersions.Has "cert-manager.io/v1" -}}
cert-manager.io/v1
{{- else if .Capabilities.APIVersions.Has "cert-manager.io/v1beta1" -}}
cert-manager.io/v1beta1
{{- else if .Capabilities.APIVersions.Has "cert-manager.io/v1alpha2" -}}
cert-manager.io/v1alpha2
{{- else if .Capabilities.APIVersions.Has "certmanager.k8s.io/v1alpha1" -}}
certmanager.k8s.io/v1alpha1
{{- else -}}
{{- print "" -}}
{{- end -}}
{{- end -}}