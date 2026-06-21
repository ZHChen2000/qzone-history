package entity

import "time"

type Activity struct {
	ID         string       `json:"id" gorm:"primaryKey"`
	SenderQQ   string       `json:"senderQQ" gorm:"index"`
	SenderName string       `json:"senderName"`
	SenderLink string       `json:"senderLink"`
	ReceiverQQ string       `json:"receiverQQ" gorm:"index"`
	Content    string       `json:"content"`
	Timestamp  time.Time    `json:"timestamp"`
	TimeText   string       `json:"timeText"`
	ImageURLs  []string     `json:"imageURLs" gorm:"serializer:json"`
	Type       ActivityType `json:"type"`
}

type ActivityType int

const (
	TypeMoment       ActivityType = iota
	TypeForward
	TypeLike
	TypeComment
	TypeBoardMessage
	TypeBoardReply
	TypeReply
	TypeView
	TypeOther
)
