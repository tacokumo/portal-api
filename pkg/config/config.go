package config

import (
	"os"

	"github.com/cockroachdb/errors"
)

type Config struct {
	PortalName string
}

func LoadFromEnv() *Config {
	return &Config{
		PortalName: os.Getenv("PORTAL_NAME"),
	}
}

func (c *Config) Validate() error {
	if c.PortalName == "" {
		return errors.New("PORTAL_NAME environment variable is required")
	}
	return nil
}
