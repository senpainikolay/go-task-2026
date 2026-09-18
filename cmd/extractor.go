package main

import (
	"context"

	"recoTask/internal/asana"
)

type Extractor interface {
	ExtractUsers(ctx context.Context) ([]asana.User, error)
	ExtractProjects(ctx context.Context) ([]asana.Project, error)
}
