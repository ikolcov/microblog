package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/ikolcov/microblog/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStorage struct {
	posts         *mongo.Collection
	subscriptions *mongo.Collection
	feed          *mongo.Collection
}

func addIndex(collection *mongo.Collection, field string) {
	index := mongo.IndexModel{
		Keys: bson.D{{field, 1}},
	}
	_, err := collection.Indexes().CreateOne(context.TODO(), index)
	if err != nil {
		panic(err)
	}
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

func (s *MongoStorage) UpdatePost(postUpdate models.Post) (models.Post, error) {
	if postUpdate.AuthorId == "" {
		return *new(models.Post), models.ErrUnauthorized
	}
	post, err := s.GetPost(postUpdate.Id)
	if err != nil {
		return *new(models.Post), err
	}
	if post.AuthorId != postUpdate.AuthorId {
		return *new(models.Post), models.ErrFobidden
	}

	id, _ := primitive.ObjectIDFromHex(string(postUpdate.Id))
	filter := bson.D{{"_id", id}}
	update := bson.D{{"$set", bson.D{{"text", postUpdate.Text}, {"lastmodifiedat", postUpdate.LastModifiedAt}}}}

	if _, err := s.posts.UpdateOne(context.TODO(), filter, update); err != nil {
		return *new(models.Post), err
	}
	post.Text = postUpdate.Text
	post.LastModifiedAt = postUpdate.LastModifiedAt
	return post, nil
}

func (s *MongoStorage) getAllUserPosts(userId models.UserID) ([]models.Post, error) {
	findOptions := options.Find()
	cur, err := s.posts.Find(context.TODO(), bson.D{{"authorid", userId}}, findOptions)
	if err != nil {
		return nil, err
	}
	posts := make([]models.Post, 0)
	for cur.Next(context.TODO()) {
		var elem models.Post
		if err := cur.Decode(&elem); err != nil {
			return nil, err
		}
		var id models.HexId
		if err := cur.Decode(&id); err != nil {
			return nil, err
		}
		elem.Id = models.PostID(id.ID.Hex())
		posts = append(posts, elem)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	cur.Close(context.TODO())

	return posts, nil
}

func (s *MongoStorage) GetUserPosts(userId models.UserID, page int, size int) (models.PostsPage, error) {
	allUserPosts, err := s.getAllUserPosts(userId)
	if err != nil {
		return models.PostsPage{}, err
	}

	l, r := 0, len(allUserPosts)-1
	for l < r {
		allUserPosts[l], allUserPosts[r] = allUserPosts[r], allUserPosts[l]
		l++
		r--
	}

	return getPostsPage(allUserPosts, page, size)
}

func getPostsPage(posts []models.Post, page int, size int) (models.PostsPage, error) {
	from := (page - 1) * size
	if from < 0 || from > len(posts) {
		return models.PostsPage{}, models.ErrBadRequest
	}
	to := from + size
	if to > len(posts) {
		to = len(posts)
	}

	postsPage := models.PostsPage{}
	postsPage.Posts = posts[from:to]
	if to < len(posts) {
		postsPage.NextPage = fmt.Sprint(page + 1)
	}
	return postsPage, nil
}

func (s *MongoStorage) AddSubscription(subscription models.Subscription) error {
	if subscription.From == "" || subscription.To == "" || subscription.From == subscription.To {
		return models.ErrBadRequest
	}

	s.feed.InsertOne(context.TODO(), models.Feed{
		User:  subscription.From,
		Posts: make([]models.Post, 0),
	})
	_, err := s.subscriptions.InsertOne(context.TODO(), subscription)
	if err != nil && strings.Contains(err.Error(), "duplicate") {
		return nil
	}
	return err
}

func (s *MongoStorage) GetSubscriptions(userId models.UserID) (models.UsersList, error) {
	cur, err := s.subscriptions.Find(context.TODO(), bson.D{{"from", userId}}, options.Find())
	if err != nil {
		return models.UsersList{}, err
	}
	users := make([]models.UserID, 0)
	for cur.Next(context.TODO()) {
		var elem models.Subscription
		if err := cur.Decode(&elem); err != nil {
			return models.UsersList{}, err
		}
		users = append(users, elem.To)
	}
	if err := cur.Err(); err != nil {
		return models.UsersList{}, err
	}
	cur.Close(context.TODO())

	return models.UsersList{users}, nil
}

func (s *MongoStorage) GetSubscribers(userId models.UserID) (models.UsersList, error) {
	cur, err := s.subscriptions.Find(context.TODO(), bson.D{{"to", userId}}, options.Find())
	if err != nil {
		return models.UsersList{}, err
	}
	users := make([]models.UserID, 0)
	for cur.Next(context.TODO()) {
		var elem models.Subscription
		if err := cur.Decode(&elem); err != nil {
			return models.UsersList{}, err
		}
		users = append(users, elem.From)
	}
	if err := cur.Err(); err != nil {
		return models.UsersList{}, err
	}
	cur.Close(context.TODO())

	return models.UsersList{users}, nil
}

func (s *MongoStorage) UpdateUserFeed(userId string) error {
	subscriptions, err := s.GetSubscriptions(models.UserID(userId))
	if err != nil {
		return err
	}

	allPosts := make([]models.Post, 0)
	for _, userId := range subscriptions.Users {
		allUserPosts, err := s.getAllUserPosts(userId)
		if err != nil {
			return err
		}
		allPosts = append(allPosts, allUserPosts...)
	}

	sort.Slice(allPosts, func(i, j int) bool {
		return allPosts[i].CreatedTime.After(allPosts[j].CreatedTime)
	})

	filter := bson.D{{"user", userId}}
	update := bson.D{{"$set", bson.D{{"posts", allPosts}}}}

	_, err = s.feed.UpdateOne(context.TODO(), filter, update)
	return err
}

func (s *MongoStorage) GetFeed(userId models.UserID, page int, size int) (models.PostsPage, error) {
	var result models.Feed
	if err := s.feed.FindOne(context.TODO(), bson.D{{"user", userId}}).Decode(&result); err != nil {
		return models.PostsPage{}, err
	}
	return getPostsPage(result.Posts, page, size)
}

func NewMongoStorage(mongoUrl string, mongoDbName string) *MongoStorage {
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
	subscriptions := client.Database(mongoDbName).Collection("subscriptions")
	feed := client.Database(mongoDbName).Collection("feed")

	addIndex(posts, "authorid")
	addIndex(subscriptions, "from")
	addIndex(subscriptions, "to")

	if _, err := subscriptions.Indexes().CreateOne(context.TODO(), mongo.IndexModel{
		Keys:    bson.D{{"from", 1}, {"to", 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		panic(err)
	}

	if _, err := feed.Indexes().CreateOne(context.TODO(), mongo.IndexModel{
		Keys:    bson.D{{"user", 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		panic(err)
	}

	return &MongoStorage{
		posts:         posts,
		subscriptions: subscriptions,
		feed:          feed,
	}
}
