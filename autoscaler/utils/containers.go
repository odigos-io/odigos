package utils

import "fmt"

var ImagePrefix string

func GetContainerImage(origName string) string {
	if ImagePrefix == "" {
		return origName
	}

	// if ImagePrefix has a trailing slash, remove it
	if ImagePrefix[len(ImagePrefix)-1] == '/' {
		ImagePrefix = ImagePrefix[:len(ImagePrefix)-1]
	}

	return fmt.Sprintf("%s/%s", ImagePrefix, origName)
}
