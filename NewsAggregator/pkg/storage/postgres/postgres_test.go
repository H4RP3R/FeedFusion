package postgres

import (
	"context"
	"errors"
	"news/pkg/storage"
	"news/pkg/storage/memdb"
	"os"
	"reflect"
	"testing"

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
	db, err := New(conf.ConString())
	if err != nil {
		return nil, storage.ErrConnectDB
	}

	err = db.Ping()
	if err != nil {
		return nil, storage.ErrDBNotResponding
	}

	return db, nil
}

// truncatePosts restores the original state of DB for further testing.
func truncatePosts(db *Store) error {
	_, err := db.db.Exec(context.Background(), "TRUNCATE TABLE posts")
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

	for i, post := range testPosts {
		testPosts[i].ID, err = db.AddPost(post)
		if err != nil {
			t.Errorf("unexpected error while adding post: %v", err)
		}
	}

	for _, post := range testPosts {
		var gotPost storage.Post
		err := db.db.QueryRow(context.Background(), `
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

	err = db.AddPosts(testPosts)
	if err != nil {
		t.Errorf("unexpected error while adding multiple posts: %v", err)
	}

	rows, err := db.db.Query(context.Background(), `
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

func TestStore_Posts(t *testing.T) {
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

	for i, post := range testPosts {
		testPosts[i].ID, err = db.AddPost(post)
		if err != nil {
			t.Fatalf("unexpected error while populating DB: %v", err)
		}
	}

	for n := 1; n < len(testPosts); n++ {
		posts, err := db.Posts(n)
		if err != nil {
			t.Errorf("unexpected error retrieving %d posts from DB", n)
		}
		wantPosts := testPosts[:n]
		if !reflect.DeepEqual(posts, wantPosts) {
			t.Errorf("want posts\n%+v\ngot posts\n%+v\n", wantPosts, posts)
		}
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

	for i, post := range testPosts {
		testPosts[i].ID, err = db.AddPost(post)
		if err != nil {
			t.Fatalf("unexpected error while populating DB: %v", err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts, err := db.FilterPosts(tt.text)
			if err != nil {
				t.Errorf("FilterPosts() returned error: %v", err)
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

	for i, post := range testPosts {
		testPosts[i].ID, err = db.AddPost(post)
		if err != nil {
			t.Fatalf("unexpected error while populating DB: %v", err)
		}
	}

	targetPost := testPosts[0]

	gotPost, err := db.Post(targetPost.ID)
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
	post, gotErr := db.Post(uuid.FromStringOrNil("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"))
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("want error %v, got %v", wantErr, gotErr)
	}
	if !reflect.DeepEqual(post, storage.Post{}) {
		t.Errorf("want empty post, got post %+v", post)
	}
}
