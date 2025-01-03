package main

import (
	"app/gates/server"
	storage "app/gates/storage/postgres"
	"app/iternal/config"
	"app/iternal/logger"
	"fmt"
	chi "github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //драйвер postgres
	goose "github.com/pressly/goose/v3"
	"net/http"
	"os"
)

func main() {

	//настройка конфига
	cfg := config.MustLoad()

	//регистрация логгера
	log := logger.MustInitLogger(cfg)

	//регистрация бд
	// получение значение DB_HOST из среды, значение среды todo: прописать значение среды в docker-compose
	dbhost := os.Getenv("DB_HOST")
	if dbhost == "" {
		dbhost = cfg.DB.Host
	}
	//Подключение к бд
	connstr := fmt.Sprintf("user=%s password=%s dbname=denet_test_task host=%s sslmode=%s", cfg.DB.User, cfg.DB.Pass, dbhost, cfg.DB.Ssl)
	conn, err := sqlx.Connect("postgres", connstr) //драйвер и имя бд
	if err != nil {
		panic(err)
	}
	db := storage.NewDB(conn, log)
	//накатываем миграцию
	//err = goose.Down(conn.DB, "./gates\\storage\\migrations")
	err = goose.Up(conn.DB, "./gates\\storage\\migrations")
	if err != nil {
		panic(err)
	}

	//Настройка роутера и запуск REST сервера
	router := chi.NewRouter()
	_ = server.NewServer(db, cfg, log, router)
	restServerAddr := cfg.Rest.Host + ":" + cfg.Rest.Port //получение адреса rest сервера из конфига
	err = http.ListenAndServe(restServerAddr, router)
	if err != nil {
		panic(err)
	}
}
