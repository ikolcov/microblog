package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PostID string

type UserID string

type UsersList struct {
	Users []UserID `json:"users"`
}

type Post struct {
	Id             PostID    `json:"id"`
	Text           string    `json:"text"`
	AuthorId       UserID    `json:"authorId"`
	CreatedAt      string    `json:"createdAt"`
	LastModifiedAt string    `json:"lastModifiedAt"`
	CreatedTime    time.Time `json:"-"`
}

type HexId struct {
	ID primitive.ObjectID `bson:"_id"`
}

type PostsPage struct {
	Posts    []Post `json:"posts"`
	NextPage string `json:"nextPage,omitempty"`
}

type Subscription struct {
	From UserID
	To   UserID
}

type Feed struct {
	User  UserID
	Posts []Post
}
