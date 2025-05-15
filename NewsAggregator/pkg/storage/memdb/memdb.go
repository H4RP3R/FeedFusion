package memdb

import (
	"context"
	"sort"
	"sync"

	"news/pkg/storage"

	"github.com/gofrs/uuid"
)

type Store struct {
	mu    sync.Mutex
	posts map[uuid.UUID]storage.Post
}

func New() *Store {
	db := Store{
		posts: make(map[uuid.UUID]storage.Post),
	}

	return &db
}

func (db *Store) AddPost(ctx context.Context, post storage.Post) (id uuid.UUID, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	post.ID = uuid.NewV5(uuid.NamespaceURL, post.Link)
	db.posts[post.ID] = post

	return post.ID, nil
}

func (db *Store) AddPosts(ctx context.Context, posts []storage.Post) (err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, post := range posts {
		post.ID = uuid.NewV5(uuid.NamespaceURL, post.Link)
		db.posts[post.ID] = post
	}

	return
}

func (db *Store) LatestPosts(ctx context.Context, page, limit int) (posts []storage.Post, numPages int, err error) {
	if limit <= 0 {
		return []storage.Post{}, 0, nil
	}

	db.mu.Lock()
	allPosts := make([]storage.Post, 0, len(db.posts))
	for _, v := range db.posts {
		allPosts = append(allPosts, v)
	}
	db.mu.Unlock()

	sort.Slice(allPosts, func(i, j int) bool {
		return allPosts[i].Published.After(allPosts[j].Published)
	})

	totalPosts := len(allPosts)
	numPages = (totalPosts + limit - 1) / limit

	pageIndex := page - 1
	if pageIndex < 0 {
		pageIndex = 0
	}

	start := pageIndex * limit
	if start >= totalPosts {
		return []storage.Post{}, numPages, nil
	}

	end := start + limit
	if end > totalPosts {
		end = totalPosts
	}

	return allPosts[start:end], numPages, nil
}

func (db *Store) FilterPosts(ctx context.Context, contains string, page, limit int) (posts []storage.Post, numPages int, err error) {
	return
}

func (db *Store) Post(ctx context.Context, id uuid.UUID) (post storage.Post, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	post, ok := db.posts[id]
	if !ok {
		return storage.Post{}, storage.ErrPostNotFound
	}

	return post, nil
}
