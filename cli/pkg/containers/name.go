package containers

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func GetImageName(imagePrefix string, name string, version string) string {
	var fullName string
	if strings.Contains(name, "@") || strings.Contains(name, ":") {
		// don't append the tag if the image is pinned to a SHA or has a tag, for example:
		// registry.connect.redhat.com/odigos/odigos-instrumentor-ubi9@SHA26:ab312...
		fullName = name
	} else {
		fullName = fmt.Sprintf("%s:%s", name, version)
	}
	if imagePrefix == "" {
		imagePrefix = k8sconsts.OdigosImagePrefix
	}

	// if ImagePrefix has a trailing slash, remove it
	if imagePrefix[len(imagePrefix)-1] == '/' {
		imagePrefix = imagePrefix[:len(imagePrefix)-1]
	}

	return fmt.Sprintf("%s/%s", imagePrefix, fullName)
}
