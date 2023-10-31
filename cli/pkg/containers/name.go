package containers

import "fmt"

func GetImageName(imagePrefix string, name string, version string) string {
	if imagePrefix == "" {
		return fmt.Sprintf("%s:%s", name, version)
	}

	// if ImagePrefix has a trailing slash, remove it
	if imagePrefix[len(imagePrefix)-1] == '/' {
		imagePrefix = imagePrefix[:len(imagePrefix)-1]
	}

	return fmt.Sprintf("%s/%s:%s", imagePrefix, name, version)
}
