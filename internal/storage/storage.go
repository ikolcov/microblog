package storage

import "github.com/ikolcov/microblog/internal/models"

type Storage interface {
	AddPost(post models.Post) (models.PostID, error)
	GetPost(postId models.PostID) (models.Post, error)
	GetUserPosts(userId models.UserID, page int, size int) (models.PostsPage, error)
}
