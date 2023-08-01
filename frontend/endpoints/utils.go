package endpoints

import (
	"path"

	"github.com/keyval-dev/odigos/common/consts"
)

const cdnUrl = "https://d15jtxgb40qetw.cloudfront.net"

func GetImageURL(image string) string {
	return path.Join(cdnUrl, image)
}

func IsSystemNamespace(namespace string) bool {
	return namespace == "kube-system" ||
		namespace == consts.DefaultNamespace ||
		namespace == "local-path-storage" ||
		namespace == "istio-system" ||
		namespace == "linkerd"
}
