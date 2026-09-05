package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// NewViper returns a viper instance configured for CODEAGENT_* environment
// variables and an optional config file.
//
// Search order when configFile is empty:
//   - $HOME/.codeagent/config.(yaml|yml|json|toml|...)
func NewViper(configFile string) (*viper.Viper, error) {
	v := viper.New()
	v.SetEnvPrefix("CODEAGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	if strings.TrimSpace(configFile) != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, err
		}
		return v, nil
	}

	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return v, nil
	}

	v.SetConfigName("config")
	v.AddConfigPath(filepath.Join(home, ".codeagent"))
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return v, nil
		}
		return nil, err
	}

	return v, nil
}
