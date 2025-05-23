package models

import (
	"time"

	"github.com/gofrs/uuid"
)

type Post struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Published time.Time `json:"published"`
	Link      string    `json:"link"`
	Comments  []Comment `json:"comments,omitempty"`
}

type Preview struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Published time.Time `json:"published"`
	Link      string    `json:"link"`
}

type Comment struct {
	ID        uuid.UUID  `json:"id"`
	PostID    uuid.UUID  `json:"post_id"`
	ParentID  uuid.UUID  `json:"parent_id,omitempty"`
	Author    string     `json:"author"`
	Text      string     `json:"text"`
	Published time.Time  `json:"published"`
	Replies   []*Comment `json:"replies,omitempty"`
}
