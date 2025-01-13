package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log/slog"
	"os"
)

const defaultConfigPath = "./Gateway.yaml"

// GatewayConfig - конфигурация сервиса APIGateway
type GatewayConfig struct {
	Env            string `yaml:"env" env-default:"prod"`
	HTTPServer     `yaml:"http_server" env-required:"true"`
	NewsServer     string `yaml:"news_server" env-required:"true"`
	CommentsServer string `yaml:"comments_server" env-required:"true"`
}

// HTTPServer - конфигурация Web сервера
type HTTPServer struct {
	Address string `yaml:"address" env-required:"true"`
}

// MustLoadGatewayConfig - Загружаем конфигурацию из файла
func MustLoadGatewayConfig(log *slog.Logger) *GatewayConfig {

	// Получаем путь к файлу конфига
	configPath := os.Getenv("GATEWAY_CONFIG_PATH")
	if configPath == "" {
		log.Warn("GATEWAY_CONFIG_PATH environment variable not set. using default", "configPath", defaultConfigPath)
		configPath = defaultConfigPath
	}

	// Проверяем существование файла конфигурации
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Error("config file does not exist", "configPath", configPath)
		os.Exit(1)
	}

	var cfg GatewayConfig

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Error("error reading config file", "err", err)
		os.Exit(1)
	}

	return &cfg
}
