//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

func TestCoreCancellationRecoveryValidatesProofWithoutRepeatingMutation(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "committed", true: "wrong_receipt"}[corrupt], func(t *testing.T) {
			db := testdb.New(t)
			c, node, template, proof := cancellationFixtureV2()
			server := New(db, c)
			t.Cleanup(server.Close)
			ctx := context.Background()
			op, _, err := db.CreateCoreOperation(ctx, "buyer", "cancel-request", node.ID, template.Command, template.Payload)
			if err != nil {
				t.Fatal(err)
			}
			proof["operationId"] = op.ID
			if corrupt {
				proof["snapshotSha256"] = "different"
			}
			if _, err = db.MarkCoreSent(ctx, op.ID); err != nil {
				t.Fatal(err)
			}
			if err = db.MarkCoreUnknown(ctx, op.ID); err != nil {
				t.Fatal(err)
			}
			calls := 0
			server.Core.Connect(node.ID, func(ctx context.Context, _ string, input any) error {
				calls++
				frame := input.(map[string]any)
				if frame["command"] != "operation.query" {
					t.Fatal("repeated mailbox mutation", frame["command"])
				}
				recovered, _ := json.Marshal(map[string]any{"operationId": op.ID, "status": "COMPLETED", "data": map[string]any{"code": "OK", "value": proof}})
				reply, _ := json.Marshal(map[string]any{"operationId": frame["operationId"], "status": "COMPLETED", "data": map[string]any{"acquired": false, "state": "COMPLETED", "result": string(recovered)}})
				id := frame["operationId"].(string)
				err := db.CoreReply(ctx, node.ID, id, "COMPLETED", reply)
				server.Core.Notify(id)
				return err
			})
			_, err = server.QueryCoreOperation(ctx, op.ID)
			if (err != nil) != corrupt {
				t.Fatal("proof validation result", err)
			}
			current, err := db.CoreOperation(ctx, op.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := "COMPLETED"
			if corrupt {
				want = "UNKNOWN"
			}
			if current.State != want || calls != 1 {
				t.Fatal(current.State, calls)
			}
			var mails int
			if err = db.DB.QueryRow("SELECT COUNT(*) FROM core_mail_receipts").Scan(&mails); err != nil {
				t.Fatal(err)
			}
			if mails != 0 {
				t.Fatal("fabricated mail receipt")
			}
		})
	}
}
