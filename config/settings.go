package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Settings struct {
	Host       SettingsHost    `env-prefix:"HOST_"`
	Session    SettingsSession `env-prefix:"SESSION_"`
	ConfigPath string          `env:"CONFIG_PATH" env-default:"/etc/oauth-resource-proxy/config.yaml"`
}

type SettingsHost struct {
	Address string `env:"ADDRESS"`
	Port    int    `env:"PORT" env-default:"8080"`
}

type SettingsSession struct {
	Key string `env:"KEY"`
	// Possible values: "filesystem", "redis"
	StoreDriver string `env:"STORE_DRIVER" env-default:"filesystem" env-description:"Session store driver: filesystem or redis"`
	// when redis
	Redis struct {
		Address  string `env:"ADDRESS"`
		Port     int    `env:"PORT" env-default:"6379"`
		Username string `env:"USERNAME"`
		Password string `env:"PASSWORD"`
		DB       int    `env:"DB" env-default:"0"`
		PoolSize int    `env:"POOL_SIZE" env-default:"10"`
	} `env-prefix:"REDIS_"`
	// when filesystem
	StoreDirectory string `env:"STORE_DIRECTORY"`
}

func loadSettingsFromEnv() (Settings, error) {
	var settings Settings
	err := cleanenv.ReadEnv(&settings)
	return settings, err
}

func (s Settings) GetWSAddress() string {
	return fmt.Sprintf("%s:%d", s.Host.Address, s.Host.Port)
}
