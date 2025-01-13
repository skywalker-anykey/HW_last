package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log/slog"
	"os"
)

const defaultConfigPath = "./Censor.yaml"

// CensorConfig - конфигурация сервиса APIGateway
type CensorConfig struct {
	Env        string `yaml:"env" env-default:"prod"`
	HTTPServer `yaml:"http_server" env-required:"true"`
	BadList    []string `yaml:"bad_list"`
}

// HTTPServer - конфигурация Web сервера
type HTTPServer struct {
	Address string `yaml:"address" env-required:"true"`
}

// MustLoadCensorConfig - Загружаем конфигурацию из файла
func MustLoadCensorConfig(log *slog.Logger) *CensorConfig {

	// Получаем путь к файлу конфига
	configPath := os.Getenv("CENSOR_CONFIG_PATH")
	if configPath == "" {
		log.Warn("CENSOR_CONFIG_PATH environment variable not set. using default", "configPath", defaultConfigPath)
		configPath = defaultConfigPath
	}

	// Проверяем существование файла конфигурации
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Error("config file does not exist", "configPath", configPath)
		os.Exit(1)
	}

	var cfg CensorConfig

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Error("error reading config file", "err", err)
		os.Exit(1)
	}

	return &cfg
}
