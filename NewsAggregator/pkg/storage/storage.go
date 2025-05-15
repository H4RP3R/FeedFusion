package storage

import (
	"fmt"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gofrs/uuid"
)

var (
	ErrConnectDB       = fmt.Errorf("unable to establish DB connection")
	ErrDBNotResponding = fmt.Errorf("DB not responding")
	ErrPostNotFound    = fmt.Errorf("post not found")
)

type Post struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Published time.Time `json:"published"`
	Link      string    `json:"link"`
}

type Storage interface {
	// AddPost adds a single post to the storage and returns the post ID and an error if any occurs.
	AddPost(post Post) (id uuid.UUID, err error)

	// AddPosts adds multiple posts to the storage and returns an error if any occurs.
	AddPosts(posts []Post) (err error)

	// LatestPosts fetches recent posts in descending order by date.
	// Returns a list of posts, total page count, and an error if any occurs.
	LatestPosts(currentPage, limit int) (posts []Post, numPages int, err error)

	// Post retrieves a post by its ID. It returns the post and an error if any occurs.
	Post(id uuid.UUID) (post Post, err error)

	// FilterPosts returns a list of posts whose titles contain the given substring,
	// total page count and an error if any occurs.
	FilterPosts(contains string, page, limit int) (posts []Post, numPages int, err error)
}

// ValidatePosts accepts a slice of posts and removes the invalid ones, i.e., posts containing any empty fields.
func ValidatePosts(posts ...Post) []Post {
	var validPosts []Post
	for _, p := range posts {
		if p.Title != "" && p.Content != "" && p.Link != "" && !p.Published.IsZero() {
			if _, err := url.ParseRequestURI(p.Link); err == nil {
				validPosts = append(validPosts, p)
			}
		} else {
			log.Warnf("Invalidated post: %+v", p)
		}
	}

	return validPosts
}
