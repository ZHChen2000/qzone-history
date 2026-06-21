package entity

import "time"

type Friend struct {
	UserQQ    string    `json:"userQQ" gorm:"primaryKey"`
	FriendQQ  string    `json:"friendQQ" gorm:"primaryKey"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatarURL"`
	AddedTime time.Time `json:"addedTime"`
}
