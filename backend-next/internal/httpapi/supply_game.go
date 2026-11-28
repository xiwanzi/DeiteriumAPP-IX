package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

type supplyGameRequest struct {
	PlayerUUID          string                       `json:"playerUuid"`
	SessionEpoch        string                       `json:"sessionEpoch"`
	RequestID           string                       `json:"requestId"`
	Draw                *store.SupplyDrawRequest     `json:"draw,omitempty"`
	Exchange            *store.SupplyExchangeRequest `json:"exchange,omitempty"`
	OrderID             string                       `json:"orderId,omitempty"`
	ExchangeID          string                       `json:"exchangeId,omitempty"`
	Filter              *store.SupplyFilter          `json:"filter,omitempty"`
	TicketCatalog       bool                         `json:"ticketCatalog,omitempty"`
	AwaitReveal         bool                         `json:"awaitReveal,omitempty"`
	PresentationVersion int                          `json:"presentationVersion,omitempty"`
	KnownPools          map[string]int64             `json:"knownPools,omitempty"`
	KnownTicketsDigest  string                       `json:"knownTicketsDigest,omitempty"`
}

func (s *Server) supplyNode(r *http.Request) (config.Node, error) {
	auth := r.Header.Get("Authorization")
	node, ok := s.Config.AuthenticateNode(r.Header.Get("X-Deuterium-Node-ID"), strings.TrimPrefix(auth, "Bearer "))
	if !ok || !strings.HasPrefix(auth, "Bearer ") || !node.Chat || !s.Core.Online(node.ID) {
		return node, store.ErrUnauthorized
	}
	return node, nil
}

func (s *Server) supplyPlayer(node config.Node, uuid, epoch string) (store.CorePlayerIdentity, error) {
	if !gameUUID.MatchString(uuid) || epoch == "" || len(epoch) > 80 {
		return store.CorePlayerIdentity{}, store.ErrUnauthorized
	}
	p, err := s.Core.Player("", uuid)
	if err != nil || !p.Online || p.ServerID != node.ID || p.SessionEpoch != epoch {
		return p, store.ErrUnauthorized
	}
	return p, nil
}

func (s *Server) registerSupplyGame(mux *http.ServeMux) {
	for _, action := range []string{"state", "draw", "reveal-complete", "exchange", "order", "exchange-status", "history", "ledger"} {
		mux.HandleFunc("POST /bridge/v1/supply/"+action, func(w http.ResponseWriter, r *http.Request) {
			node, err := s.supplyNode(r)
			if err != nil {
				failError(w, r, err)
				return
			}
			var in supplyGameRequest
			if socialBodyV2(w, r, &in, 8192) != nil {
				socialFailureV2(w, r, store.ErrSocialInvalid)
				return
			}
			player, err := s.supplyPlayer(node, in.PlayerUUID, in.SessionEpoch)
			if err != nil {
				failError(w, r, err)
				return
			}
			var out any
			switch action {
			case "state":
				out, err = s.supplyGameState(r.Context(), player, in)
			case "draw":
				if in.Draw == nil {
					err = store.ErrSocialInvalid
					break
				}
				in.Draw.AwaitReveal = in.AwaitReveal
				var raw json.RawMessage
				raw, err = s.Store.DrawSupply(r.Context(), player.PlayerUUID, player.GameID, *in.Draw)
				if err == nil {
					var order store.SupplyOrder
					err = json.Unmarshal(raw, &order)
					if err == nil {
						receipt, e := s.supplyPlayerOrder(r.Context(), order.ID, player.PlayerUUID)
						if e == nil && in.AwaitReveal {
							receipt["awaitReveal"] = true
						}
						out, err = receipt, e
					}
				}
			case "reveal-complete":
				if !supply.ValidID(in.OrderID) {
					err = store.ErrSocialInvalid
					break
				}
				err = s.Store.ReleaseSupplyReveal(r.Context(), in.OrderID, player.PlayerUUID)
				if err == nil {
					orderID := in.OrderID
					go func() {
						ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
						defer cancel()
						s.processSupplyOrderMail(ctx, orderID)
					}()
				}
				out = map[string]bool{"released": true}
			case "exchange":
				if in.Exchange == nil {
					err = store.ErrSocialInvalid
					break
				}
				var raw json.RawMessage
				raw, err = s.Store.CreateSupplyExchange(r.Context(), player.PlayerUUID, player.GameID, *in.Exchange)
				if err == nil {
					var x store.SupplyExchange
					err = json.Unmarshal(raw, &x)
					if err == nil {
						attempt, cancel := context.WithTimeout(r.Context(), 2*time.Second)
						if e := s.processSupplyExchange(attempt, x.ID, player.PlayerUUID); e != nil && attempt.Err() == nil {
							slog.Warn("Supply exchange awaiting recovery", "exchange", x.ID, "error", e)
						}
						cancel()
						out, err = s.supplyPlayerExchange(r.Context(), x.ID, player.PlayerUUID)
					}
				}
			case "exchange-status":
				out, err = s.supplyPlayerExchange(r.Context(), in.ExchangeID, player.PlayerUUID)
			case "order":
				out, err = s.supplyPlayerOrder(r.Context(), in.OrderID, player.PlayerUUID)
			case "history", "ledger":
				f := store.SupplyFilter{Limit: 25}
				if in.Filter != nil {
					f = *in.Filter
				}
				f.PlayerUUID = player.PlayerUUID
				if action == "ledger" {
					out, err = s.Store.SupplyLedger(r.Context(), f)
				} else {
					var page map[string]any
					page, err = s.Store.SupplyOrders(r.Context(), f)
					if err == nil {
						items := []any{}
						for _, order := range page["items"].([]store.SupplyOrder) {
							items = append(items, publicSupplyOrder(order))
						}
						page["items"] = items
						out = page
					}
				}
			}
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			success(w, r, out)
		})
	}
	s.registerSupplyDeliveryBridge(mux)
	mux.HandleFunc("POST /bridge/v1/supply/tickets", func(w http.ResponseWriter, r *http.Request) {
		node, err := s.supplyNode(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		var in struct {
			OperatorUUID string                  `json:"operatorUuid"`
			SessionEpoch string                  `json:"sessionEpoch"`
			Console      bool                    `json:"console"`
			Operator     bool                    `json:"operator"`
			Adjustment   *store.SupplyAdjustment `json:"adjustment,omitempty"`
			PlayerUUID   string                  `json:"playerUuid"`
			History      bool                    `json:"history"`
		}
		if socialBodyV2(w, r, &in, 8192) != nil || !in.Operator {
			failError(w, r, store.ErrUnauthorized)
			return
		}
		actor := "console:" + node.ID
		if !in.Console {
			p, e := s.supplyPlayer(node, in.OperatorUUID, in.SessionEpoch)
			if e != nil {
				failError(w, r, e)
				return
			}
			actor = p.PlayerUUID
		}
		// Operator and UUID are supplied only by the trusted Core command handler, never forwarded from game packets.
		var out any
		if in.Adjustment != nil {
			out, err = s.Store.AdjustSupplyTickets(r.Context(), actor, *in.Adjustment, false)
		} else if in.History {
			out, err = s.Store.SupplyLedger(r.Context(), store.SupplyFilter{PlayerUUID: in.PlayerUUID, Limit: 10})
		} else {
			out, err = s.Store.SupplyAccount(r.Context(), in.PlayerUUID)
		}
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		success(w, r, out)
	})
}

func (s *Server) registerSupplyDeliveryBridge(mux *http.ServeMux) {
	mux.HandleFunc("POST /bridge/v1/supply/deliveries", func(w http.ResponseWriter, r *http.Request) {
		node, err := s.supplyNode(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		players, _ := s.Core.Presence([]string{node.ID})
		ids := []string{}
		for _, p := range players {
			live, e := s.supplyPlayer(node, p.PlayerUUID, p.SessionEpoch)
			if e == nil {
				ids = append(ids, live.PlayerUUID)
			}
		}
		if len(ids) > 256 {
			ids = ids[:256]
		}
		out, err := s.Store.PendingSupplyDeliveries(r.Context(), ids)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		success(w, r, map[string]any{"items": out})
	})
	for _, action := range []string{"claim", "begin"} {
		mux.HandleFunc("POST /bridge/v1/supply/deliveries/"+action, func(w http.ResponseWriter, r *http.Request) {
			node, err := s.supplyNode(r)
			if err != nil {
				failError(w, r, err)
				return
			}
			var in struct {
				ID             string `json:"id"`
				Token          string `json:"token"`
				PlayerUUID     string `json:"playerUuid"`
				SessionEpoch   string `json:"sessionEpoch"`
				InventoryEpoch string `json:"inventoryEpoch"`
			}
			if socialBodyV2(w, r, &in, 4096) != nil {
				socialFailureV2(w, r, store.ErrSocialInvalid)
				return
			}
			if _, err = s.supplyPlayer(node, in.PlayerUUID, in.SessionEpoch); err != nil {
				failError(w, r, err)
				return
			}
			var out any
			if action == "claim" {
				out, err = s.Store.ClaimSupplyDelivery(r.Context(), in.ID, node.ID, in.PlayerUUID, in.SessionEpoch)
			} else {
				err = s.Store.BeginSupplyDelivery(r.Context(), in.ID, node.ID, in.SessionEpoch, in.Token, in.InventoryEpoch)
				out = map[string]bool{"started": err == nil}
			}
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			success(w, r, out)
		})
	}
	mux.HandleFunc("POST /bridge/v1/supply/deliveries/finish", func(w http.ResponseWriter, r *http.Request) {
		node, err := s.supplyNode(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		var in store.SupplyDeliveryResult
		if socialBodyV2(w, r, &in, 8192) != nil {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		out, err := s.Store.FinishSupplyDelivery(r.Context(), node.ID, in)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		success(w, r, out)
	})
}

func (s *Server) supplyGameState(ctx context.Context, player store.CorePlayerIdentity, req supplyGameRequest) (map[string]any, error) {
	published, err := s.Store.SupplyPlayerCatalog(ctx)
	if err != nil {
		return nil, err
	}
	account, err := s.Store.SupplyAccount(ctx, player.PlayerUUID)
	if err != nil {
		return nil, err
	}
	pools := make([]store.SupplyPool, 0, len(published.Pools))
	for _, pool := range published.Pools {
		currency := pool.Definition.Currency()
		if published.Tickets[currency].Enabled && (req.TicketCatalog || supply.LegacyCurrency(currency)) {
			pools = append(pools, pool)
		}
	}
	catalog, art, deltaMode, err := s.publicSupplyCatalogDelta(ctx, pools, req.KnownPools, req.PresentationVersion)
	if err != nil {
		return nil, err
	}
	var credits any
	node, e := s.economyNode()
	if e == nil {
		balanceCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		op, e := s.coreCall(balanceCtx, player.PlayerUUID, store.ID("sb_"), node, "wallet.balance", map[string]string{"playerUuid": player.PlayerUUID})
		cancel()
		if e == nil {
			var balance map[string]any
			if CoreResult(op, &balance) == nil {
				credits = balance["amount"]
			}
		}
	}
	ticketsJSON, _ := json.Marshal(published.Tickets)
	ticketsDigest := store.Digest(ticketsJSON)[:16]
	out := map[string]any{
		"protocol":            1,
		"presentationVersion": supplyPresentationVersion,
		"serverTime":          time.Now().UTC(),
		"catalog":             catalog,
		"artwork":             art,
		"account":             publicSupplyAccount(account),
		"settings":            published.Settings,
		"ticketsDigest":       ticketsDigest,
		"credits":             credits,
	}
	if deltaMode && req.KnownTicketsDigest == ticketsDigest {
		out["ticketsUnchanged"] = true
	} else {
		out["tickets"] = published.Tickets
	}
	return out, nil
}

func (s *Server) supplyPlayerExchange(ctx context.Context, id, uuid string) (map[string]any, error) {
	x, err := s.Store.SupplyExchange(ctx, id, uuid)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(x)
	if err != nil {
		return nil, err
	}
	out, err := supplyRawObject(raw)
	if err != nil {
		return nil, err
	}
	account, err := s.Store.SupplyAccount(ctx, uuid)
	if err != nil {
		return nil, err
	}
	out["account"] = publicSupplyAccount(account)
	return out, nil
}
