package asana

import (
	"net/url"
	"strconv"
)

// Pagination models Asana's limit/offset query parameters.
type Pagination struct {
	Limit  int
	Offset string
}

// QueryParams renders the pagination as URL query parameters, omitting
// fields that are unset.
func (p Pagination) QueryParams() url.Values {
	values := url.Values{}
	if p.Limit > 0 {
		values.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Offset != "" {
		values.Set("offset", p.Offset)
	}
	return values
}

// NextPage is the pagination cursor Asana returns alongside a page of results.
type NextPage struct {
	Offset string `json:"offset"`
	Path   string `json:"path"`
	URI    string `json:"uri"`
}

// paginatedResponse is the shape shared by every paginated Asana list
// endpoint: a page of data plus an optional cursor to the next page.
type paginatedResponse[T any] struct {
	Data     []T       `json:"data"`
	NextPage *NextPage `json:"next_page"`
}

// User is the shape of a single element in the GET /users response data array.
type User struct {
	GID          string `json:"gid"`
	ResourceType string `json:"resource_type"`
	Name         string `json:"name"`
	Email        string `json:"email"`
}

// Project is the shape of a single element in the GET /projects response data array.
type Project struct {
	GID          string `json:"gid"`
	ResourceType string `json:"resource_type"`
	Name         string `json:"name"`
}
