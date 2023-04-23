package models

import "time"

type PostID string

type UserID string

type Post struct {
	Id        PostID    `json:"id"`
	Text      string    `json:"text"`
	AuthorId  UserID    `json:"authorId"`
	CreatedAt time.Time `json:"createdAt"`
}

type PostsPage struct {
	Posts    []Post `json:"posts"`
	NextPage string `json:"nextPage,omitempty"`
}
