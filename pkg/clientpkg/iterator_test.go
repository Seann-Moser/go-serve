package clientpkg

import (
	"context"
	"net/http"
	"testing"
)

type Book struct {
	AccessType       string `json:"access_type"`
	BookId           string `json:"book_id"`
	BookName         string `json:"book_name"`
	CoverImage       string `json:"cover_image"`
	CreatedTimestamp string `json:"created_timestamp"`
	Description      string `json:"description"`
	Dislikes         int    `json:"dislikes"`
	Downloads        int    `json:"downloads"`
	Followers        int    `json:"followers"`
	Likes            int    `json:"likes"`
	MinAge           int    `json:"min_age"`
	Public           bool   `json:"public"`
	TotalPages       int    `json:"total_pages"`
	UpdatedTimestamp string `json:"updated_timestamp"`
	Views            int    `json:"views"`
}

func TestIterator(t *testing.T) {
	c, err := New("https://auth.mnlib.com", "book", 9, true, &http.Client{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	it := NewIterator[Book](context.Background(), c, RequestData{
		Path:   "/book/list",
		Method: http.MethodGet,
	})
	//it.Current()
	for it.Next() {
		println(it.Current().BookName)
	}
}
