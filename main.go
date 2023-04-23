package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/ikolcov/microblog/internal/models"
	"github.com/ikolcov/microblog/internal/storage"
	"github.com/ikolcov/microblog/internal/utils"
)

func getServerPort() uint16 {
	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		if port, err := strconv.ParseUint(serverPort, 10, 16); err == nil {
			return uint16(port)
		}
	}
	panic("Port should be set in env var SERVER_PORT")
}

func main() {
	port := getServerPort()
	router := mux.NewRouter()

	s := storage.NewPostsStorage()
	router.HandleFunc("/api/v1/posts", func(w http.ResponseWriter, r *http.Request) {
		var post models.Post
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&post); err != nil {
			utils.BadRequest(w, err.Error())
			return
		}

		post.AuthorId = models.UserID(r.Header.Get("System-Design-User-Id"))
		post.CreatedAt = time.Now().Format("2006-01-02T15:04:05Z")

		postId, err := s.AddPost(post)
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
	}).Methods("POST")
	http.ListenAndServe(fmt.Sprintf(":%v", port), router)
}
