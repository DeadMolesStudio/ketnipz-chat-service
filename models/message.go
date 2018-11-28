package models

import (
	"time"
)

//easyjson:json
type Message struct {
	ID       uint      `json:"id,omitempty" db:"message_id"`
	Author   *uint     `json:"author,omitempty" db:"author_id"`
	To       *uint     `json:"to,omitempty" db:"to_user"`
	Created  time.Time `json:"created"`
	IsEdited bool      `json:"is_edited" db:"is_edited"`
	Message  string    `json:"message" db:"message_text"`
}

type Messages struct {
	Msgs *[]Message `json:"messages"`
}
