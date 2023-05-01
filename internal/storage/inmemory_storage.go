package storage

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/ikolcov/microblog/internal/models"
)

type InMemoryStorage struct {
	posts       []models.Post
	postsByUser map[models.UserID][]int
	mutex       sync.RWMutex
}

func (s *InMemoryStorage) AddPost(post models.Post) (models.PostID, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if post.AuthorId == "" {
		return *new(models.PostID), models.ErrUnauthorized
	}

	id := len(s.posts)
	post.Id = models.PostID(fmt.Sprint(id))
	s.posts = append(s.posts, post)
	s.postsByUser[post.AuthorId] = append(s.postsByUser[post.AuthorId], id)

	return post.Id, nil
}

func (s *InMemoryStorage) GetPost(postId models.PostID) (models.Post, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	id, err := strconv.Atoi(string(postId))
	if err != nil || id < 0 || id >= len(s.posts) {
		return *new(models.Post), models.ErrNotFound
	}
	post := s.posts[id]
	return post, nil
}

func (s *InMemoryStorage) GetUserPosts(userId models.UserID, page int, size int) (models.PostsPage, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	posts := s.postsByUser[userId]
	postIds := make([]int, 0)
	for i := len(posts) - 1; i >= 0; i-- {
		postIds = append(postIds, posts[i])
	}

	from := (page - 1) * size
	if from < 0 || from > len(postIds) {
		return *new(models.PostsPage), models.ErrBadRequest
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

	return postsPage, nil
}

func NewInMemoryStorage() Storage {
	return &InMemoryStorage{
		posts:       make([]models.Post, 0),
		postsByUser: make(map[models.UserID][]int),
	}
}
