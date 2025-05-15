package postgres

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"news/pkg/storage"
)

type Store struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, conStr string) (*Store, error) {
	db, err := pgxpool.Connect(ctx, conStr)
	if err != nil {
		return nil, err
	}
	s := Store{
		db: db,
	}

	return &s, nil
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *Store) Close() {
	s.db.Close()
}

// AddPost inserts a single post into the database or updates it if a post with the same ID already exists.
// The post ID is generated as a UUIDv5 based on the post's Link.
// The method returns the ID of the inserted or updated post and an error if any occurs.
func (s *Store) AddPost(ctx context.Context, post storage.Post) (id uuid.UUID, err error) {
	post.ID = uuid.NewV5(uuid.NamespaceURL, post.Link)
	err = s.db.QueryRow(ctx, `
		INSERT INTO posts (id, title, content, published, link)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id)
		DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			published = EXCLUDED.published,
			link = EXCLUDED.link
		RETURNING id
	`,
		post.ID,
		post.Title,
		post.Content,
		post.Published,
		post.Link,
	).Scan(&id)
	if err != nil {
		return
	}

	return
}

// AddPosts inserts or updates a batch of posts in the database within a single transaction.
// For each post, it generates a UUIDv5 based on the post's Link to use as the ID.
// If a post with the same ID already exists, it updates the existing record with the new data.
// Returns an error if beginning the transaction, executing the batch, or committing fails.
func (s *Store) AddPosts(ctx context.Context, posts []storage.Post) (err error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	batch := new(pgx.Batch)
	for _, post := range posts {
		post.ID = uuid.NewV5(uuid.NamespaceURL, post.Link)
		batch.Queue(`
			INSERT INTO posts (id, title, content, published, link)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id)
			DO UPDATE SET
				title = EXCLUDED.title,
				content = EXCLUDED.content,
				published = EXCLUDED.published,
				link = EXCLUDED.link
		`,
			post.ID,
			post.Title,
			post.Content,
			post.Published,
			post.Link,
		)
	}

	res := tx.SendBatch(ctx, batch)
	err = res.Close()
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// LatestPosts returns a paginated list of posts ordered by published date descending.
// It accepts the page number and the number of items per page as parameters.
// If page or limit are less than or equal to zero, they default to 1 and 10 respectively.
// The method returns the posts for the requested page, the total number of pages available,
// and any error encountered during the database queries.
func (s *Store) LatestPosts(ctx context.Context, page, limit int) (posts []storage.Post, numPages int, err error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	rows, err := s.db.Query(ctx, `
        SELECT id, title, content, published, link
        FROM posts
        ORDER BY published DESC
        LIMIT $1 OFFSET $2
    `, limit,
		offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var p storage.Post
		err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Content,
			&p.Published,
			&p.Link)
		if err != nil {
			return nil, 0, err
		}
		p.Published = p.Published.UTC()
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var totalPosts int
	err = s.db.QueryRow(ctx, `SELECT COUNT(id) FROM posts`).Scan(&totalPosts)
	if err != nil {
		return nil, 0, err
	}

	numPages = (totalPosts + limit - 1) / limit
	return
}

// FilterPosts returns a paginated list of posts whose titles contain the given substring.
// If the substring is empty, it returns an empty list without error.
// It returns the list of matching posts for the specified page and limit,
// along with the total number of pages available for the given filter and page size.
// Returns an error if any occurs.
func (s *Store) FilterPosts(ctx context.Context, contains string, page, limit int) ([]storage.Post, int, error) {
	if contains == "" {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	rows, err := s.db.Query(ctx, `
		SELECT id, title, content, published, link
		FROM posts
		WHERE title ILIKE $1
		LIMIT $2 OFFSET $3
	`,
		"%"+contains+"%",
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []storage.Post
	for rows.Next() {
		var p storage.Post
		err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Content,
			&p.Published,
			&p.Link)
		if err != nil {
			return nil, 0, err
		}
		p.Published = p.Published.UTC()
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var totalPosts int
	err = s.db.QueryRow(ctx, `
	SELECT COUNT(id) FROM posts WHERE title ILIKE $1
	`,
		"%"+contains+"%",
	).Scan(&totalPosts)
	if err != nil {
		return nil, 0, err
	}

	numPages := (totalPosts + limit - 1) / limit
	return posts, numPages, nil
}

// Post retrieves a post by its ID. It returns the post and an error if any occurs.
func (s *Store) Post(ctx context.Context, id uuid.UUID) (post storage.Post, err error) {
	err = s.db.QueryRow(ctx, `
		SELECT id, title, content, published, link
		FROM posts
		WHERE id = $1
	`,
		id,
	).Scan(
		&post.ID,
		&post.Title,
		&post.Content,
		&post.Published,
		&post.Link,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = storage.ErrPostNotFound
		}
		return
	}

	post.Published = post.Published.UTC()
	return
}
