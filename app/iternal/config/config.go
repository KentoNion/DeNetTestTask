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

type Rewards struct {
	DailySteps      int `yaml:"10k_daily_steps"`
	wakeInTime      int `yaml:"wake_in_time"`
	SleepFor8h      int `yaml:"8h_sleep"`
	PushUps         int `yaml:"10_pushups"`
	InviteAFriend   int `yaml:"inviting_a_friend"`
	beingInvited    int `yaml:"being_invited"`
	morningExercise int `yaml:"morning_exercise"`
}

type Config struct {
	Env     string  `yaml:"env"`
	DB      DB      `yaml:"postgres_db"`
	APIKeys Rest    `yaml:"API_keys"`
	Log     Log     `yaml:"logger"`
	Rewards Rewards `yaml:"rewards"`
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
