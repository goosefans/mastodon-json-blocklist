package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"strings"
)

type Config struct {
	Environment         string `default:"dev"`
	LogLevel            string `default:"info" split_words:"true"`
	TaskInterval        string `default:"5m" split_words:"true"`
	JSONUrl             string `required:"true" split_words:"true"`
	MastodonBaseURL     string `required:"true" split_words:"true"`
	MastodonAccessToken string `required:"true" split_words:"true"`
}

func (cfg *Config) IsDevEnv() bool {
	return strings.ToLower(cfg.Environment) == "dev"
}

func Load() (*Config, error) {
	_ = godotenv.Overload()
	cfg := new(Config)
	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
