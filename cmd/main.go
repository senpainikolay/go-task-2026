package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/time/rate"

	"recoTask/internal/asana"
	"recoTask/internal/config"
)

func main() {
	infoLog := log.New(os.Stdout, "INFO: ", log.LstdFlags)
	errLog := log.New(os.Stderr, "ERROR: ", log.LstdFlags|log.Lshortfile)

	cfg, err := config.NewConfig()
	if err != nil {
		errLog.Fatalf("load config: %v", err)
	}

	requestsPerSecond := float64(cfg.MaxRequestsPerMinute) / 60
	limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), 5)
	extractor := asana.NewClient(
		// TODO: Remove magic number; Add to ENV Var;
		// + Coordonate with a context timeout
		&http.Client{Timeout: 60 * time.Second},
		cfg.AsanaToken,
		"",
		cfg.DefaultWorkspace,
		limiter,
		cfg.LimiterMaxRetries,
		cfg.PaginationLimit,
	)

	app := NewApp(extractor, infoLog, errLog)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.Run(ctx)
}
