package backend

import (
	"context"

	"github.com/bluemir/nemokan/internal/pubsub"
	"github.com/bluemir/nemokan/internal/server/backend/auth"
	"github.com/bluemir/nemokan/internal/server/backend/documents"
	"github.com/bluemir/nemokan/internal/server/backend/jobs"
	"gorm.io/gorm"
)

// Config from file
type Config struct {
	Auth      auth.Config
	Jobs      jobs.Config
	Documents documents.Config
}
type Backends struct {
	Auth      *auth.Manager
	Events    *pubsub.Hub
	Jobs      *jobs.Manager
	Documents *documents.Manager
}

func Initialize(ctx context.Context, conf *Config, db *gorm.DB) (*Backends, error) {
	events, err := pubsub.NewHub(ctx)
	if err != nil {
		return nil, err
	}
	// init components

	authManager, err := auth.New(ctx, &conf.Auth, db, events)
	if err != nil {
		return nil, err
	}
	jobs, err := jobs.New(ctx, &conf.Jobs, db, events)
	if err != nil {
		return nil, err
	}

	documents, err := documents.New(ctx, &conf.Documents, db)
	if err != nil {
		return nil, err
	}

	return &Backends{
		Events:    events,
		Auth:      authManager,
		Jobs:      jobs,
		Documents: documents,
	}, nil
}
