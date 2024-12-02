package proxyutil

import (
	"log/slog"

	"github.com/planetscale/psdb/auth"
	"github.com/planetscale/psdbproxy"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

type Config struct {
	Logger       *zap.Logger
	LocalAddr    string
	UpstreamAddr string
	Username     string
	Password     string
}

func New(cfg Config) *psdbproxy.Server {
	return &psdbproxy.Server{
		Addr:          cfg.LocalAddr,
		Logger:        slog.New(zapslog.NewHandler(cfg.Logger.Core())),
		UpstreamAddr:  cfg.UpstreamAddr,
		Authorization: auth.NewBasicAuth(cfg.Username, cfg.Password),
	}
}
