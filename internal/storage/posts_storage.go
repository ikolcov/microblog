package storage

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/ikolcov/microblog/internal/models"
)

type PostsStorage struct {
	posts       []models.Post
	postsByUser map[models.UserID][]int
	mutex       sync.RWMutex
}

type Storage interface {
	AddPost(post models.Post) (models.PostID, error)
	GetPost(postId models.PostID) (models.Post, error)
	GetUserPosts(userId models.UserID, page int, size int) (models.PostsPage, error)
}

var ErrUnauthorized = errors.New("user token is invalid")
var ErrNotFound = errors.New("post is not found")
var ErrBadRequest = errors.New("bad page token")

func (s *PostsStorage) AddPost(post models.Post) (models.PostID, error) {
	if post.AuthorId == "" {
		return *new(models.PostID), ErrUnauthorized
	}

	s.mutex.Lock()
	id := len(s.posts)
	post.Id = models.PostID(fmt.Sprint(id))
	s.posts = append(s.posts, post)
	s.postsByUser[post.AuthorId] = append(s.postsByUser[post.AuthorId], id)
	s.mutex.Unlock()

	return post.Id, nil
}
func (s *PostsStorage) GetPost(postId models.PostID) (models.Post, error) {
	s.mutex.RLock()
	id, err := strconv.Atoi(string(postId))
	if err != nil || id < 0 || id >= len(s.posts) {
		return *new(models.Post), ErrNotFound
	}
	post := s.posts[id]
	s.mutex.RUnlock()
	return post, nil
}
func (s *PostsStorage) GetUserPosts(userId models.UserID, page int, size int) (models.PostsPage, error) {
	s.mutex.RLock()
	posts := s.postsByUser[userId]
	postIds := make([]int, 0)
	for i := len(posts) - 1; i >= 0; i-- {
		postIds = append(postIds, posts[i])
	}

	from := (page - 1) * size
	if from < 0 || from > len(postIds) {
		return *new(models.PostsPage), ErrBadRequest
	}
	to := from + size
	if to > len(postIds) {
		to = len(postIds)
	}

	postIds = postIds[from:to]

	postsPage := models.PostsPage{
		Posts: make([]models.Post, 0),
	}
	for _, id := range postIds {
		postsPage.Posts = append(postsPage.Posts, s.posts[id])
	}
	if to < len(posts) {
		postsPage.NextPage = fmt.Sprint(page + 1)
	}
	s.mutex.RUnlock()

	return postsPage, nil
}
func NewPostsStorage() Storage {
	return &PostsStorage{
		posts:       make([]models.Post, 0),
		postsByUser: make(map[models.UserID][]int),
	}
}
