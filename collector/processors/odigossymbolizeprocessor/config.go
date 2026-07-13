package odigossymbolizeprocessor

// Config configures odigossymbolizeprocessor.
//
// The processor needs no platform binding: it symbolizes native frames of the
// process identified by each ResourceProfiles' process.pid resource attribute,
// reading /proc/<pid>/maps and the on-disk ELF. The same processor runs in the
// k8s node collector and the VM agent collector. All fields are optional and
// default to sane values — symbolization works out of the box with `{}`.
type Config struct {
	// PIDAttribute is the resource attribute carrying the process id.
	// Defaults to "process.pid".
	PIDAttribute string `mapstructure:"pid_attribute"`

	// MaxSymbolCache caps cached parsed binaries by count (LRU). 0 = default.
	MaxSymbolCache int `mapstructure:"max_symbol_cache"`
	// MaxSymbolBytes caps the total bytes of parsed symbols held across all cached
	// binaries (LRU eviction by memory — the real guard at scale). 0 = default.
	MaxSymbolBytes int64 `mapstructure:"max_symbol_bytes"`
	// MaxMapsCache caps cached per-pid /proc maps (LRU). 0 = default.
	MaxMapsCache int `mapstructure:"max_maps_cache"`
	// MapsTTLSeconds is how long a cached /proc/<pid>/maps is reused. 0 = default.
	MapsTTLSeconds int `mapstructure:"maps_ttl_seconds"`
	// ParseWorkers is the number of background ELF-parse workers. 0 = default.
	ParseWorkers int `mapstructure:"parse_workers"`
}

// Validate checks configuration. All fields are optional.
func (c *Config) Validate() error { return nil }

func (c *Config) pidAttribute() string {
	if c.PIDAttribute == "" {
		return "process.pid"
	}
	return c.PIDAttribute
}
