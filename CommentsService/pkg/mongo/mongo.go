package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"comments/pkg/models"
)

var (
	ErrConnectDB       = fmt.Errorf("unable to establish DB connection")
	ErrDBNotResponding = fmt.Errorf("DB not responding")

	ErrPostIDNotProvided     = fmt.Errorf("postID not provided")
	ErrParentCommentNotFound = fmt.Errorf("parent comment not found")
	ErrCommentsNotFound      = fmt.Errorf("comments not found")
)

type Storage struct {
	client *mongo.Client
	dbName string
}

func New(conf *Config) (*Storage, error) {
	opt := conf.Options()
	client, err := mongo.Connect(context.Background(), opt)
	if err != nil {
		return nil, err
	}

	s := Storage{client: client, dbName: conf.DBName}
	s.createCollection("comments")

	return &s, nil
}

func (s *Storage) Ping() error {
	return s.client.Ping(context.Background(), nil)
}

func (s *Storage) Close(ctx context.Context) {
	s.client.Disconnect(ctx)
}

// CreateComment inserts a new comment into the database.
//
// Validates that PostID is provided and, if ParentID is set, verifies the parent comment exists in
// the same post. If the comment's ID or Published timestamp are zero values, they are automatically
// generated here. Returns an error if validation fails or insertion encounters issues.
func (s *Storage) CreateComment(comment models.Comment) (models.Comment, error) {
	if comment.PostID == uuid.Nil {
		return models.Comment{}, ErrPostIDNotProvided
	}

	if comment.ID == uuid.Nil {
		id, err := uuid.NewV4()
		if err != nil {
			return models.Comment{}, err
		}
		comment.ID = id
	}

	if comment.Published.IsZero() {
		comment.Published = time.Now()
	}

	coll := s.client.Database(s.dbName).Collection("comments")

	if comment.ParentID != uuid.Nil {
		cnt, err := coll.CountDocuments(context.Background(), bson.M{
			"_id":     comment.ParentID,
			"post_id": comment.PostID,
		})
		if err != nil {
			return models.Comment{}, err
		}
		if cnt == 0 {
			return models.Comment{}, ErrParentCommentNotFound
		}
	}

	_, err := coll.InsertOne(context.Background(), comment)
	if err != nil {
		return models.Comment{}, err
	}

	return comment, nil
}

// Comments returns the nested comment tree for a given postID.
//
// It fetches all comments for the post sorted by Published ascending,
// then builds and returns the tree by linking replies to their parents.
//
// Returns root comments or an error if postID is invalid or query fails.
func (s *Storage) Comments(postID uuid.UUID) ([]*models.Comment, error) {
	if postID == uuid.Nil {
		return nil, ErrPostIDNotProvided
	}

	coll := s.client.Database(s.dbName).Collection("comments")
	opts := options.Find().SetSort(bson.D{{Key: "published", Value: 1}})

	cur, err := coll.Find(context.Background(), bson.M{"post_id": postID}, opts)
	if err != nil {
		return nil, err
	}

	var comments []models.Comment
	if err := cur.All(context.Background(), &comments); err != nil {
		return nil, err
	}

	if len(comments) == 0 {
		return nil, ErrCommentsNotFound
	}

	commentMap := make(map[uuid.UUID]*models.Comment)
	for i := range comments {
		commentMap[comments[i].ID] = &comments[i]
	}

	var roots []*models.Comment
	for i := range comments {
		c := &comments[i]
		if c.ParentID == uuid.Nil {
			roots = append(roots, c)
		} else if parent, ok := commentMap[c.ParentID]; ok {
			parent.Replies = append(parent.Replies, c)
		}
	}

	return roots, nil
}

// createCollection creates a collection with the given name in the database if it doesn't already exist.
func (s *Storage) createCollection(collName string) error {
	collExists, err := collectionExists(s.client.Database(s.dbName), collName)
	if err != nil {
		return err
	}

	if !collExists {
		err := s.client.Database(s.dbName).CreateCollection(context.Background(), collName)
		if err != nil {
			return err
		}
	}

	return nil
}

// collectionExists checks if a collection with the given name exists in the database.
func collectionExists(db *mongo.Database, collName string) (bool, error) {
	names, err := db.ListCollectionNames(context.Background(), bson.D{})
	if err != nil {
		return false, fmt.Errorf("failed to list collection names: %w", err)
	}

	for _, name := range names {
		if name == collName {
			return true, nil
		}
	}

	return false, nil
}
