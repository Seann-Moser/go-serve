package pagination

import (
	"encoding/json"
	"github.com/tidwall/gjson"
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

func New(body []byte, r *http.Request) *Pagination {
	if body == nil {
		return GeneratePagination(r)
	}

	results := gjson.GetBytes(body, "page")
	if !results.IsObject() {
		return GeneratePagination(r)
	}
	page := Pagination{}
	err := json.Unmarshal([]byte(results.String()), &page)
	if err == nil {
		if page.ItemsPerPage > MaxItemsPerPage || page.ItemsPerPage == 0 {
			page.ItemsPerPage = MaxItemsPerPage
		}
		return &page
	}
	return GeneratePagination(r)
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
