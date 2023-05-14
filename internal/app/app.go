package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/ikolcov/microblog/internal/models"
	"github.com/ikolcov/microblog/internal/storage"
	"github.com/ikolcov/microblog/internal/utils"
)

type AppConfig struct {
	Port        uint16
	MongoUrl    string
	MongoDbName string
	RedisUrl    string
}

type App struct {
	config          AppConfig
	storage         *storage.MongoStorage
	machineryServer *machinery.Server
}

func New(config AppConfig, machineryServer *machinery.Server) *App {
	return &App{
		config:          config,
		storage:         storage.NewMongoStorage(config.MongoUrl, config.MongoDbName),
		machineryServer: machineryServer,
	}
}

func (a *App) notifySubscriber(userId models.UserID) {
	task := tasks.Signature{
		Name: "notify",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: userId,
			},
		},
	}
	if _, err := a.machineryServer.SendTaskWithContext(context.Background(), &task); err != nil {
		panic(err)
	}
}

func (a *App) addPost(w http.ResponseWriter, r *http.Request) {
	var post models.Post
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&post); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	post.AuthorId = models.UserID(r.Header.Get("System-Design-User-Id"))
	post.CreatedTime = time.Now()
	post.CreatedAt = post.CreatedTime.Format("2006-01-02T15:04:05.999Z")
	post.LastModifiedAt = post.CreatedAt

	postId, err := a.storage.AddPost(post)
	if errors.Is(err, models.ErrUnauthorized) {
		utils.Unauthorized(w, err.Error())
		return
	} else if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	post.Id = postId

	subscribers, err := a.storage.GetSubscribers(post.AuthorId)
	if err == nil {
		for _, subscriber := range subscribers.Users {
			a.notifySubscriber(subscriber)
		}
	}

	err = utils.RespondJSON(w, http.StatusOK, post)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
}

func (a *App) getPost(w http.ResponseWriter, r *http.Request) {
	post, err := a.storage.GetPost(models.PostID(chi.URLParam(r, "postId")))
	if errors.Is(err, models.ErrNotFound) {
		utils.NotFound(w, err.Error())
		return
	} else if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	err = utils.RespondJSON(w, http.StatusOK, post)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
}

func getParam(r *http.Request, key string, defaultValue int) (int, error) {
	param := r.URL.Query().Get(key)
	if param == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(param)
}

func (a *App) getUserPosts(w http.ResponseWriter, r *http.Request) {
	userId := models.UserID(chi.URLParam(r, "userId"))
	page, err := getParam(r, "page", 1)
	if err != nil || page < 1 {
		utils.BadRequest(w, "invalid page")
		return
	}
	size, err := getParam(r, "size", 10)
	if err != nil || size < 1 || size > 100 {
		utils.BadRequest(w, "invalid size")
		return
	}

	postsPage, err := a.storage.GetUserPosts(userId, page, size)
	if errors.Is(err, models.ErrBadRequest) {
		utils.BadRequest(w, err.Error())
		return
	} else if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	err = utils.RespondJSON(w, http.StatusOK, postsPage)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
}

func (a *App) ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (a *App) updatePost(w http.ResponseWriter, r *http.Request) {
	var post models.Post
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&post); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	post.AuthorId = models.UserID(r.Header.Get("System-Design-User-Id"))
	post.Id = models.PostID(chi.URLParam(r, "postId"))
	post.LastModifiedAt = time.Now().Format("2006-01-02T15:04:05.999Z")

	post, err := a.storage.UpdatePost(post)
	if errors.Is(err, models.ErrUnauthorized) {
		utils.Unauthorized(w, err.Error())
		return
	} else if errors.Is(err, models.ErrFobidden) {
		utils.Forbidden(w, err.Error())
		return
	} else if errors.Is(err, models.ErrNotFound) {
		utils.NotFound(w, err.Error())
		return
	} else if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	subscribers, err := a.storage.GetSubscribers(post.AuthorId)
	if err == nil {
		for _, subscriber := range subscribers.Users {
			a.notifySubscriber(subscriber)
		}
	}

	err = utils.RespondJSON(w, http.StatusOK, post)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
}

func (a *App) subscribeToUser(w http.ResponseWriter, r *http.Request) {
	from := models.UserID(r.Header.Get("System-Design-User-Id"))
	to := models.UserID(chi.URLParam(r, "userId"))

	err := a.storage.AddSubscription(models.Subscription{
		From: from,
		To:   to,
	})
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *App) getSubscriptions(w http.ResponseWriter, r *http.Request) {
	userId := models.UserID(r.Header.Get("System-Design-User-Id"))

	usersList, err := a.storage.GetSubscriptions(userId)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, usersList)
}

func (a *App) getSubscribers(w http.ResponseWriter, r *http.Request) {
	userId := models.UserID(r.Header.Get("System-Design-User-Id"))

	usersList, err := a.storage.GetSubscribers(userId)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, usersList)
}

func (a *App) getFeed(w http.ResponseWriter, r *http.Request) {
	userId := models.UserID(r.Header.Get("System-Design-User-Id"))
	page, err := getParam(r, "page", 1)
	if err != nil || page < 1 {
		utils.BadRequest(w, "invalid page")
		return
	}
	size, err := getParam(r, "size", 10)
	if err != nil || size < 1 || size > 100 {
		utils.BadRequest(w, "invalid size")
		return
	}

	postsPage, err := a.storage.GetFeed(userId, page, size)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	err = utils.RespondJSON(w, http.StatusOK, postsPage)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
}

func (a *App) Start() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	r.Post("/api/v1/posts", a.addPost)
	r.Get("/api/v1/posts/{postId}", a.getPost)
	r.Get("/api/v1/users/{userId}/posts", a.getUserPosts)
	r.Get("/maintenance/ping", a.ping)
	r.Patch("/api/v1/posts/{postId}", a.updatePost)
	r.Post("/api/v1/users/{userId}/subscribe", a.subscribeToUser)
	r.Get("/api/v1/subscriptions", a.getSubscriptions)
	r.Get("/api/v1/subscribers", a.getSubscribers)
	r.Get("/api/v1/feed", a.getFeed)

	http.ListenAndServe(fmt.Sprintf(":%v", a.config.Port), r)
}
