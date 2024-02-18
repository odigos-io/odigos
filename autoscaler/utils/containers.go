package utils

import "fmt"

var ImagePrefix string

func GetCollectorContainerImage(origName string, version string) string {
	imageWithTag := fmt.Sprintf("%s:%s", origName, version)
	if ImagePrefix == "" {
		return imageWithTag
	}

	// if ImagePrefix has a trailing slash, remove it
	if ImagePrefix[len(ImagePrefix)-1] == '/' {
		ImagePrefix = ImagePrefix[:len(ImagePrefix)-1]
	}

	return fmt.Sprintf("%s/%s", ImagePrefix, imageWithTag)
}
