package api

import "news/pkg/storage"

type Pagination struct {
	TotalPages  int `json:"total_pages"`
	CurrentPage int `json:"current_page"`
	Limit       int `json:"limit"`
}

type PostsResponse struct {
	Posts      []storage.Post `json:"posts"`
	Pagination Pagination     `json:"pagination"`
}
