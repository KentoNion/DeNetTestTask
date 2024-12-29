package logger

import (
	"YoutHubBot/internal/config"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func MustInitLogger(cfg *config.Config) *slog.Logger {
	var logFile *os.File
	var err error
	if cfg.Log.FilePath != "" { //Если строка в конфиге пустая, это будет означать что нам не нужно сохранение логов в файл
		logFile, err = os.OpenFile(cfg.Log.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("error opening file:", err)
		}
	}

	var log *slog.Logger

	switch cfg.Env {
	case envLocal:
		if cfg.Log.FilePath == "" {
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			return log
		}
		log = slog.New(slog.NewTextHandler(io.MultiWriter(os.Stdout, logFile), &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		if cfg.Log.FilePath == "" {
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
			return log
		}
		log = slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stdout, logFile), &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	if cfg.Log.FilePath != "" {
		log.Info(fmt.Sprintf("Logs are saving to: %s", cfg.Log.FilePath))
	}
	return log
}
