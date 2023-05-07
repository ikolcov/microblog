package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ikolcov/microblog/internal/models"
	"github.com/redis/go-redis/v9"
)

type CachedStorage struct {
	client            *redis.Client
	persistentStorage Storage
}

func (s *CachedStorage) AddPost(post models.Post) (models.PostID, error) {
	postId, err := s.persistentStorage.AddPost(post)
	if err != nil {
		return postId, err
	}
	post.Id = postId
	s.store(post)
	return postId, nil
}

func (s *CachedStorage) GetPost(postId models.PostID) (models.Post, error) {
	if post := s.load(postId); post != nil {
		return *post, nil
	}
	post, err := s.persistentStorage.GetPost(postId)
	if err != nil {
		return post, err
	}
	s.store(post)
	return post, nil
}

func (s *CachedStorage) UpdatePost(postUpdate models.Post) (models.Post, error) {
	post, err := s.persistentStorage.UpdatePost(postUpdate)
	if err != nil {
		return post, err
	}
	s.store(post)
	return post, nil
}

func (s *CachedStorage) GetUserPosts(userId models.UserID, page int, size int) (models.PostsPage, error) {
	return s.persistentStorage.GetUserPosts(userId, page, size)
}

func (s *CachedStorage) store(post models.Post) {
	value, err := json.Marshal(post)
	if err != nil {
		panic(err)
	}
	if err := s.client.Set(context.TODO(), s.redisKey(post.Id), value, time.Hour).Err(); err != nil {
		panic(err)
	}
	fmt.Println("successful store", post.Id)
}

func (s *CachedStorage) load(postId models.PostID) *models.Post {
	result, err := s.client.Get(context.TODO(), s.redisKey(postId)).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		panic(err)
	}
	var post models.Post
	if err := json.Unmarshal([]byte(result), &post); err != nil {
		panic(err)
	}
	fmt.Println("successful load", post.Id)
	return &post
}

func (s *CachedStorage) redisKey(key models.PostID) string {
	// add a prefix not to collide with other data stored in the same redis
	return "postid:" + string(key)
}

func NewCachedStorage(redisUrl string, persistentStorage Storage) Storage {
	client := redis.NewClient(&redis.Options{Addr: redisUrl})
	return &CachedStorage{
		client:            client,
		persistentStorage: persistentStorage,
	}
}
