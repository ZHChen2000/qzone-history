package entity

import (
	"crypto/md5"
	"encoding/hex"
	"gorm.io/gorm"
	"time"
)

type Moment struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	UserQQ          string    `json:"userQQ" gorm:"index"`
	SenderQQ        string    `json:"senderQQ" gorm:"index"`
	Content         string    `json:"content"`
	Timestamp       time.Time `json:"timestamp"`
	TimeText        string    `json:"timeText"`
	ImageURLs       []string  `json:"imageURLs" gorm:"serializer:json"`
	Likes           int       `json:"likes"`
	Comments        []Comment `json:"comments" gorm:"foreignKey:MomentID"`
	Views           int       `json:"views"`
	IsDeleted       bool      `json:"isDeleted"`
	IsReconstructed bool      `json:"isReconstructed"`
}

func (moment *Moment) BeforeCreate(tx *gorm.DB) (err error) {
	if moment.ID == "" {
		key := moment.Content + moment.UserQQ
		hash := md5.Sum([]byte(key))
		moment.ID = hex.EncodeToString(hash[:])
	}
	return
}
