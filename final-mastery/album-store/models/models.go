package models

type Album struct {
	AlbumID     string `json:"album_id" db:"album_id"`
	Title       string `json:"title" db:"title"`
	Description string `json:"description" db:"description"`
	Owner       string `json:"owner" db:"owner"`
}

type Photo struct {
	PhotoID string  `json:"photo_id" db:"photo_id"`
	AlbumID string  `json:"album_id" db:"album_id"`
	Seq     int     `json:"seq" db:"seq"`
	Status  string  `json:"status" db:"status"`
	URL     *string `json:"url,omitempty" db:"url"`
}