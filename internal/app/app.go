package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/ikolcov/microblog/internal/models"
	"github.com/ikolcov/microblog/internal/storage"
	"github.com/ikolcov/microblog/internal/utils"
)

type AppConfig struct {
	Port uint16
}

type App struct {
	config  AppConfig
	storage storage.Storage
}

func New(config AppConfig) *App {
	return &App{
		config:  config,
		storage: storage.NewPostsStorage(),
	}
}

func (a *App) addPost(w http.ResponseWriter, r *http.Request) {
	var post models.Post
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&post); err != nil {
		panic(err)
	}

	post.AuthorId = models.UserID(r.Header.Get("System-Design-User-Id"))
	post.CreatedAt = time.Now()

	postId, err := a.storage.AddPost(post)
	if errors.Is(err, storage.ErrUnauthorized) {
		utils.Unauthorized(w, err.Error())
		return
	} else if err != nil {
		panic(err)
	}
	post.Id = postId

	utils.RespondJSON(w, http.StatusOK, post)
}

func (a *App) getPost(w http.ResponseWriter, r *http.Request) {
	post, err := a.storage.GetPost(models.PostID(chi.URLParam(r, "postId")))
	if errors.Is(err, storage.ErrNotFound) {
		utils.NotFound(w, err.Error())
		return
	} else if err != nil {
		panic(err)
	}

	utils.RespondJSON(w, http.StatusOK, post)
}

func getParam(r *http.Request, key string, defaultValue int) int {
	param := r.URL.Query().Get(key)
	if parsedParam, err := strconv.Atoi(param); err == nil {
		return parsedParam
	}
	return defaultValue
}

func (a *App) getUserPosts(w http.ResponseWriter, r *http.Request) {
	userId := models.UserID(chi.URLParam(r, "userId"))
	page := getParam(r, "page", 1)
	size := getParam(r, "size", 10)

	postsPage, err := a.storage.GetUserPosts(userId, page, size)
	if errors.Is(err, storage.ErrBadRequest) {
		utils.BadRequest(w, err.Error())
	} else if err != nil {
		panic(err)
	}

	utils.RespondJSON(w, http.StatusOK, postsPage)
}

func (a *App) initRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	r.Route("/api/v1/posts", func(r chi.Router) {
		r.Post("/", a.addPost)
		r.Get("/{postId}", a.getPost)
	})
	r.Get("/api/v1/users/{userId}/posts", a.getUserPosts)
	return r
}

func (a *App) Start() {
	handler := a.initRoutes()
	http.ListenAndServe(fmt.Sprintf(":%v", a.config.Port), handler)
}
