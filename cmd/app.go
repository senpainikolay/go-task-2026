package main

import (
	"context"
	"log"
	"time"

	"recoTask/internal/storage"
)

const (
	// We want to choose intervals wisely to not load up the Rate Limiter
	// TODO: Add as ENV VARS
	usersTickInterval    = 30 * time.Second // 1 * time.Second
	projectsTickInterval = 5 * time.Minute  // 1 * time.Second
)

type App struct {
	extractor Extractor
	infoLog   *log.Logger
	errLog    *log.Logger
}

func NewApp(extractor Extractor, infoLog, errLog *log.Logger) *App {
	return &App{extractor: extractor, infoLog: infoLog, errLog: errLog}
}

// Run ticks users and projects extraction independently, each on its own
// interval, until ctx is cancelled.
func (a *App) Run(ctx context.Context) {
	usersTicker := time.NewTicker(usersTickInterval)
	defer usersTicker.Stop()

	projectsTicker := time.NewTicker(projectsTickInterval)
	defer projectsTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.infoLog.Println("shutting down")
			return
		case <-usersTicker.C:
			go a.extractUsers(ctx)
		case <-projectsTicker.C:
			go a.extractProjects(ctx)
		}
	}
}

func (a *App) extractUsers(ctx context.Context) {
	users, err := a.extractor.ExtractUsers(ctx)
	if err != nil {
		a.errLog.Printf("extract users: %v", err)
		return
	}
	for _, u := range users {
		if err := storage.WriteJSON("out/users", u.GID+".json", u); err != nil {
			a.errLog.Printf("write user %s: %v", u.GID, err)
		}
	}
	a.infoLog.Printf("wrote %d user(s) to out/users", len(users))
}

func (a *App) extractProjects(ctx context.Context) {
	projects, err := a.extractor.ExtractProjects(ctx)
	if err != nil {
		a.errLog.Printf("extract projects: %v", err)
		return
	}
	for _, p := range projects {
		if err := storage.WriteJSON("out/projects", p.GID+".json", p); err != nil {
			a.errLog.Printf("write project %s: %v", p.GID, err)
		}
	}
	a.infoLog.Printf("wrote %d project(s) to out/projects", len(projects))
}
