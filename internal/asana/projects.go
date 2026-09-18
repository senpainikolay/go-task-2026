package asana

import "context"

// ExtractProjects fetches every page of GET /projects, following next_page until exhausted.
func (a *Client) ExtractProjects(ctx context.Context) ([]Project, error) {
	return GetAllPages[Project](ctx, a, "/projects", Pagination{Limit: a.paginationLimit})
}
