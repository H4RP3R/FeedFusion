package api

import "gateway/pkg/models"

type Pagination struct {
	TotalPages  int `json:"total_pages"`
	CurrentPage int `json:"current_page"`
	Limit       int `json:"limit"`
}

type PostsResponse struct {
	Posts      []models.Post `json:"posts"`
	Pagination Pagination    `json:"pagination"`
}
