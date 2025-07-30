package logger

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

func parseEnv(cfg *Config) error {
	err := env.Parse(cfg)
	if err != nil {
		return fmt.Errorf("env.Parse: %w", err)
	}
	return nil
}
