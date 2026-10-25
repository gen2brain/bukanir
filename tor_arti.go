//go:build !systemtor

package bukanir

import (
	"context"

	arti "github.com/gen2brain/go-arti"
)

// SystemTor reports whether Tor goes through the system daemon instead of the embedded client.
const SystemTor = false

// newTorProvider builds a bootstrapped pool of embedded Tor clients.
var newTorProvider = func(ctx context.Context, cfg Config, cacheDir string, instances int) (torProvider, error) {
	opts := []arti.Option{arti.WithCircuits(DefaultTorCircuits)}
	if cfg.TorLog != nil {
		opts = append(opts, arti.WithStderr(cfg.TorLog))
	}
	return arti.NewPool(ctx, cacheDir, instances, opts...)
}
