package config

// Source defines an interface for providing configuration overrides.
type Source interface {
	Load() (*Config, error)
}

// Builder orchestrates the loading and merging of configuration from
// multiple sources in a Chain of Responsibility pattern.
type Builder struct {
	sources []Source
}

// NewBuilder initializes an empty Builder pipeline.
func NewBuilder() *Builder {
	return &Builder{
		sources: make([]Source, 0),
	}
}

// AddSource appends a new configuration source to the pipeline.
// Sources added later take precedence.
func (b *Builder) AddSource(s Source) {
	b.sources = append(b.sources, s)
}

// Build iterates through all sources, loads their config, and deep merges them
// into a single unified Config object.
func (b *Builder) Build() (Config, error) {
	cfg := DefaultConfig()

	for _, s := range b.sources {
		override, err := s.Load()
		if err != nil {
			return cfg, err
		}
		cfg.Merge(override)
	}

	return cfg, nil
}
