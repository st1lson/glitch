package config

// ProfileSource loads configuration from a chaos profile.
type ProfileSource struct {
	Name string
}

// NewProfileSource returns a ProfileSource that wraps the LoadProfile logic.
func NewProfileSource(name string) *ProfileSource {
	return &ProfileSource{Name: name}
}

// Load attempts to parse the named profile and returns it mapped to a Config struct.
// It maps the Profile's Latency and Failure directly into the Config.
func (s *ProfileSource) Load() (*Config, error) {
	if s.Name == "" {
		return nil, nil // No profile specified, skip.
	}

	profile, err := LoadProfile(s.Name)
	if err != nil {
		return nil, err
	}

	return &Config{
		Latency: profile.Latency,
		Failure: profile.Failure,
	}, nil
}
