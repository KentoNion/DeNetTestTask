package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
)

type DB struct {
	User string `yaml:"user" env-required:"true"`
	Pass string `yaml:"password" env-required:"true"`
	Host string `yaml:"host" env-required:"true"`
	Port string `yaml:"port"`
	Ssl  string `yaml:"sslmode" env-required:"true"`
}

type Rest struct {
	Host string `yaml:"host" env-required:"true"`
	Port string `yaml:"port" env-required:"true"`
}

type Log struct {
	FilePath string `yaml:"logger_file_path"`
}

type Config struct {
	Env     string         `yaml:"env"`
	DB      DB             `yaml:"postgres_db"`
	Rest    Rest           `yaml:"RestServer"`
	Log     Log            `yaml:"logger"`
	Rewards map[string]int `yaml:"rewards"` // Ключ — название награды, значение — очки
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "../config.yaml"
	}

	//проверка существует ли файл
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatal("cannot read config file")
	}

	var cfg Config

	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		log.Fatal(err)
	}
	return &cfg
}
