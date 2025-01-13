package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log/slog"
	"os"
)

const defaultConfigPath = "./Comments.yaml"

// CommentsConfig - конфигурация сервиса APIGateway
type CommentsConfig struct {
	Env          string `yaml:"env" env-default:"prod"`
	StoragePath  string `yaml:"storage_path" env-default:"./Comments.db"`
	HTTPServer   `yaml:"http_server" env-required:"true"`
	CensorServer string `yaml:"censor_server" env-required:"true"`
}

// HTTPServer - конфигурация Web сервера
type HTTPServer struct {
	Address string `yaml:"address" env-required:"true"`
}

// MustLoadCommentsConfig - Загружаем конфигурацию из файла
func MustLoadCommentsConfig(log *slog.Logger) *CommentsConfig {

	// Получаем путь к файлу конфига
	configPath := os.Getenv("COMMENTS_CONFIG_PATH")
	if configPath == "" {
		log.Warn("COMMENTS_CONFIG_PATH environment variable not set. using default", "configPath", defaultConfigPath)
		configPath = defaultConfigPath
	}

	// Проверяем существование файла конфигурации
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Error("config file does not exist", "configPath", configPath)
		os.Exit(1)
	}

	var cfg CommentsConfig

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Error("error reading config file", "err", err)
		os.Exit(1)
	}

	return &cfg
}
