package storage

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/ikolcov/microblog/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStorage struct {
	posts *mongo.Collection
}

func (s *MongoStorage) AddPost(post models.Post) (models.PostID, error) {
	if post.AuthorId == "" {
		return *new(models.PostID), models.ErrUnauthorized
	}

	insertResult, err := s.posts.InsertOne(context.TODO(), post)
	if err != nil {
		return *new(models.PostID), err
	}

	return models.PostID(insertResult.InsertedID.(primitive.ObjectID).Hex()), nil
}

func (s *MongoStorage) GetPost(postId models.PostID) (models.Post, error) {
	id, err := primitive.ObjectIDFromHex(string(postId))
	if err != nil {
		return *new(models.Post), models.ErrNotFound
	}

	var result models.Post
	err = s.posts.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return *new(models.Post), models.ErrNotFound
	}
	result.Id = postId
	return result, err
}

func (s *MongoStorage) GetUserPosts(userId models.UserID, page int, size int) (models.PostsPage, error) {
	findOptions := options.Find()
	postsPage := models.PostsPage{}
	cur, err := s.posts.Find(context.TODO(), bson.D{{"authorid", userId}}, findOptions)
	if err != nil {
		return postsPage, err
	}
	posts := make([]models.Post, 0)
	for cur.Next(context.TODO()) {
		var elem models.Post
		if err := cur.Decode(&elem); err != nil {
			return postsPage, err
		}
		posts = append(posts, elem)
	}
	if err := cur.Err(); err != nil {
		return postsPage, err
	}
	cur.Close(context.TODO())

	l := 0
	r := len(posts) - 1
	for l < r {
		posts[l], posts[r] = posts[r], posts[l]
		l++
		r--
	}

	from := (page - 1) * size
	if from < 0 || from > len(posts) {
		return postsPage, models.ErrBadRequest
	}
	to := from + size
	if to > len(posts) {
		to = len(posts)
	}

	postsPage.Posts = posts[from:to]
	if to < len(posts) {
		postsPage.NextPage = fmt.Sprint(page + 1)
	}
	return postsPage, nil
}

func NewMongoStorage(mongoUrl string, mongoDbName string) Storage {
	clientOptions := options.Client().ApplyURI(mongoUrl)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	posts := client.Database(mongoDbName).Collection("posts")
	return &MongoStorage{
		posts: posts,
	}
}
