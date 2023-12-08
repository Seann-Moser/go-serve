package clientpkg

import (
	"context"
	"encoding/json"

	"github.com/Seann-Moser/go-serve/pkg/pagination"
)

type Request func(ctx context.Context, data RequestData, p *pagination.Pagination) *ResponseData

type Iterator[T any] struct {
	ctx     context.Context
	request Request
	err     error

	current     *T
	currentItem int
	currentPage uint

	totalItems   int
	offset       int
	currentPages []*T
	singlePage   bool

	RequestData RequestData
	message     string
}

func NewIterator[T any](ctx context.Context, request Request, data RequestData) *Iterator[T] {
	it := &Iterator[T]{
		ctx:          ctx,
		request:      request,
		currentPages: make([]*T, 0),
		RequestData:  data,
	}
	it.getPages()
	return it
}

func (i *Iterator[T]) Current() *T {
	if i.current == nil {
		if len(i.currentPages) == 0 {
			if !i.getPages() {
				return nil
			}
		}
		if i.currentItem-i.offset >= len(i.currentPages) {
			return nil
		}
		i.current = i.currentPages[i.currentItem-i.offset]
	}
	return i.current
}

func (i *Iterator[T]) Message() string {
	return i.message
}
func (i *Iterator[T]) Err() error {
	return i.err
}

func (i *Iterator[T]) FullList() ([]*T, error) {
	var fullList []*T
	fullList = append(fullList, i.currentPages...)
	for i.Next() {
		current := i.Current()
		if current != nil {
			fullList = append(fullList, current)
		}
	}
	if i.Err() != nil {
		return nil, i.Err()
	}
	return fullList, nil
}

func (i *Iterator[T]) Next() bool {
	if i.singlePage {
		return false
	}
	if i.totalItems == 0 {
		if !i.getPages() {
			return false
		}
		if len(i.currentPages) == 0 {
			return false
		}
		i.current = i.currentPages[i.currentItem-i.offset]
		return true
	}
	if i.currentItem < i.totalItems {
		i.currentItem += 1
		if i.currentItem-i.offset >= len(i.currentPages) {
			if !i.getPages() {
				return false
			}
		}
		if i.currentItem-i.offset >= len(i.currentPages) {
			return false
		}
		i.current = i.currentPages[i.currentItem-i.offset]
		return true
	}
	return false
}

// todo support cookies
func (i *Iterator[T]) getPages() bool {
	data := i.request(i.ctx, i.RequestData, i.nextPage())
	if data.Err != nil {
		i.err = data.Err
		i.message = data.Message
		return false
	} else {
		i.message = data.Message
		if len(data.Data) == 0 {
			return false
		}
		i.err = json.Unmarshal(data.Data, &i.currentPages)
		if i.err != nil {
			var single T
			//logic to read single response
			tmpErr := json.Unmarshal(data.Data, &single)
			if tmpErr != nil {
				return false
			}
			i.err = nil
			i.singlePage = true
			i.currentPages = []*T{&single}
			return true
		}
		i.totalItems = int(data.Page.TotalItems)

		i.offset = int((data.Page.CurrentPage - 1) * data.Page.ItemsPerPage)
	}
	if i.err != nil {
		return false
	}
	return true
}

func (i *Iterator[T]) nextPage() *pagination.Pagination {
	if i.currentPage <= 0 {
		i.currentPage = 1
	}
	page := &pagination.Pagination{
		CurrentPage: i.currentPage,
	}
	i.currentPage += 1
	return page
}
