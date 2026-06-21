package entity

import "time"

type User struct {
	QQ              string            `json:"qq" gorm:"primaryKey"`
	Nickname        string            `json:"nickname"`
	AvatarURL       string            `json:"avatarURL"`
	Cookies         map[string]string `json:"cookies" gorm:"serializer:json"`
	LoginStatus     LoginStatus       `json:"loginStatus"`
	LoginExpireTime time.Time         `json:"loginExpireTime"`
}
