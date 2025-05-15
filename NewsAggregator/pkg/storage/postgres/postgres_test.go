package postgres

import (
	"context"
	"errors"
	"news/pkg/storage"
	"news/pkg/storage/memdb"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
)

const testPostsPath = "../../../test_data/post_examples.json"
const defaultPostgresPass = "some_pass"
const defaultPostgresPort = "5432"

func postgresConf() Config {
	pass := os.Getenv("POSTGRES_PASSWORD")
	if pass == "" {
		pass = defaultPostgresPass
	}

	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = defaultPostgresPort
	}

	conf := Config{
		User:     "postgres",
		Password: pass,
		Host:     "localhost",
		Port:     port,
		DBName:   "news",
	}

	return conf
}

func storageConnect() (*Store, error) {
	conf := postgresConf()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := New(ctx, conf.ConString())
	if err != nil {
		return nil, storage.ErrConnectDB
	}

	err = db.Ping(ctx)
	if err != nil {
		return nil, storage.ErrDBNotResponding
	}

	return db, nil
}

// truncatePosts restores the original state of DB for further testing.
func truncatePosts(db *Store) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.db.Exec(ctx, "TRUNCATE TABLE posts")
	if err != nil {
		return err
	}

	return nil
}

func TestMain(m *testing.M) {
	log.SetLevel(log.PanicLevel)
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestStore_AddPost(t *testing.T) {
	db, err := storageConnect()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := truncatePosts(db)
		if err != nil {
			t.Errorf("unexpected error clearing posts table: %v", err)
		}

		db.Close()
	})

	testPosts, err := memdb.LoadTestPosts(testPostsPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i, post := range testPosts {
		testPosts[i].ID, err = db.AddPost(ctx, post)
		if err != nil {
			t.Errorf("unexpected error while adding post: %v", err)
		}
	}

	for _, post := range testPosts {
		var gotPost storage.Post
		err := db.db.QueryRow(ctx, `
			SELECT id, title, content, published, link
			FROM posts
			WHERE id = $1
		`,
			post.ID,
		).Scan(
			&gotPost.ID,
			&gotPost.Title,
			&gotPost.Content,
			&gotPost.Published,
			&gotPost.Link,
		)
		gotPost.Published = gotPost.Published.UTC()
		if err != nil {
			t.Fatalf("unexpected error retrieving post ID:%v: %v", post.ID, err)
		}
		if !reflect.DeepEqual(post, gotPost) {
			t.Errorf("want post\n%+v\ngot post\n%+v\n", post, gotPost)
		}
	}
}

func TestStore_AddPosts(t *testing.T) {
	db, err := storageConnect()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := truncatePosts(db)
		if err != nil {
			t.Errorf("unexpected error clearing posts table: %v", err)
		}

		db.Close()
	})

	testPosts, err := memdb.LoadTestPosts(testPostsPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.AddPosts(ctx, testPosts)
	if err != nil {
		t.Errorf("unexpected error while adding multiple posts: %v", err)
	}

	rows, err := db.db.Query(ctx, `
		SELECT id, title, content, published, link
		FROM posts
	`)
	if err != nil {
		t.Fatalf("unexpected error retrieving posts: %v", err)
	}
	var gotPosts []storage.Post
	for rows.Next() {
		var p storage.Post
		err = rows.Scan(
			&p.ID,
			&p.Title,
			&p.Content,
			&p.Published,
			&p.Link,
		)
		if err != nil {
			t.Errorf("unexpected error while scanning posts: %v", err)
		}
		p.Published = p.Published.UTC()
		gotPosts = append(gotPosts, p)
	}
	if rows.Err() != nil {
		t.Fatalf("unexpected error retrieving posts: %v", err)
	}

	wantPostCnt := len(testPosts)
	gotPostCnt := len(gotPosts)
	if wantPostCnt != gotPostCnt {
		t.Errorf("want %d posts in DB, got %d posts in DB", wantPostCnt, gotPostCnt)
	}
}

func TestStore_LatestPosts(t *testing.T) {
	db, err := storageConnect()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := truncatePosts(db)
		if err != nil {
			t.Errorf("unexpected error clearing posts table: %v", err)
		}

		db.Close()
	})

	tests := []struct {
		name         string
		currentPage  int
		itemsPerPage int
		wantTitles   []string
		wantPagesNum int
		wantErr      bool
	}{
		{
			name:         "First page, 5 per page",
			currentPage:  1,
			itemsPerPage: 5,
			wantTitles: []string{
				"A Tale of a Cat",
				"Кириллица в названии",
				"Post 18",
				"Post 17",
				"Post 16",
			},
			wantPagesNum: 4,
			wantErr:      false,
		},
		{
			name:         "Second page, 5 per page",
			currentPage:  2,
			itemsPerPage: 5,
			wantTitles: []string{
				"Post 15",
				"Post 14",
				"Post 13",
				"Post 12",
				"Post 11",
			},
			wantPagesNum: 4,
			wantErr:      false,
		},
		{
			name:         "Third page, 5 per page",
			currentPage:  3,
			itemsPerPage: 5,
			wantTitles: []string{
				"Post 10",
				"Post 9",
				"Post 8",
				"Post 7",
				"Post 6",
			},
			wantPagesNum: 4,
			wantErr:      false,
		},
		{
			name:         "Fourth page, 5 per page",
			currentPage:  4,
			itemsPerPage: 5,
			wantTitles: []string{
				"Post 5",
				"Post 4",
				"Post 3",
				"Post 2",
				"Post 1",
			},
			wantPagesNum: 4,
			wantErr:      false,
		},
		{
			name:         "Page out of range",
			currentPage:  5,
			itemsPerPage: 5,
			wantTitles:   nil,
			wantPagesNum: 4,
			wantErr:      false,
		},
		{
			name:         "Zero items per page (uses default 10)",
			currentPage:  1,
			itemsPerPage: 0,
			wantTitles: []string{
				"A Tale of a Cat",
				"Кириллица в названии",
				"Post 18",
				"Post 17",
				"Post 16",
				"Post 15",
				"Post 14",
				"Post 13",
				"Post 12",
				"Post 11",
			},
			wantPagesNum: 2, // 20 posts / 10 per page = 2 pages
			wantErr:      false,
		},
		{
			name:         "Negative page number (uses default page 1)",
			currentPage:  -1,
			itemsPerPage: 5,
			wantTitles: []string{
				"A Tale of a Cat",
				"Кириллица в названии",
				"Post 18",
				"Post 17",
				"Post 16",
			},
			wantPagesNum: 4,
			wantErr:      false,
		},
	}

	testPosts, err := memdb.LoadTestPosts(testPostsPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i, post := range testPosts {
		testPosts[i].ID, err = db.AddPost(ctx, post)
		if err != nil {
			t.Fatalf("unexpected error while populating DB: %v", err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts, pagesNum, err := db.LatestPosts(ctx, tt.currentPage, tt.itemsPerPage)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LatestPosts() error = %v, wantErr %v", err, tt.wantErr)
			}

			if pagesNum != tt.wantPagesNum {
				t.Errorf("LatestPosts() pagesNum = %v, want %v", pagesNum, tt.wantPagesNum)
			}

			var gotTitles []string
			for _, p := range posts {
				gotTitles = append(gotTitles, p.Title)
			}

			if !reflect.DeepEqual(gotTitles, tt.wantTitles) {
				t.Errorf("LatestPosts() titles = %v, want %v", gotTitles, tt.wantTitles)
			}
		})
	}
}

func TestStore_FilterPosts(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		wantMatchCnt int
	}{
		{name: "Exact match", text: "A Tale of a Cat", wantMatchCnt: 1},
		{name: "Partial match", text: "e of a", wantMatchCnt: 1},
		{name: "Mixed case", text: "tAlE", wantMatchCnt: 1},
		{name: "Lowercase search", text: "post", wantMatchCnt: 18},
		{name: "Uppercase search", text: "POST", wantMatchCnt: 18},
		{name: "Cyrillic title", text: "назван", wantMatchCnt: 1},
		{name: "No match", text: "x-x-x", wantMatchCnt: 0},
		{name: "Empty string", text: "", wantMatchCnt: 0},
	}

	db, err := storageConnect()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := truncatePosts(db)
		if err != nil {
			t.Errorf("unexpected error clearing posts table: %v", err)
		}

		db.Close()
	})

	testPosts, err := memdb.LoadTestPosts(testPostsPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i, post := range testPosts {
		testPosts[i].ID, err = db.AddPost(ctx, post)
		if err != nil {
			t.Fatalf("unexpected error while populating DB: %v", err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts, _, err := db.FilterPosts(ctx, tt.text, 1, 100)
			if err != nil {
				t.Fatalf("FilterPosts() returned error: %v", err)
			}
			if len(posts) != tt.wantMatchCnt {
				t.Errorf("want posts %d, got %d", tt.wantMatchCnt, len(posts))
			}
		})
	}
}

func TestStore_Post(t *testing.T) {
	db, err := storageConnect()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := truncatePosts(db)
		if err != nil {
			t.Errorf("unexpected error clearing posts table: %v", err)
		}

		db.Close()
	})

	testPosts, err := memdb.LoadTestPosts(testPostsPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i, post := range testPosts {
		testPosts[i].ID, err = db.AddPost(ctx, post)
		if err != nil {
			t.Fatalf("unexpected error while populating DB: %v", err)
		}
	}

	targetPost := testPosts[0]

	gotPost, err := db.Post(ctx, targetPost.ID)
	if err != nil {
		t.Errorf("unexpected error retrieving post %v from DB: %v", targetPost.ID, err)
	}
	if !reflect.DeepEqual(gotPost, targetPost) {
		t.Errorf("want post\n%+v\ngot post\n%+v\n", targetPost, gotPost)
	}
}

func TestStore_PostNotExist(t *testing.T) {
	db, err := storageConnect()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	wantErr := storage.ErrPostNotFound
	targetPostID, err := uuid.FromString("01234567-89ab-cdef-0123-456789abcdef")
	if err != nil {
		t.Fatalf("unexpected error while parsing UUID: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	post, gotErr := db.Post(ctx, targetPostID)
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("want error %v, got %v", wantErr, gotErr)
	}
	if !reflect.DeepEqual(post, storage.Post{}) {
		t.Errorf("want empty post, got post %+v", post)
	}
}
