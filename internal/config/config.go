package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Redis     RedisConfig     `yaml:"redis"`
	GitHub    GitHubConfig    `yaml:"github"`
	Log       LogConfig       `yaml:"log"`
	Inspector InspectorConfig `yaml:"inspector"`
}

type InspectorConfig struct {
	Workers int `yaml:"workers"`
}

type RedisConfig struct {
	Addr            string        `yaml:"addr"`
	Password        string        `yaml:"password"`
	DB              int           `yaml:"db"`
	TTL             time.Duration `yaml:"ttl"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
}

type GitHubConfig struct {
	Token string `yaml:"token"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

// envVarPattern -- регулярное выражение для поиска плейсхолдеров ${ENV_VAR}
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)}`)

// MustLoad читает YAML-конфиг и подставляет значения переменных окружения
func MustLoad(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Errorf("чтение файла конфигурации: %w", err))
	}

	resolved := envVarPattern.ReplaceAllStringFunc(string(data), func(match string) string {
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		val := os.Getenv(varName)
		if val == "" {
			return match
		}
		return val
	})

	var cfg Config
	if err := yaml.Unmarshal([]byte(resolved), &cfg); err != nil {
		panic(fmt.Errorf("парсинг конфигурации: %w", err))
	}

	return &cfg
}
