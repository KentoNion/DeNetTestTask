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
	"log/slog"
	"net/http"
	"strconv"
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
		auth:    auth.NewService(db, log, cfg, "secret", pkg.NormalClock{}),
	}

	//роутим эндпоинты авторизации
	r.Method(http.MethodGet, "/login/{id}", http.HandlerFunc(server.loginHandler))
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

	idParamStr := chi.URLParam(r, "id")
	if idParamStr == "" {
		s.log.Debug(op, ": empty id")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}
	idParam, err := strconv.Atoi(idParamStr)
	if err != nil {
		s.log.Debug(op, ": failed to convert srt to int Atoi")
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
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
	if user.Nickname == "" { //todo вынести в отдельную функцию, validate user
		s.log.Debug(op, ": No nickname")
		http.Error(w, "Nickname and Email is required", http.StatusBadRequest)
		return
	}
	if user.Email == "" { //todo туда же в отдельную функцию
		s.log.Debug(op, ": no email")
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	err := domain.VerifyEmail(user.Email)
	if err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		s.log.Debug(op, ": failed to validate request body: "+err.Error())
		return
	}
	s.log.Debug(op, ": email verified")
	//Вызов домейновой функции по добавлению пользователя
	s.log.Debug(op, ": trying to make duser")
	duser := user.toDomain()
	s.log.Debug(op, ": duser made")
	err = s.srv.AddUser(s.context, duser)
	if err != nil {
		s.log.Error(op, ": failed to add user: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.log.Info(op, "registered user", user.Nickname)
	w.WriteHeader(http.StatusCreated) //ответ
	return
}

func (s Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	const op = "gates.server.statusHandler"
	s.log.Info(op, ": starting status")
	//извлечение userid из адреса
	idParamStr := chi.URLParam(r, "id")
	if idParamStr == "" {
		s.log.Debug(op, ": empty id")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}
	idParam, err := strconv.Atoi(idParamStr)
	if err != nil {
		s.log.Debug(op, ": failed to convert srt to int Atoi")
		http.Error(w, "User ID must consist of numbers only", http.StatusBadRequest)
		return
	}
	//извлекаем юзера из мидлвера
	var user domain.User
	user, ok := userFromContext(r.Context())
	if !ok {
		s.log.Error(op, ": user not found in context")
		http.Error(w, "user not found in context", http.StatusInternalServerError)
		return
	}
	var resp []byte
	//если id из запроса совпадает с тем что был в jwt переданный мидлвер авторизации, то формируем ответ из юзера извлечённым из мидлвера (чтоб сократить кол-во обращений в бд)
	if user.ID == domain.UserID(idParam) {
		//формирование ответа
		resp, err = json.Marshal(user)
		if err != nil {
			s.log.Error(op, ": failed to encode user: ", err.Error())
			http.Error(w, "Something went wrong: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else { //если не совпадает, тогда ходим в бд по нужному id и формируем ответ
		user, err := s.srv.Status(s.context, domain.UserID(idParam))
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
	var set LeaderboardSettings
	//декодировка json, попытка извлечь параметры сортировки, номер страницы, размер (опционально)
	if err := json.NewDecoder(r.Body).Decode(&set); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		s.log.Error(op, ": failed to decode request body: "+err.Error())
		return
	}
	r.Body.Close()
	s.log.Debug(op, "leaderboard settings: ", set)
	leaderboard, err := s.srv.Leaderbord(s.context, set.SortBy, set.Page, set.Size)
	if err != nil {
		s.log.Error(op, ": failed to get leaderboard: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var resp []user //собираю ответ без указания email и информации о приглашении
	for _, duser := range leaderboard {
		usr := user{
			Id:         duser.ID,
			Nickname:   duser.Nickname,
			Score:      duser.Score,
			Registered: duser.Registered,
		}
		s.log.Debug(op, usr)
		resp = append(resp, usr)
	}
	s.log.Debug(op, "leaderboard:", resp)
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
	user, ok := userFromContext(r.Context())
	if !ok {
		s.log.Error(op, ": user not found in context")
		http.Error(w, "Lost data from auth", http.StatusInternalServerError)
		return
	}
	//получение id из адреса
	idParamStr := chi.URLParam(r, "id")
	if idParamStr == "" {
		s.log.Debug(op, ": empty id")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}
	idParam, err := strconv.Atoi(idParamStr)
	if err != nil {
		s.log.Debug(op, ": failed to convert srt to int Atoi")
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	if user.ID != domain.UserID(idParam) {
		s.log.Debug(op, ": request user doesn't match auth user")
		http.Error(w, "You don't have permission, you may add points only to your account", http.StatusBadRequest)
		return
	}
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		s.log.Error(op, ": Failed to decode request body: "+err.Error())
		return
	}
	r.Body.Close()
	task := req.Task
	err = s.srv.TaskComplete(s.context, user.ID, task)
	if err == domain.ErrNotExistingReward {
		s.log.Debug(op, "User tried to claim not existing reward")
		http.Error(w, "This task doesn't exist", http.StatusBadRequest)
		return
	}
	if err != nil {
		s.log.Error(op, ": failed to complete task: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	s.log.Info(op, ": registered task for user", user.ID)
	return
}

func (s Server) referrerHandler(w http.ResponseWriter, r *http.Request) {
	const op = "gates.server.reffererHandler"
	//Аналогично taskComplete, считаю что рефералки может прописывать юзер только сам себе (указывать кто пригласил)
	s.log.Info(op, ": starting refferer Handler")
	user, ok := userFromContext(r.Context())
	if !ok {
		s.log.Error(op, ": user not found in conext")
		http.Error(w, "Lost data from auth", http.StatusInternalServerError)
		return
	}
	//получение id из адреса
	idParamStr := chi.URLParam(r, "id")
	if idParamStr == "" {
		s.log.Debug(op, ": empty id")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}
	idParam, err := strconv.Atoi(idParamStr)
	if err != nil {
		s.log.Debug(op, ": failed to convert srt to int Atoi")
		http.Error(w, "User ID must consist of numbers only", http.StatusBadRequest) //если не сработал atoi, пользователь явно ввёл что-то кроме цифр как idшник
		return
	}
	if user.ID != domain.UserID(idParam) {
		s.log.Debug(op, ": request user doesn't match auth user")
		http.Error(w, "you don't have permission, you may add points only to your account", http.StatusBadRequest) //Права на вписание "пригласившего" есть только у приглашённого
		return
	}
	var referrer RefRequest
	if err := json.NewDecoder(r.Body).Decode(&referrer); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		s.log.Error(op, ": failed to decode request body: "+err.Error())
		return
	}
	ref, err := strconv.Atoi(referrer.ID)
	if err != nil {
		s.log.Debug(op, ": failed to convert srt to int Atoi")
		http.Error(w, "Referrer user ID must consist of numbers only", http.StatusBadRequest) //если не сработал atoi, пользователь явно ввёл что-то кроме цифр как idшник
		return
	}
	r.Body.Close()
	err = s.srv.InvitedBy(s.context, user.ID, domain.UserID(ref))
	if err == sql.ErrNoRows {
		s.log.Debug(op, ": referrer not found")
		http.Error(w, "Referrer not found, no such user", http.StatusNotFound) //не нашёлся пригласивший в бд
		return
	}
	if err != nil {
		s.log.Error(op, ": failed to invited user: "+err.Error())
		http.Error(w, "Something went wrong: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	s.log.Info(op, ": invited user", user.ID)
}
