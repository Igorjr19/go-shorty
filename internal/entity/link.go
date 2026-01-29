package entity

import "time"

type Link struct {
	Code        string
	OriginalURL string
	CreatedAt   time.Time
}
