package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/zanuka/vastion-api/internal/config"
	"github.com/zanuka/vastion-api/internal/repository/mongodb"
	"github.com/zanuka/vastion-api/internal/router"
	"github.com/zanuka/vastion-api/internal/service"
)

type App struct {
	server *http.Server
	client *mongodb.Client
}

func New(cfg config.Config) (*App, error) {
	client, err := mongodb.Connect(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx); err != nil {
		slog.Warn("mongo ping", "err", err)
	}

	idxCtx, idxCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer idxCancel()
	if err := client.EnsureIndexes(idxCtx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	h := router.New(router.Deps{
		Health:      service.NewHealth(client),
		Detections:  service.NewDetections(mongodb.NewDetectionRepository(client), mongodb.NewAcknowledgementRepository(client)),
		Sites:       service.NewSites(mongodb.NewSiteRepository(client)),
		CORSOrigins: cfg.CORSOrigins,
		Logger:      slog.Default(),
	})

	return &App{
		server: &http.Server{
			Addr:              ":" + cfg.Port,
			Handler:           h,
			ReadHeaderTimeout: 5 * time.Second,
		},
		client: client,
	}, nil
}

func (a *App) Serve() error {
	slog.Info("listen", "addr", a.server.Addr)
	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *App) Close(ctx context.Context) error {
	var serveErr error
	if a.server != nil {
		serveErr = a.server.Shutdown(ctx)
	}
	if a.client != nil {
		if err := a.client.Disconnect(ctx); err != nil && serveErr == nil {
			return err
		}
	}
	return serveErr
}
