package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type PostID string

type UserID string

type Post struct {
	Id        PostID `json:"id"`
	Text      string `json:"text"`
	AuthorId  UserID `json:"authorId"`
	CreatedAt string `json:"createdAt"`
}

type HexId struct {
	ID primitive.ObjectID `bson:"_id"`
}

type PostsPage struct {
	Posts    []Post `json:"posts"`
	NextPage string `json:"nextPage,omitempty"`
}
