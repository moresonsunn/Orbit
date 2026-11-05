//go:build !linux || !cgo

package gpu

import (
	"context"
	"log/slog"
)

type NVMLProvider struct {
	log *slog.Logger
}

func NewNVMLProvider(log *slog.Logger) *NVMLProvider {
	return &NVMLProvider{log: log}
}

func (p *NVMLProvider) Name() string {
	return "nvidia"
}

func (p *NVMLProvider) Available() bool {
	return false
}

func (p *NVMLProvider) Shutdown() {}

func (p *NVMLProvider) Collect(ctx context.Context) ([]DeviceMetrics, error) {
	return nil, nil
}
