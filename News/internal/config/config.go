package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log/slog"
	"os"
)

const defaultConfigPath = "./News.yaml"

// NewsConfig - конфигурация сервиса APIGateway
type NewsConfig struct {
	Env         string `yaml:"env" env-default:"prod"`
	StoragePath string `yaml:"storage_path" env-default:"./News.db"`
	HTTPServer  `yaml:"http_server" env-required:"true"`
	RSS         `yaml:"rss" env-required:"true"`
}

// HTTPServer - конфигурация Web сервера
type HTTPServer struct {
	Address string `yaml:"address" env-required:"true"`
}

// RSS -конфигурация RSS
type RSS struct {
	URLS          []string `yaml:"urls" env-required:"true"`
	RequestPeriod int      `yaml:"request_period" env-default:"5"`
}

// MustLoadNewsConfig - Загружаем конфигурацию из файла
func MustLoadNewsConfig(log *slog.Logger) *NewsConfig {

	// Получаем путь к файлу конфига
	configPath := os.Getenv("NEWS_CONFIG_PATH")
	if configPath == "" {
		log.Warn("NEWS_CONFIG_PATH environment variable not set. using default", "configPath", defaultConfigPath)
		configPath = defaultConfigPath
	}

	// Проверяем существование файла конфигурации
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Error("config file does not exist", "configPath", configPath)
		os.Exit(1)
	}

	var cfg NewsConfig

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Error("error reading config file", "err", err)
		os.Exit(1)
	}

	return &cfg
}
