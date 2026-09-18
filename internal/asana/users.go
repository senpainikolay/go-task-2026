package asana

import "context"

// ExtractUsers fetches every page of GET /users, following next_page until exhausted.
func (a *Client) ExtractUsers(ctx context.Context) ([]User, error) {
	return GetAllPages[User](ctx, a, "/users", Pagination{Limit: a.paginationLimit})
}
