package postgres

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"

	"news/pkg/storage"
)

type Store struct {
	db *pgxpool.Pool
}

func New(conStr string) (*Store, error) {
	db, err := pgxpool.Connect(context.Background(), conStr)
	if err != nil {
		return nil, err
	}
	s := Store{
		db: db,
	}

	return &s, nil
}

func (s *Store) Ping() error {
	return s.db.Ping(context.Background())
}

func (s *Store) Close() {
	s.db.Close()
}

func (s *Store) AddPost(post storage.Post) (id uuid.UUID, err error) {
	post.ID = uuid.NewV5(uuid.NamespaceURL, post.Link)
	err = s.db.QueryRow(context.Background(), `
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
		log.Errorf("error adding post: %v", err)
		return
	}

	log.Infof("post ID:%v added successfully", id)

	return
}

func (s *Store) AddPosts(posts []storage.Post) (err error) {
	ctx := context.Background()
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

func (s *Store) Posts(n int) (posts []storage.Post, err error) {
	rows, err := s.db.Query(context.Background(), `
		SELECT id, title, content, published, link
		FROM posts
		ORDER BY published DESC
		LIMIT $1
	`,
		n,
	)
	if err != nil {
		return nil, err
	}

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
			return nil, err
		}
		p.Published = p.Published.UTC()
		posts = append(posts, p)
	}

	return posts, rows.Err()
}

func (s *Store) FilterPosts(contains string) (posts []storage.Post, err error) {
	rows, err := s.db.Query(context.Background(), `
		SELECT id, title, content, published, link
		FROM posts
		WHERE title ILIKE $1
	`,
		"%"+contains+"%",
	)
	if err != nil {
		return nil, err
	}

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
			return nil, err
		}
		p.Published = p.Published.UTC()
		posts = append(posts, p)
	}

	return
}
