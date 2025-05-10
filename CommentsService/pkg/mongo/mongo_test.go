package mongo

import (
	"comments/pkg/models"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

func TestStorage_CreateComment(t *testing.T) {
	db, err := StorageConnect()
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}

	t.Cleanup(func() {
		err := RestoreDB(db)
		if err != nil {
			t.Logf("WARNING: unable to restore DB state after the test: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		db.Close(ctx)
	})

	testCommentID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}
	testReplyID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}
	targetPostID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}

	var testComment = models.Comment{
		ID:        testCommentID,
		PostID:    targetPostID,
		Author:    "John Doe",
		Text:      "This is a test comment",
		Published: time.Date(2025, 1, 12, 10, 22, 13, 0, time.UTC),
	}
	var testReply = models.Comment{
		ID:        testReplyID,
		PostID:    targetPostID,
		ParentID:  testCommentID,
		Author:    "Alex Smith",
		Text:      "This is a test comment",
		Published: time.Date(2025, 1, 15, 14, 01, 55, 0, time.UTC),
	}
	var testComments = map[uuid.UUID]models.Comment{
		testCommentID: testComment,
		testReplyID:   testReply,
	}

	gotComment, err := db.CreateComment(testComment)
	if err != nil {
		t.Errorf("unexpected error adding comment: %v", err)
	}
	if !reflect.DeepEqual(gotComment, testComment) {
		t.Errorf("want comment\n%+v\n\ngot comment\n%+v\n", testComment, gotComment)
	}
	gotReply, err := db.CreateComment(testReply)
	if err != nil {
		t.Errorf("unexpected error adding reply: %v", err)
	}
	if !reflect.DeepEqual(gotReply, testReply) {
		t.Errorf("want reply\n%+v\n\ngot reply\n%+v\n", testReply, gotReply)
	}

	coll := db.client.Database(db.dbName).Collection("comments")
	cur, err := coll.Find(context.Background(), bson.M{})
	if err != nil {
		t.Fatalf("unexpected error retrieving comment from DB: %v", err)
	}
	defer cur.Close(context.Background())

	var gotComments []models.Comment
	for cur.Next(context.Background()) {
		var c models.Comment
		err := cur.Decode(&c)
		if err != nil {
			t.Fatalf("unexpected error decoding comment: %v", err)
		}
		gotComments = append(gotComments, c)
	}
	if cur.Err() != nil {
		t.Fatalf("unexpected error decoding comments: %v", err)
	}

	for _, gotComment := range gotComments {
		wantComment := testComments[gotComment.ID]
		if !reflect.DeepEqual(wantComment, gotComment) {
			t.Errorf("want comment\n%+v\n\ngot comment\n%+v\n", wantComment, gotComment)
		}
	}
}

// TODO: negative test cases for CreateComment().

func TestStorage_Comments(t *testing.T) {
	db, err := StorageConnect()
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}

	t.Cleanup(func() {
		err := RestoreDB(db)
		if err != nil {
			t.Logf("WARNING: unable to restore DB state after the test: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		db.Close(ctx)
	})

	postID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate postID: %v", err)
	}

	// Comments structure:
	// comment
	// ├─ reply1
	// │  └─ reply1_a
	// └─ reply2

	commentID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}
	reply1ID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}
	reply1_aID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}
	reply2ID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}

	testComments := []models.Comment{
		{
			ID:        commentID,
			PostID:    postID,
			Author:    "Alice",
			Text:      "Top-level comment",
			Published: time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:        reply1ID,
			PostID:    postID,
			ParentID:  commentID,
			Author:    "Bob",
			Text:      "Reply to top-level comment",
			Published: time.Date(2025, 5, 1, 10, 5, 0, 0, time.UTC),
		},
		{
			ID:        reply1_aID,
			PostID:    postID,
			ParentID:  reply1ID,
			Author:    "Carol",
			Text:      "Nested reply",
			Published: time.Date(2025, 5, 1, 10, 10, 0, 0, time.UTC),
		},
		{
			ID:        reply2ID,
			PostID:    postID,
			ParentID:  commentID,
			Author:    "Dave",
			Text:      "Another reply to top-level comment",
			Published: time.Date(2025, 5, 1, 10, 7, 0, 0, time.UTC),
		},
	}

	coll := db.client.Database(db.dbName).Collection("comments")
	for _, c := range testComments {
		_, err := coll.InsertOne(context.Background(), c)
		if err != nil {
			t.Fatalf("unexpected error adding comments %v: %v", c.ID, err)
		}
	}

	gotComments, err := db.Comments(postID)
	if err != nil {
		t.Fatalf("unexpected error retrieving comments: %v", err)
	}

	wantComments := []*models.Comment{
		{
			ID:        commentID,
			PostID:    postID,
			Author:    "Alice",
			Text:      "Top-level comment",
			Published: time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC),
			Replies: []*models.Comment{
				{
					ID:        reply1ID,
					PostID:    postID,
					ParentID:  commentID,
					Author:    "Bob",
					Text:      "Reply to top-level comment",
					Published: time.Date(2025, 5, 1, 10, 5, 0, 0, time.UTC),
					Replies: []*models.Comment{
						{
							ID:        reply1_aID,
							PostID:    postID,
							ParentID:  reply1ID,
							Author:    "Carol",
							Text:      "Nested reply",
							Published: time.Date(2025, 5, 1, 10, 10, 0, 0, time.UTC),
						},
					},
				},
				{
					ID:        reply2ID,
					PostID:    postID,
					ParentID:  commentID,
					Author:    "Dave",
					Text:      "Another reply to top-level comment",
					Published: time.Date(2025, 5, 1, 10, 7, 0, 0, time.UTC),
				},
			},
		},
	}

	if !reflect.DeepEqual(wantComments, gotComments) {
		t.Errorf("want comments\n%+v\n\ngot comments\n%+v\n", wantComments, gotComments)
	}
}
