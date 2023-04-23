package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
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
		utils.BadRequest(w, err.Error())
		return
	}

	post.AuthorId = models.UserID(r.Header.Get("System-Design-User-Id"))
	post.CreatedAt = time.Now().Format("2006-01-02T15:04:05Z")

	postId, err := a.storage.AddPost(post)
	if errors.Is(err, storage.ErrUnauthorized) {
		utils.Unauthorized(w, err.Error())
		return
	} else if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	post.Id = postId

	err = utils.RespondJSON(w, http.StatusOK, post)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
}

func (a *App) getPost(w http.ResponseWriter, r *http.Request) {
	post, err := a.storage.GetPost(models.PostID(mux.Vars(r)["postId"]))
	if errors.Is(err, storage.ErrNotFound) {
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
	userId := models.UserID(mux.Vars(r)["userId"])
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
	if errors.Is(err, storage.ErrBadRequest) {
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

func (a *App) Start() {
	router := mux.NewRouter()

	router.HandleFunc("/api/v1/posts", a.addPost).Methods("POST")
	router.HandleFunc("/api/v1/posts/{postId}", a.getPost).Methods("GET")
	router.HandleFunc("/api/v1/users/{userId}/posts", a.getUserPosts).Methods("GET")

	http.ListenAndServe(fmt.Sprintf(":%v", a.config.Port), router)
}
