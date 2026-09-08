package httpapi

import (
	"context"
	"time"
)

func (s *Server) coreWorker(nodes []string) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-tick.C:
			ctx, cancel := context.WithTimeout(s.ctx, 4*time.Second)
			_ = s.Store.ExpireCoreOperations(ctx)
			for _, node := range nodes {
				if !s.Core.Online(node) {
					continue
				}
				ops, err := s.Store.QueuedCoreOperations(ctx, node)
				if err != nil {
					continue
				}
				for _, op := range ops {
					if ctx.Err() != nil {
						break
					}
					_ = s.dispatchCore(op)
				}
			}
			cancel()
		}
	}
}
