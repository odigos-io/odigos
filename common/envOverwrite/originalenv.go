package envOverwrite

// this type is used to store the original values of the environment variables the user has set
// for a specific deployment manifest.
// In k8s we need to store such map for each container in the pod.
// In systemd, this can store the original values of the systemd service.
//
// When we want to rollback the changes we made to the environment variables, we can fetch the original
// values from this map and set them back to the manifest.
//
// The key is the environment variable name.
// The value is either the original value of the environment variable or nil if the environment variable
// was not set by the user.
type OriginalEnv map[string]*string
