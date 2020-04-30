package leap

import (
	"errors"

	"github.com/influxdata/kapacitor/listmap"
)

// Config declares the needed configuration options for the service Wfe
type Config struct {
	Enabled   bool   `toml:"enabled" override:"enabled"`
	URL       string `toml:"url" override:"url"`
	AuthToken string `toml:"token" override:"token"`
	Workflow  string `toml:"workflow" override:"workflow"`
	Workspace string `toml:"workspace"  override:"workspace"`
}

type Configs []Config

func (cs *Configs) UnmarshalTOML(data interface{}) error {
	return listmap.DoUnmarshalTOML(cs, data)
}

// NewConfig returns a blank config
func NewConfig() Config {
	return Config{}
}

// Validate checks config was specified
func (c Config) Validate() error {
	if c.Enabled && c.URL == "" {
		return errors.New("must specify wfe server URL")
	}
	return nil
}
