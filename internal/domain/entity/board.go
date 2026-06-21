package entity

import (
	"crypto/md5"
	"encoding/hex"
	"gorm.io/gorm"
	"time"
)

type BoardMessage struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	UserQQ    string    `json:"userQQ" gorm:"index"`
	SenderQQ  string    `json:"senderQQ" gorm:"index"`
	SenderName string   `json:"senderName"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	TimeText  string    `json:"timeText"`
}

func (message *BoardMessage) BeforeCreate(tx *gorm.DB) (err error) {
	if message.ID == "" {
		key := message.Content + message.UserQQ + message.SenderQQ
		hash := md5.Sum([]byte(key))
		message.ID = hex.EncodeToString(hash[:])
	}
	return
}
