package models

import (
	"time"

	"github.com/gofrs/uuid"
)

type Comment struct {
	ID        uuid.UUID `bson:"_id" json:"id"`
	PostID    uuid.UUID `bson:"post_id" json:"post_id"`
	ParentID  uuid.UUID `bson:"parent_id,omitempty" json:"parent_id,omitempty"`
	Author    string    `bson:"author" json:"author"`
	Text      string    `bson:"text" json:"text"`
	Published time.Time `bson:"published" json:"published"`
}
