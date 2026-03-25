package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Log      LogConfig      `yaml:"log"`
}

type ServerConfig struct {
	Port int `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD" env-default:"postgres"`
	Name     string `yaml:"name" env:"DB_NAME" env-default:"subscriptions"`
	SSLMode  string `yaml:"sslmode" env:"DB_SSLMODE" env-default:"disable"`
}

type LogConfig struct {
	Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

func Load(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	return &cfg, nil
}
