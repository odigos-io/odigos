package containers

import "fmt"

var ImagePrefix string

func GetImageName(name string, version string) string {
	if ImagePrefix == "" {
		return fmt.Sprintf("%s:%s", name, version)
	}

	// if ImagePrefix has a trailing slash, remove it
	if ImagePrefix[len(ImagePrefix)-1] == '/' {
		ImagePrefix = ImagePrefix[:len(ImagePrefix)-1]
	}

	return fmt.Sprintf("%s/%s:%s", ImagePrefix, name, version)
}
