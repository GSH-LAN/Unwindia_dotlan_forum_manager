package database

import (
	"time"
)

type Result struct {
	Result DotlanForumStatusList
	Error  error
}

type DotlanForumStatusList []DotlanForumStatus

type DotlanForumStatus struct {
	ID                  string    `bson:"_id" json:"unwindiaMatchID"`
	DotlanForumPostID   int       `bson:"dotlanForumPostID" json:"dotlanForumPostID"`
	DotlanForumThreadID int       `bson:"dotlanForumThreadID" json:"dotlanForumThreadID"`
	CreatedAt           time.Time `bson:"createdAt,omitempty"`
	UpdatedAt           time.Time `bson:"updatedAt,omitempty"`
}
