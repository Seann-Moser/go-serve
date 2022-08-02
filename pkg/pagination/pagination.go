package pagination

import (
	"net/http"
	"strconv"
)

const MaxItemsPerPage = 1000

type Pagination struct {
	CurrentPage  uint `json:"current_page"`
	NextPage     uint `json:"next_page"`
	TotalItems   uint `json:"total_items"`
	TotalPages   uint `json:"total_pages"`
	ItemsPerPage uint `json:"items_per_page"`
}

func GeneratePagination(r *http.Request) *Pagination {
	p := &Pagination{
		CurrentPage:  1,
		NextPage:     0,
		TotalItems:   0,
		TotalPages:   0,
		ItemsPerPage: 0,
	}
	q := r.URL.Query()
	if currentPage := q.Get("page"); currentPage != "" {
		if v, err := strconv.Atoi(currentPage); err == nil {
			p.CurrentPage = uint(v)
		}
	}
	if itemsPerPage := q.Get("items_per_page"); itemsPerPage != "" {
		if v, err := strconv.Atoi(itemsPerPage); err == nil {
			p.ItemsPerPage = uint(v)
		}
	}
	if p.ItemsPerPage > MaxItemsPerPage || p.ItemsPerPage == 0 {
		p.ItemsPerPage = MaxItemsPerPage
	}
	return p
}
