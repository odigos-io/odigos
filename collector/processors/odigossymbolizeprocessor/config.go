package odigossymbolizeprocessor

// defaultServerEndpoint is the well-known unix socket of the node-local symbolize
// server (run by vm-agent / odiglet). Kept in sync with symbolizeserver.DefaultSocketPath.
const defaultServerEndpoint = "/var/odigos/symbolize.sock"

// Config configures odigossymbolizeprocessor.
//
// The processor does no binary analysis itself: it batches each profile's native
// frames and asks the node-local symbolize server (which has /proc access and owns
// the ELF engine + caches) to resolve them. The same processor runs in the k8s node
// collector and the VM agent collector. All fields are optional — it works with `{}`.
type Config struct {
	// PIDAttribute is the resource attribute carrying the process id.
	// Defaults to "process.pid".
	PIDAttribute string `mapstructure:"pid_attribute"`

	// ServerEndpoint is the symbolize server's unix socket path.
	// Defaults to /var/odigos/symbolize.sock.
	ServerEndpoint string `mapstructure:"server_endpoint"`
}

// Validate checks configuration. All fields are optional.
func (c *Config) Validate() error { return nil }

func (c *Config) pidAttribute() string {
	if c.PIDAttribute == "" {
		return "process.pid"
	}
	return c.PIDAttribute
}
