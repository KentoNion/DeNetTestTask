package server

import (
	"app/auth"
	"app/domain"
	"app/iternal/config"
	"app/iternal/pkg"
	"context"
	"database/sql"
	"github.com/gin-gonic/gin"
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

func NewServer(db domain.UserStore, cfg *config.Config, log *slog.Logger, r *gin.Engine) *Server {
	server := &Server{ //формируем структуру сервера
		db:      db,
		context: context.Background(),
		log:     log,
		srv:     domain.NewUserService(db, log, cfg),
		auth:    auth.NewService(db, log, cfg, "secret", pkg.NormalClock{}),
	}

	//роутим эндпоинты авторизации
	r.GET("/login/:id", server.loginHandler)
	r.POST("/register", server.registerHandler)
	//эндпоинты с авторизацией
	authorized := r.Group("/")
	authorized.Use(server.AuthMiddleware())
	{
		authorized.GET("/users/:id/status", server.statusHandler)
		authorized.GET("/users/leaderboard", server.leaderboard)
		authorized.PATCH("/users/:id/task/complete", server.taskCompleteHandler)
		authorized.PATCH("/users/:id/referrer", server.referrerHandler)
	}
	server.log.Info("router configured")
	return server
}

func (s Server) loginHandler(c *gin.Context) {
	//логин моковый, он требует только ввести id юзера и отдаёт jwt токен
	const op = "gates.server.loginHandler"
	s.log.Info(op, ": starting login")

	idParamStr := c.Param("id")
	if idParamStr == "" {
		s.log.Debug(op, ": empty id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	idParam, err := strconv.Atoi(idParamStr)
	if err != nil {
		s.log.Debug(op, ": failed to convert srt to int Atoi")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := domain.UserID(idParam)

	s.log.Debug("login: ", id)
	token, err := s.auth.Login(s.context, id)
	if err != nil {
		s.log.Error(op, ": failed to login: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.log.Info(op, ": sucesfully logged in")
	s.log.Debug(op, "token: ", token)

	c.JSON(http.StatusOK, token)
	return
}

func (s Server) registerHandler(c *gin.Context) {
	const op = "gates.server.registerHandler"
	s.log.Info(op, ": starting register")
	var user user
	//декодировка json, извлечение данных нового пользователя
	if err := c.ShouldBindJSON(&user); err != nil {
		s.log.Error(op, ": failed to decode request body: "+err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"invalid request": err.Error()})
		return
	}

	//проверка наличия никнейма в json
	if user.Nickname == "" { //todo вынести в отдельную функцию, validate user
		s.log.Debug(op, ": No nickname")
		c.JSON(http.StatusBadRequest, gin.H{"error": "nickname is required"})
	}
	if user.Email == "" { //todo туда же в отдельную функцию
		s.log.Debug(op, ": no email")
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}

	err := domain.VerifyEmail(user.Email)
	if err != nil {
		s.log.Debug(op, ": failed to validate request body: "+err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"failed do validate email adress: ": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.log.Info(op, "registered user", user.Nickname)
	c.Status(http.StatusCreated) //ответ
	return
}

func (s Server) statusHandler(c *gin.Context) {
	const op = "gates.server.statusHandler"
	s.log.Info(op, ": starting status")
	//извлечение userid из адреса
	idParamStr := c.Param("id")
	if idParamStr == "" {
		s.log.Debug(op, ": empty id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	idParam, err := strconv.Atoi(idParamStr)
	if err != nil {
		s.log.Debug(op, ": failed to convert srt to int Atoi")
		c.JSON(http.StatusBadRequest, gin.H{"error:": "User ID must consist of numbers only"})
		return
	}
	//извлекаем юзера из мидлвера
	userAny, ok := c.Get("user")
	if !ok {
		s.log.Error(op, ": user not found in context")
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found in context"})
		return
	}
	user, ok := userAny.(domain.User)
	if !ok {
		s.log.Error(op, "failed to convert user to domain.User", userAny)
		c.JSON(http.StatusBadRequest, gin.H{"error": "internal server error"})
		return
	}
	//если id из запроса совпадает с тем что был в jwt переданный мидлвер авторизации, то формируем ответ из юзера извлечённым из мидлвера (чтоб сократить кол-во обращений в бд)
	if user.ID == domain.UserID(idParam) {
		//формирование ответа
		s.log.Info(op, ": status sucessfully retrieved", "logged user = requested user")
		c.JSON(http.StatusOK, user)
		return
	}
	user, err = s.srv.Status(s.context, domain.UserID(idParam))
	if err != nil {
		s.log.Error(op, ": failed to status user: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	s.log.Info(op, ": status sucessfully retrieved")
	c.JSON(http.StatusOK, user)
	return
}

func (s Server) leaderboard(c *gin.Context) {
	const op = "gates.server.leaderboard"
	s.log.Info(op, ": starting leaderboard")
	var set LeaderboardSettings
	//декодировка json, попытка извлечь параметры сортировки, номер страницы, размер (опционально)
	if err := c.ShouldBindJSON(&set); err != nil {
		s.log.Error(op, ": failed to decode request body: "+err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"Invalid request": err.Error()})
		return
	}
	s.log.Debug(op, "leaderboard settings: ", set)
	leaderboard, err := s.srv.Leaderbord(s.context, set.SortBy, set.Page, set.Size)
	if err != nil {
		s.log.Error(op, ": failed to get leaderboard: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"Something went wrong": err.Error()})
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

	s.log.Info(op, ": leaderboard sucessfully retrieved")
	c.JSON(http.StatusOK, resp)
	return
}

func (s Server) taskCompleteHandler(c *gin.Context) {
	const op = "gates.server.taskCompleteHandler"
	//в этом хендлере я подумал что добавлять поинты юзер может только сам себе, так что буду сверять id из authorize мидлвера и id указанный в адрессе, если не сходится то прекращать работу
	s.log.Info(op, ": starting task complete")
	userAny, ok := c.Get("user")
	if !ok {
		s.log.Error(op, ": user not found in context")
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found in context"})
		return
	}
	user, ok := userAny.(domain.User)
	if !ok {
		s.log.Error(op, ": user not found in context")
		c.JSON(http.StatusBadRequest, gin.H{"error": "can't map user to domain.User"})
		return
	}
	//получение id из адреса
	idParamStr := c.Param("id")
	if idParamStr == "" {
		s.log.Debug(op, ": empty id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
		return
	}
	idParam, err := strconv.Atoi(idParamStr)
	if err != nil {
		s.log.Debug(op, ": failed to convert srt to int Atoi")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Id must consist of numbers only"})
		return
	}
	if user.ID != domain.UserID(idParam) {
		s.log.Debug(op, ": request user doesn't match auth user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "You don't have permission, you may add points only to your account"})
		return
	}
	var req TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.log.Debug(op, ": failed to decode request body: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decode request body: " + err.Error()})
		return
	}
	task := req.Task
	err = s.srv.TaskComplete(s.context, user.ID, task)
	if err == domain.ErrNotExistingReward {
		s.log.Debug(op, "User tried to claim not existing reward")
		c.JSON(http.StatusBadRequest, gin.H{"error": "This task doesn't exist"})
		return
	}
	if err != nil {
		s.log.Error(op, ": failed to complete task: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
		return
	}
	s.log.Info(op, ": registered task for user", user.ID)
	c.Status(http.StatusOK)
	return
}

func (s Server) referrerHandler(c *gin.Context) {
	const op = "gates.server.reffererHandler"
	//Аналогично taskComplete, считаю что рефералки может прописывать юзер только сам себе (указывать кто пригласил)
	s.log.Info(op, ": starting refferer Handler")
	userAny, ok := c.Get("user")
	if !ok {
		s.log.Error(op, ": user not found in conext")
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found in context"})
		return
	}
	user, ok := userAny.(domain.User)
	if !ok {
		s.log.Error(op, ": can't map user to domain.User")
		c.JSON(http.StatusBadRequest, gin.H{"error": "something went wrong"})
		return
	}
	//получение id из адреса
	idParamStr := c.Param("id")
	if idParamStr == "" {
		s.log.Debug(op, ": empty id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
		return
	}
	idParam, err := strconv.Atoi(idParamStr)
	if err != nil {
		s.log.Debug(op, ": failed to convert srt to int Atoi")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Id must consist of numbers only"})
		return
	}
	if user.ID != domain.UserID(idParam) {
		s.log.Debug(op, ": request user doesn't match auth user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "you don't have permission, you may add points only to your account"})
		return
	}
	var referrer RefRequest
	if err := c.ShouldBindJSON(&referrer); err != nil {
		s.log.Error(op, ": failed to decode request body: "+err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decode request body: " + err.Error()})
		return
	}
	ref, err := strconv.Atoi(referrer.ID)
	if err != nil {
		s.log.Debug(op, ": failed to convert srt to int Atoi")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Id must consist of numbers only"})
		return
	}
	err = s.srv.InvitedBy(s.context, user.ID, domain.UserID(ref))
	if err == sql.ErrNoRows {
		s.log.Debug(op, ": referrer not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "referrer not found"})
		return
	}
	if err != nil {
		s.log.Error(op, ": failed to invited user: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
		return
	}
	s.log.Info(op, ": invited user", user.ID)
	c.Status(http.StatusOK)
}
