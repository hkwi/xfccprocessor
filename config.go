package xfccsubjectprocessor

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	TargetAttribute string `mapstructure:"target_attribute"`
	Overwrite       bool   `mapstructure:"overwrite"`
}

var _ component.Config = (*Config)(nil)

func (c *Config) Validate() error {
	if c.TargetAttribute == "" {
		return fmt.Errorf("target_attribute must not be empty")
	}
	return nil
}
