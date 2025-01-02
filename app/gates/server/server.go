package server

import (
	"app/auth"
	"app/domain"
	"app/iternal/config"
	"app/iternal/pkg"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
)

type Server struct {
	db      domain.UserStore
	context context.Context
	log     *slog.Logger
	srv     *domain.UserService
	auth    *auth.Service
}

func NewServer(db domain.UserStore, cfg *config.Config, log *slog.Logger, r *chi.Mux) *Server {
	server := &Server{ //формируем структуру сервера
		db:      db,
		context: context.Background(),
		log:     log,
		srv:     domain.NewUserService(db, log, cfg),
		auth:    auth.NewService(db, log, cfg, uuid.New().String(), pkg.NormalClock{}),
	}

	//роутим эндпоинты
	r.Method(http.MethodGet, "/login", http.HandlerFunc(server.loginHandler))
	r.Method(http.MethodPost, "/register", http.HandlerFunc(server.registerHandler))
	r.Method(http.MethodGet, "/users/{id}/status", http.HandlerFunc(server.statusHandler))
	r.Method(http.MethodGet, "/users/leaderboard", http.HandlerFunc(server.leaderbord))
	r.Method(http.MethodPatch, "/users/{id}/task/complete", http.HandlerFunc(server.taskCompleteHandler))
	r.Method(http.MethodPatch, "/users/{id}/referrer", http.HandlerFunc(server.reffererHandler))
	server.log.Info("router configured")
	return server
}

func (s Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	//логин моковый, он требует только ввести id юзера и отдаёт jwt токен
	const op = "gates.server.loginHandler"
	s.log.Info(op, ": starting login")
	var id domain.UserID
	//читаем запрос
	if err := json.NewDecoder(r.Body).Decode(&id); err != nil {
		s.log.Error(op, "failed to decode login", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	r.Body.Close()
	s.log.Debug("login: ", id)
	token, err := s.auth.Login(s.context, id)
	if err != nil {
		s.log.Error(op, ": failed to login: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.log.Info(op, ": sucessfully logged in")
	s.log.Debug("token: ", token)
	resp, err := json.Marshal(token)
	if err != nil {
		s.log.Error(op, ": failed to encode token: ", err.Error())
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (s Server) registerHandler(w http.ResponseWriter, r *http.Request) {
	const op = "gates.server.registerHandler"
	s.log.Info(op, ": starting register")
	var user user
	//декодировка json с параметрами пользователя
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		s.log.Error(op, ": failed to decode request body: "+err.Error())
		return
	}
	r.Body.Close()
	//проверка наличия никнейма в json
	if user.nickname == "" { //todo вынести в отдельную функцию, validate user
		s.log.Debug(op, ": no nickname")
		http.Error(w, "nickname is required", http.StatusBadRequest)
		return
	}
	if user.email == "" { //todo туда же в отдельную функцию
		s.log.Debug(op, ": no email")
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	err := domain.VerifyEmail(user.email)
	if err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		s.log.Debug(op, ": failed to validate request body: "+err.Error())
		return
	}
	//Вызов домейновой функции по добавлению пользователя
	err = s.srv.AddUser(s.context, user.toDomain())
	if err != nil {
		s.log.Error(op, ": failed to add user: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.log.Info("registered user", user.nickname)
	w.WriteHeader(http.StatusCreated) //ответ
	return
}

func (s Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	const op = "gates.server.statusHandler"
	s.log.Info(op, ": starting status")
	//извлечение userid из адреса
	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		s.log.Debug(op, ": empty id")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}
	s.log.Debug(op, "trying to get status for user:", idParam)
	userID := domain.UserID(idParam)
	//Получение пользователя
	user, err := s.srv.Status(s.context, userID)
	if err == sql.ErrNoRows { //обработка ошибки что такого пользователя нет
		s.log.Debug(op, ": user not found")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.log.Error(op, ": failed to get status: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	//формирование ответа
	resp, err := json.Marshal(user)
	if err != nil {
		s.log.Error(op, ": failed to encode user: ", err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusBadRequest)
		return
	}
	s.log.Info(op, ": sucessfully logged in")
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
	w.WriteHeader(http.StatusOK)
}

func (s Server) leaderbord(writer http.ResponseWriter, request *http.Request) {
	const op = "gates.server.leaderbord"
}

func (s Server) taskCompleteHandler(writer http.ResponseWriter, request *http.Request) {

}

func (s Server) reffererHandler(writer http.ResponseWriter, request *http.Request) {

}
