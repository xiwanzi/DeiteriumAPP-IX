package httpapi

import "context"

// CommerceExecutorV2 is supplied by the production Core adapter. A nil executor
// or unavailable command rejects a new payment before any work is queued.
// Every retry uses the same actor and clientRequestID; uncertain results are
// reconciled by Query and must never become a second reserve/settle/refund.
type CommerceExecutorV2 interface {
	Available(command string) bool
	Execute(ctx context.Context, actorID, clientRequestID, command string, payload map[string]any) (CommerceCoreResultV2, error)
	Query(ctx context.Context, actorID, operationID string) (CommerceCoreResultV2, error)
}

type CommerceCoreResultV2 struct {
	OperationID string         `json:"operationId"`
	Status      string         `json:"status"`
	Data        map[string]any `json:"data"`
	ErrorCode   string         `json:"errorCode,omitempty"`
}

const (
	CommerceReserveV2 = "wallet.escrow.reserve"
	CommerceBindV2    = "wallet.escrow.bind"
	CommerceSettleV2  = "wallet.escrow.settle"
	CommerceRefundV2  = "wallet.escrow.refund"
)
