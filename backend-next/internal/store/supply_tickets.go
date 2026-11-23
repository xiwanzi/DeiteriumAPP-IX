package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"
	"regexp"
)

type SupplyTicket struct {
	Name         string `json:"name"`
	Icon         string `json:"icon,omitempty"`
	UnitPrice    int64  `json:"unitPrice"`
	Enabled      bool   `json:"enabled"`
	managedPrice bool
}

type SupplyPlayerCatalog struct {
	Pools    []SupplyPool
	Settings SupplySettings
	Tickets  map[string]SupplyTicket
}

const supplySettingsQuery = "SELECT version,standard_price,limited_price FROM supply_settings WHERE id=1"

var ticketPriceNumber = regexp.MustCompile(`^[0-9]{1,7}(\.[0-9]{1,12})?([Ee][+-]?[0-9]{1,2})?$`)

func supplyTicketPrice(value json.Number, fallback int64) (int64, error) {
	if value == "" {
		return fallback, nil
	}
	invalid := supplyFailure("SUPPLY_TICKET_PRICE_INVALID", "券价须为 0 至 1000000 信用点，最多两位小数")
	if !ticketPriceNumber.MatchString(value.String()) {
		return 0, invalid
	}
	price, ok := new(big.Rat).SetString(value.String())
	if !ok {
		return 0, invalid
	}
	price.Mul(price, big.NewRat(100, 1))
	if !price.IsInt() || !price.Num().IsInt64() || price.Num().Int64() > 100000000 {
		return 0, invalid
	}
	return price.Num().Int64(), nil
}

func scanSupplySettings(row *sql.Row) (SupplySettings, error) {
	var settings SupplySettings
	err := row.Scan(&settings.Version, &settings.StandardPrice, &settings.LimitedPrice)
	return settings, err
}

func supplyTickets(ctx context.Context, tx *sql.Tx, settings SupplySettings) (map[string]SupplyTicket, error) {
	tickets := map[string]SupplyTicket{
		"STANDARD": {Name: "常驻券", UnitPrice: settings.StandardPrice, Enabled: true},
		"LIMITED":  {Name: "限定券", UnitPrice: settings.LimitedPrice, Enabled: true},
	}
	var raw []byte
	err := tx.QueryRowContext(ctx, "SELECT tickets_json FROM supply_editor_ticket_catalogs ORDER BY source_revision DESC LIMIT 1").Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return tickets, nil
	}
	if err != nil {
		return nil, err
	}
	var published []editorTicket
	if err = json.Unmarshal(raw, &published); err != nil {
		return nil, err
	}
	for _, ticket := range published {
		fallback := tickets[ticket.ID]
		unitPrice, err := supplyTicketPrice(ticket.Price, fallback.UnitPrice)
		if err != nil {
			return nil, err
		}
		name := ticket.Name
		if name == "" {
			name = fallback.Name
		}
		if !ticket.Enabled {
			unitPrice = 0
		}
		tickets[ticket.ID] = SupplyTicket{
			Name: name, Icon: editorArtworkID(ticket.Icon), UnitPrice: unitPrice,
			Enabled: ticket.Enabled, managedPrice: ticket.Price != "" || !ticket.Enabled,
		}
	}
	return tickets, nil
}

func (s *Store) SupplyPlayerCatalog(ctx context.Context) (SupplyPlayerCatalog, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return SupplyPlayerCatalog{}, err
	}
	defer tx.Rollback()
	settings, err := scanSupplySettings(tx.QueryRowContext(ctx, supplySettingsQuery))
	if err != nil {
		return SupplyPlayerCatalog{}, err
	}
	tickets, err := supplyTickets(ctx, tx, settings)
	if err != nil {
		return SupplyPlayerCatalog{}, err
	}
	pools, err := supplyPools(ctx, tx, false)
	if err != nil {
		return SupplyPlayerCatalog{}, err
	}
	settings.StandardPrice = tickets["STANDARD"].UnitPrice
	settings.LimitedPrice = tickets["LIMITED"].UnitPrice
	return SupplyPlayerCatalog{Pools: pools, Settings: settings, Tickets: tickets}, tx.Commit()
}
