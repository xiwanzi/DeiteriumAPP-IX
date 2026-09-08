package httpapi

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
)

// Enabled only after the project's gc/ prefix and lifecycle policy have been checked.
func (s *Server) StartAssetLifecycleV2() {
	if os.Getenv("DEUTERIUM_ASSET_GC_ENABLED") != "true" {
		return
	}
	objects, err := objectstorage.FromEnvironment()
	if err != nil {
		slog.Warn("asset maintenance unavailable")
		return
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
			result, err := s.Store.RunAssetLifecycleV2(ctx, objects, objects.Config.Prefix, 32)
			cancel()
			if err != nil && s.ctx.Err() == nil {
				slog.Warn("asset maintenance deferred")
			}
			if result.Retired > 0 || result.Purged > 0 || result.Deferred > 0 {
				slog.Info("asset maintenance", "retired", result.Retired, "purged", result.Purged, "deferred", result.Deferred)
			}
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
