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

	//роутим эндпоинты авторизации
	r.Method(http.MethodGet, "/login{id}", http.HandlerFunc(server.loginHandler))
	r.Method(http.MethodPost, "/register", http.HandlerFunc(server.registerHandler))
	//эндпоинты с авторизацией
	r.With(server.AuthMiddleware).Method(http.MethodGet, "/users/{id}/status", http.HandlerFunc(server.statusHandler))
	r.With(server.AuthMiddleware).Method(http.MethodGet, "/users/leaderboard", http.HandlerFunc(server.leaderboard))
	r.With(server.AuthMiddleware).Method(http.MethodPatch, "/users/{id}/task/complete", http.HandlerFunc(server.taskCompleteHandler))
	r.With(server.AuthMiddleware).Method(http.MethodPatch, "/users/{id}/referrer", http.HandlerFunc(server.referrerHandler))
	server.log.Info("router configured")
	return server
}

func (s Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	//логин моковый, он требует только ввести id юзера и отдаёт jwt токен
	const op = "gates.server.loginHandler"
	s.log.Info(op, ": starting login")

	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		s.log.Debug(op, ": empty id")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}
	id := domain.UserID(idParam)

	s.log.Debug("login: ", id)
	token, err := s.auth.Login(s.context, id)
	if err != nil {
		s.log.Error(op, ": failed to login: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.log.Info(op, ": sucesfully logged in")
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
	return
}

func (s Server) registerHandler(w http.ResponseWriter, r *http.Request) {
	const op = "gates.server.registerHandler"
	s.log.Info(op, ": starting register")
	var user user
	//декодировка json, извлечение данных нового пользователя
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
	//извлекаем юзера из мидлвера
	user, ok := FromContext(r.Context())
	if !ok {
		s.log.Error(op, ": user not found in conext")
		http.Error(w, "user not found in context", http.StatusInternalServerError)
		return
	}
	var resp []byte
	var err error
	//если id из запроса совпадает с тем что был в jwt переданный мидлвер авторизации, то формируем ответ из юзера извлечённым из мидлвера (чтоб сократить кол-во обращений в бд)
	if user.id == domain.UserID(idParam) {
		//формирование ответа
		resp, err = json.Marshal(user)
		if err != nil {
			s.log.Error(op, ": failed to encode user: ", err.Error())
			http.Error(w, "Something went wrong: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else { //если не совпадает, тогда ходим в бд по нужному id и формируем ответ
		duser, err := s.srv.Status(s.context, domain.UserID(idParam))
		user = fromDomain(duser)
		resp, err = json.Marshal(user)
		if err != nil {
			s.log.Error(op, ": failed to encode user: ", err.Error())
			http.Error(w, "Something went wrong: "+err.Error(), http.StatusBadRequest)
		}
	}

	s.log.Info(op, ": status sucessfully retrieved")
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
	w.WriteHeader(http.StatusOK)
	return
}

func (s Server) leaderboard(w http.ResponseWriter, r *http.Request) {
	const op = "gates.server.leaderboard"
	s.log.Info(op, ": starting leaderboard")
	var set leaderboardSettings
	//декодировка json, попытка извлечь параметры сортировки, номер страницы, размер (опционально)
	if err := json.NewDecoder(r.Body).Decode(&set); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		s.log.Error(op, ": failed to decode request body: "+err.Error())
		return
	}
	r.Body.Close()
	leaderboard, err := s.srv.Leaderbord(s.context, set.sorter, set.page, set.size)
	if err != nil {
		s.log.Error(op, ": failed to get leaderboard: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []user //собираю ответ без указания email и информации о приглашении
	for _, duser := range leaderboard {
		usr := user{
			id:         duser.Id,
			nickname:   duser.Nickname,
			score:      duser.Score,
			registered: duser.Registered,
		}
		resp = append(resp, usr)
	}
	//формируем ответ
	responce, err := json.Marshal(resp)
	if err != nil {
		s.log.Error(op, ": failed to encode leaderboard: ", err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(responce)
	w.WriteHeader(http.StatusOK)
	s.log.Info(op, ": leaderboard sucessfully retrieved")
	return
}

func (s Server) taskCompleteHandler(w http.ResponseWriter, r *http.Request) {
	const op = "gates.server.taskCompleteHandler"
	//в этом хендлере я подумал что добавлять поинты юзер может только сам себе, так что буду сверять id из authorize мидлвера и id указанный в адрессе, если не сходится то прекращать работу
	s.log.Info(op, ": starting task complete")
	user, ok := FromContext(r.Context())
	if !ok {
		s.log.Error(op, ": user not found in conext")
		http.Error(w, "user not found in context", http.StatusInternalServerError)
		return
	}
	//получение id из адреса
	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		s.log.Debug(op, ": empty id")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}
	if user.id != domain.UserID(idParam) {
		s.log.Debug(op, ": request user doesn't match auth user")
		http.Error(w, "you don't have permission, you may add points only to your account", http.StatusBadRequest)
		return
	}
	var task string
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		s.log.Error(op, ": failed to decode request body: "+err.Error())
		return
	}
	r.Body.Close()
	err := s.srv.TaskComplete(s.context, user.id, task)
	if err != nil {
		s.log.Error(op, ": failed to complete task: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	s.log.Info(op, ": registered task for user", user.id)
	return
}

func (s Server) referrerHandler(w http.ResponseWriter, r *http.Request) {
	const op = "gates.server.reffererHandler"
	//Аналогично taskComplete, считаю что рефералки может прописывать юзер только сам себе (указывать кто пригласил)
	s.log.Info(op, ": starting task complete")
	user, ok := FromContext(r.Context())
	if !ok {
		s.log.Error(op, ": user not found in conext")
		http.Error(w, "user not found in context", http.StatusInternalServerError)
		return
	}
	//получение id из адреса
	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		s.log.Debug(op, ": empty id")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}
	if user.id != domain.UserID(idParam) {
		s.log.Debug(op, ": request user doesn't match auth user")
		http.Error(w, "you don't have permission, you may add points only to your account", http.StatusBadRequest)
		return
	}
	var referrer string
	if err := json.NewDecoder(r.Body).Decode(&referrer); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		s.log.Error(op, ": failed to decode request body: "+err.Error())
		return
	}
	r.Body.Close()
	err := s.srv.InvitedBy(s.context, user.id, domain.UserID(referrer))
	if err == sql.ErrNoRows {
		s.log.Debug(op, ": referrer not found")
		http.Error(w, "referrer not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.log.Error(op, ": failed to invited user: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	s.log.Info(op, ": invited user", user.id)
}
