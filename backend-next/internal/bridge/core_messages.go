package bridge

import (
	"strings"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func ValidateCatalog(node config.Node, e Envelope) (v store.CoreCatalog, err error) {
	if !ValidMessageID(e.EventID) || Decode(e.Payload, &v) != nil || node.ItemPrefix == "" || !strings.HasPrefix(v.ItemRef, node.ItemPrefix) || !itemRefPattern.MatchString(v.ItemRef) || v.CatalogVersion < 1 || v.CatalogVersion > 2147483647 || v.LatestRevision < 1 || v.LatestRevision > 2147483647 {
		return v, ErrProtocol
	}
	return
}
func ValidateMail(c config.Config, node config.Node, e Envelope) (v store.MailEvent, err error) {
	if node.MailCluster == "" || Decode(e.Payload, &v) != nil || !identity.ValidUUID(v.EventID) || v.ClusterID != node.MailCluster || e.EventID != "mail_"+v.EventID || v.OccurredAt <= 0 {
		return v, ErrProtocol
	}
	kind := strings.TrimSuffix(v.Type, ".event") + ".event"
	if kind != e.Type {
		return v, ErrProtocol
	}
	v.Type = kind
	if kind != "mailbox.created.event" && kind != "mailbox.claimed.event" && kind != "mailbox.revoked.event" && kind != "mailbox.uncertain.event" {
		return v, ErrProtocol
	}
	if err = ValidateReceipt(c, v.Receipt); err != nil {
		return
	}
	if kind == "mailbox.claimed.event" && (v.Receipt.Status != "CLAIMED" || !c.HasNode(v.ServerID) || v.SaveReceipt == "" || v.PlayerSessionEpoch == "" || v.ClaimOperationID == "") {
		return v, ErrProtocol
	}
	return
}
func ValidateReceipt(c config.Config, r store.MailReceipt) error {
	if !ValidMessageID(r.DeliveryID) || !ValidMessageID(r.OrderID) || r.Source != "deuterium-commerce" || !sha256Pattern.MatchString(r.SnapshotSHA256) || !identity.ValidUUID(r.RecipientUUID) || r.Revision < 1 || len(r.AllowedServerIDs) == 0 || len(r.AllowedServerIDs) > 32 || !config.NodeID.MatchString(r.InventoryDomain) {
		return ErrProtocol
	}
	if r.Status != "CREATED" && r.Status != "CLAIMING" && r.Status != "CLAIMED" && r.Status != "REVOKED" && r.Status != "FAILED" && r.Status != "UNKNOWN" {
		return ErrProtocol
	}
	seen := map[string]bool{}
	for _, id := range r.AllowedServerIDs {
		if seen[id] || !c.HasNode(id) {
			return ErrProtocol
		}
		seen[id] = true
	}
	return nil
}
