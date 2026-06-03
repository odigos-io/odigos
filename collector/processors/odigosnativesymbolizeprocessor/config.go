package odigosnativesymbolizeprocessor

// Config configures the native symbolize processor.
type Config struct {
	// Native enables on-disk native symbolization (.symtab/.dynsym, MiniDebugInfo,
	// local separate debuginfo). When false the processor is a pure pass-through:
	// native frames are forwarded raw (mapping+address) and nothing is resolved.
	Native bool `mapstructure:"native"`
}

func createDefaultConfig() *Config {
	return &Config{
		Native: true,
	}
}

func (cfg *Config) Validate() error {
	return nil
}
