package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/big"
	"sort"
	"strconv"
	"time"
)

func commerceWithinExecutionLimitV2(amount string) error {
	n, e := commerceAmountV2(amount)
	if e != nil {
		return e
	}
	if n.Sign() <= 0 || n.Cmp(big.NewInt(100000000000000)) > 0 {
		return catalogError(422, "AMOUNT_LIMIT", "单笔金额超出受控经济操作范围。")
	}
	return nil
}
func commerceNewResourceV2(kind, channel string, owner User, amount string, now time.Time) CommerceRecordV2 {
	prefix := "order_"
	state := "PAYMENT_PROCESSING"
	if kind == "COMMISSION" {
		prefix = "commission_"
		state = "FUNDING"
	}
	id := ID(prefix)
	return CommerceRecordV2{ID: id, Kind: kind, Channel: channel, OwnerID: owner.ID, OwnerUUID: owner.ServerUUID, EscrowRef: "escrow:" + id, Amount: amount, SettledAmount: "0.00", RefundedAmount: "0.00", State: state, FundsState: "PROCESSING", SnapshotID: ID("snapshot_"), Version: 1, CreatedAt: now, UpdatedAt: now}
}
func commerceCategoryNameV2(code string) string {
	return map[string]string{"MATERIALS": "建材", "EQUIPMENT": "装备", "SUPPLIES": "补给", "DECORATION": "装饰", "CONSTRUCTION": "建筑服务", "OTHER": "其他"}[code]
}
func commerceItemSnapshotV2(ctx context.Context, tx *sql.Tx, p CatalogRecordV2, quantity int64) (CatalogObjectV2, error) {
	content, version := p.Body, p.Version
	photos := catalogIDs(content, "photoAssetIds")
	category := commerceCategoryNameV2(catalogString(content, "categoryCode"))
	included := any([]any{})
	blocks := any([]any{})
	if p.Kind == "product" {
		content = p.Published
		version = p.PublishedVersion
		photos = catalogIDs(content, "galleryAssetIds")
		included = content["includedItems"]
		blocks = content["contentBlocks"]
		if e := tx.QueryRowContext(ctx, "SELECT title FROM catalog_records_v2 WHERE resource_id=? AND kind='category'", catalogString(content, "categoryId")).Scan(&category); e != nil {
			return nil, e
		}
	}
	price, ok := catalogMoney(content["price"])
	if !ok {
		return nil, catalogInvalid()
	}
	originalPrice := catalogMoneyString(price)
	if p.Kind == "product" {
		var e error
		price, e = promotionPriceV209(content)
		if e != nil {
			return nil, e
		}
	}
	return CatalogObjectV2{"productId": p.ID, "productVersion": version, "title": content["title"], "subtitle": content["subtitle"], "description": content["description"], "unitPrice": catalogMoneyString(price), "originalUnitPrice": originalPrice, "quantity": quantity, "photoAssetIds": photos, "categoryName": category, "includedItems": included, "contentBlocks": blocks}, nil
}

// Quote and checkout use the same whole-order delivery limits. This reads the
// frozen published templates only; it creates no order, hold or delivery intent.
func commerceDeliveryContentsV2(ctx context.Context, tx *sql.Tx, products map[string]CatalogRecordV2, quantities map[string]int64, nodes map[string]CatalogNodePolicyV2) (CatalogObjectV2, error) {
	if _, _, e := productMailText(products, quantities, ""); e != nil {
		return nil, e
	}
	attachments := map[string]CatalogObjectV2{}
	credits := int64(0)
	allowed := map[string]bool{}
	domain := ""
	shop := ""
	refs := map[string]bool{}
	for _, id := range commerceSortedKeysV2(quantities) {
		p := products[id]
		if shop != "" && p.StoreID != shop {
			return nil, catalogError(422, "MIXED_STORES", "一个订单只能包含同一家官方店铺的商品。")
		}
		shop = p.StoreID
		refs[catalogString(p.Published, "deliveryTemplateRef")] = true
	}
	// Different product combinations can reference the same templates in a
	// different order. Lock unique template references in one global order.
	orderedRefs := make([]string, 0, len(refs))
	for ref := range refs {
		orderedRefs = append(orderedRefs, ref)
	}
	sort.Strings(orderedRefs)
	for _, ref := range orderedRefs {
		if _, e := catalogRecordTx(ctx, tx, ref, "delivery_template", true); e != nil {
			return nil, e
		}
	}
	first := true
	for _, id := range commerceSortedKeysV2(quantities) {
		product := products[id]
		credits += catalogNumber(product.Published, "deliveryCredits") * quantities[id]
		if credits > 1000000000000 {
			return nil, catalogError(422, "DELIVERY_CREDITS_LIMIT", "单笔邮件信用点超出交付上限，请分开购买。")
		}
		template, e := catalogPublishedTemplateV2(ctx, tx, product, nodes)
		if e != nil {
			return nil, e
		}
		td := catalogString(template.Body, "inventoryDomain")
		if domain != "" && domain != td {
			return nil, catalogError(422, "DELIVERY_SCOPE_CONFLICT", "同一订单不能混用不同背包域的物品。")
		}
		domain = td
		servers := catalogIDs(template.Body, "allowedServerIds")
		present := map[string]bool{}
		for _, server := range servers {
			present[server] = true
		}
		if first {
			allowed = present
			first = false
		} else {
			for server := range allowed {
				if !present[server] {
					delete(allowed, server)
				}
			}
		}
		for _, v := range template.Body["attachments"].([]any) {
			a, _ := catalogObject(v)
			key := catalogString(a, "itemRef") + ":" + strconv.FormatInt(catalogNumber(a, "revision"), 10)
			quantity := catalogNumber(a, "quantity") * quantities[id]
			if old, exists := attachments[key]; exists {
				quantity += catalogNumber(old, "quantity")
			}
			if quantity > 99999 {
				return nil, catalogError(422, "DELIVERY_QUANTITY_LIMIT", "合并订单的物品数量超过安全交付上限。")
			}
			attachments[key] = CatalogObjectV2{"itemRef": a["itemRef"], "revision": a["revision"], "quantity": quantity, "payloadSha256": a["payloadSha256"]}
		}
	}
	if len(allowed) == 0 || (len(attachments) == 0 && credits == 0) || len(attachments) > 32 {
		return nil, catalogError(422, "DELIVERY_SCOPE_CONFLICT", "商品没有共同的安全领取节点，或附件过多，请分开购买。")
	}
	if credits > 0 && nodes != nil {
		known, supported := false, false
		for id := range allowed {
			if value := nodes[id].CreditRewards; value != nil {
				known = true
				supported = supported || *value
			}
		}
		if known && !supported {
			return nil, catalogError(503, "CREDIT_DELIVERY_UNAVAILABLE", "信用点邮件交付暂未就绪，请稍后再购买。")
		}
	}
	keys := []string{}
	for key := range attachments {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := []any{}
	for _, key := range keys {
		a := attachments[key]
		var raw string
		if e := tx.QueryRowContext(ctx, "SELECT metadata FROM item_versions WHERE item_ref=? AND revision=?", a["itemRef"], a["revision"]).Scan(&raw); e != nil {
			return nil, e
		}
		var metadata CatalogObjectV2
		if e := json.Unmarshal([]byte(raw), &metadata); e != nil {
			return nil, e
		}
		if catalogNumber(a, "quantity") > catalogNumber(metadata, "maxQuantity") {
			return nil, catalogError(422, "DELIVERY_QUANTITY_LIMIT", "购买数量超过该物品版本的安全交付上限。")
		}
		items = append(items, a)
	}
	servers := []string{}
	for id := range allowed {
		servers = append(servers, id)
	}
	sort.Strings(servers)
	snapshot := CatalogObjectV2{"schemaVersion": 1, "inventoryDomain": domain, "allowedServerIds": servers, "attachments": items}
	if credits > 0 {
		snapshot["creditAmount"] = credits
	}
	return snapshot, nil
}
func commerceMailPlanV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2, products map[string]CatalogRecordV2, quantities map[string]int64, nodes map[string]CatalogNodePolicyV2) (CatalogObjectV2, error) {
	snapshot, e := commerceDeliveryContentsV2(ctx, tx, products, quantities, nodes)
	if e != nil {
		return nil, e
	}
	snapshot["orderId"], snapshot["recipientUuid"] = d.ID, d.OwnerUUID
	raw := catalogJSON(snapshot)
	if len(raw) > 24000 {
		return nil, catalogError(422, "DELIVERY_SIZE_LIMIT", "订单交付快照过大，请分开购买。")
	}
	title, body, e := productMailText(products, quantities, catalogString(d.Body, "orderNo"))
	if e != nil {
		return nil, e
	}
	seller, _ := catalogObject(d.Body["seller"])
	sender := catalogString(seller, "displayName")
	if sender == "" {
		sender = "官方商城"
	}
	return CatalogObjectV2{"source": "deuterium-commerce", "deliveryId": ID("delivery_"), "orderId": d.ID, "recipientUuid": d.OwnerUUID, "title": title, "body": body, "sender": sender, "snapshotJson": raw, "snapshotSha256": Digest([]byte(raw)), "allowedServerIds": snapshot["allowedServerIds"], "inventoryDomain": snapshot["inventoryDomain"]}, nil
}

func (s *Store) PrepareOrderV2(ctx context.Context, actor, key, quoteID string, expected int64, channel string, available bool, nodes map[string]CatalogNodePolicyV2) (CommerceMutationV2, error) {
	input := map[string]any{"quoteId": quoteID, "expectedQuoteVersion": expected, "channel": channel}
	return s.commerceMutateV2(ctx, actor, key, "order.create", input, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		if e := commerceAvailableV2(available); e != nil {
			return result, e
		}
		if channel != "OFFICIAL_STORE" && channel != "PLAYER_MARKET" {
			return result, catalogInvalid()
		}
		owner, e := commerceUserTxV2(ctx, tx, actor, true)
		if e != nil {
			return result, e
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		var quotedChannel, raw string
		var expires time.Time
		e = tx.QueryRowContext(ctx, "SELECT channel,snapshot,expires_at FROM catalog_quotes_v2 WHERE quote_id=? AND user_id=? FOR UPDATE", quoteID, actor).Scan(&quotedChannel, &raw, &expires)
		if e == sql.ErrNoRows {
			return result, catalogNotFound()
		}
		if e != nil {
			return result, e
		}
		if quotedChannel != channel {
			return result, catalogInvalid()
		}
		if !expires.After(now) {
			return result, catalogError(409, "QUOTE_EXPIRED", "报价已过期，请重新获取。")
		}
		var existing string
		e = tx.QueryRowContext(ctx, "SELECT resource_id FROM commerce_resources_v2 WHERE quote_id=? FOR UPDATE", quoteID).Scan(&existing)
		if e == nil {
			return result, catalogError(409, "QUOTE_ALREADY_USED", "此报价已创建订单，请在我的订单查询原结果。")
		}
		if e != sql.ErrNoRows {
			return result, e
		}
		var quote CatalogObjectV2
		if e = json.Unmarshal([]byte(raw), &quote); e != nil {
			return result, e
		}
		if expected != catalogNumber(quote, "version") {
			return result, catalogVersion()
		}
		amount := catalogString(quote, "totalAmount")
		if amount != "0.00" || channel != "OFFICIAL_STORE" {
			if e = commerceWithinExecutionLimitV2(amount); e != nil {
				return result, e
			}
		}
		lines, ok := quote["items"].([]any)
		if !ok || len(lines) < 1 || len(lines) > 100 {
			return result, catalogInvalid()
		}
		quantities := map[string]int64{}
		lineByID := map[string]CatalogObjectV2{}
		for _, v := range lines {
			line, ok := catalogObject(v)
			if !ok {
				return result, catalogInvalid()
			}
			id := catalogString(line, "productId")
			if !CatalogReferenceV2(id) || quantities[id] != 0 || !catalogRange(line["quantity"], 1, 999) {
				return result, catalogInvalid()
			}
			quantities[id] = catalogNumber(line, "quantity")
			lineByID[id] = line
		}
		if channel == "PLAYER_MARKET" && len(quantities) != 1 {
			return result, catalogInvalid()
		}
		kind := "product"
		if channel == "PLAYER_MARKET" {
			kind = "listing"
		}
		products := map[string]CatalogRecordV2{}
		total := new(big.Int)
		shop := ""
		var seller User
		confirmationHours := int64(72)
		construction := false
		for _, id := range commerceSortedKeysV2(quantities) {
			p, e := catalogRecordTx(ctx, tx, id, kind, true)
			if e != nil {
				return result, e
			}
			if p.State != "ACTIVE" {
				return result, catalogNotFound()
			}
			content, version := p.Body, p.Version
			if kind == "product" {
				content = p.Published
				version = p.PublishedVersion
				if content == nil {
					return result, catalogNotFound()
				}
				if shop != "" && shop != p.StoreID {
					return result, catalogError(422, "MIXED_STORES", "一个订单只能包含同一家官方店铺的商品。")
				}
				shop = p.StoreID
			} else {
				if p.OwnerID == actor {
					return result, catalogError(422, "SELF_PURCHASE", "不能购买自己发布的商品。")
				}
				seller, e = commerceUserTxV2(ctx, tx, p.OwnerID, true)
				if e != nil {
					return result, e
				}
				construction = content["categoryCode"] == "CONSTRUCTION"
				if construction {
					confirmationHours = catalogNumber(content, "workHours")
				}
			}
			line := lineByID[id]
			quantity := quantities[id]
			if version != catalogNumber(line, "productVersion") {
				return result, catalogVersion()
			}
			if p.Stock != nil && *p.Stock < quantity {
				return result, catalogError(422, "INSUFFICIENT_STOCK", "商品库存不足。")
			}
			if kind == "product" && quantity > catalogNumber(content, "limitPerOrder") {
				return result, catalogError(422, "PURCHASE_LIMIT", "数量超过单笔限购。")
			}
			if kind == "product" {
				if e := promotionCheckLimitsV209(ctx, tx, owner.ServerUUID, p, quantity, now); e != nil {
					return result, e
				}
			}
			price, ok := catalogMoney(content["price"])
			if !ok {
				return result, catalogInvalid()
			}
			if kind == "product" {
				price, e = promotionPriceV209(content)
				if e != nil {
					return result, e
				}
			}
			if catalogMoneyString(price) != catalogString(line, "unitPrice") {
				return result, catalogVersion()
			}
			subtotal := new(big.Int).Mul(price, big.NewInt(quantity))
			if catalogMoneyString(subtotal) != catalogString(line, "subtotal") {
				return result, catalogVersion()
			}
			total.Add(total, subtotal)
			products[id] = p
		}
		if kind == "product" {
			// Recompute exactly the quoted coupon, never silently substitute a new
			// promotion after the buyer has confirmed the amount.
			coupons := []CatalogRecordV2{}
			if applied, ok := catalogObject(quote["coupon"]); ok {
				coupon, e := catalogRecordTx(ctx, tx, catalogString(applied, "couponId"), "coupon", true)
				if e != nil {
					return result, e
				}
				if coupon.Version != catalogNumber(applied, "version") {
					return result, catalogVersion()
				}
				coupons = append(coupons, coupon)
			}
			pricing, e := promotionTotalsV209(products, quantities, coupons)
			if e != nil {
				return result, e
			}
			total, e = commerceAmountV2(catalogString(pricing, "totalAmount"))
			if e != nil {
				return result, e
			}
			if quote["couponDiscount"] != nil && pricing["couponDiscount"] != quote["couponDiscount"] {
				return result, catalogVersion()
			}
		}
		if catalogMoneyString(total) != amount {
			return result, catalogVersion()
		}
		delivery, ok := catalogObject(quote["delivery"])
		if !ok {
			return result, catalogInvalid()
		}
		if kind == "listing" {
			for _, p := range products {
				methods, _ := catalogStrings(p.Body["deliveryMethods"], 1, 2, 1, 16, true)
				allowed := false
				for _, m := range methods {
					if m == delivery["method"] {
						allowed = true
					}
				}
				if !allowed {
					return result, catalogError(422, "DELIVERY_UNAVAILABLE", "商品交付方式已改变，请重新获取报价。")
				}
			}
		} else if delivery["method"] != "MAILBOX" {
			return result, catalogInvalid()
		}
		d := commerceNewResourceV2("ORDER", channel, owner, amount, now)
		d.StoreID = shop
		d.QuoteID = quoteID
		buyerParty := commercePartyV2(owner)
		var sellerParty CatalogObjectV2
		if kind == "listing" {
			d.PayeeID = seller.ID
			d.PayeeUUID = seller.ServerUUID
			sellerParty = commercePartyV2(seller)
			for _, p := range products {
				sellerParty["contactQq"] = p.Body["contactQq"]
			}
		} else {
			storeDoc, e := catalogRecordTx(ctx, tx, shop, "store", false)
			if e != nil {
				return result, e
			}
			sellerParty = CatalogObjectV2{"kind": "OFFICIAL_STORE", "playerRef": nil, "storeId": shop, "displayName": storeDoc.Body["name"], "contactQq": storeDoc.Body["contactQq"]}
		}
		snapshots := []any{}
		for _, v := range lines {
			line, _ := catalogObject(v)
			id := catalogString(line, "productId")
			snap, e := commerceItemSnapshotV2(ctx, tx, products[id], quantities[id])
			if e != nil {
				return result, e
			}
			snapshots = append(snapshots, snap)
		}
		d.Body = CatalogObjectV2{"orderNo": "D" + now.Format("20060102") + d.ID[len("order_"):], "construction": construction, "buyer": buyerParty, "seller": sellerParty, "items": snapshots, "delivery": delivery, "confirmationHours": confirmationHours, "shippedAt": nil, "workCompletedAt": nil, "confirmedAt": nil}
		if channel == "OFFICIAL_STORE" {
			for _, k := range []string{"originalTotal", "productDiscount", "couponDiscount", "discountTotal", "coupon", "cartItems"} {
				if v, ok := quote[k]; ok {
					d.Body[k] = v
				}
			}
		}
		if channel == "OFFICIAL_STORE" {
			plan, e := commerceMailPlanV2(ctx, tx, d, products, quantities, nodes)
			if e != nil {
				return result, e
			}
			d.Body["mailboxPlan"] = plan
			d.Body["mailboxState"] = "NOT_CREATED"
		}
		snapshot := CatalogObjectV2{"snapshotId": d.SnapshotID, "orderId": d.ID, "buyer": buyerParty, "seller": sellerParty, "items": snapshots, "totalAmount": amount, "delivery": delivery, "confirmationHours": confirmationHours, "capturedAt": now}
		for _, k := range []string{"originalTotal", "productDiscount", "couponDiscount", "discountTotal", "coupon"} {
			if v, ok := d.Body[k]; ok {
				snapshot[k] = v
			}
		}
		if len(catalogJSON(snapshot)) > 1<<20 {
			return result, catalogError(422, "SNAPSHOT_SIZE_LIMIT", "订单详情过多，请分开购买。")
		}
		d.SnapshotSHA256 = commerceSnapshotHashV2(snapshot)
		snapshot["sha256"] = d.SnapshotSHA256
		if e = commerceSaveV2(ctx, tx, &d, true); e != nil {
			return result, e
		}
		if channel == "OFFICIAL_STORE" {
			if e = promotionReserveV209(ctx, tx, owner, d, quote, now); e != nil {
				return result, e
			}
		}
		if e = commerceInsertSnapshotV2(ctx, tx, d, snapshot, now); e != nil {
			return result, e
		}
		for _, id := range commerceSortedKeysV2(quantities) {
			p := products[id]
			quantity := quantities[id]
			if p.Stock != nil {
				n := *p.Stock - quantity
				p.Stock = &n
				p.Version++
				p.UpdatedAt = now
				if e = catalogSave(ctx, tx, &p, false); e != nil {
					return result, e
				}
			}
			holdState := "RESERVED"
			if p.Stock == nil {
				holdState = "UNLIMITED"
			}
			if _, e = tx.ExecContext(ctx, "INSERT INTO commerce_stock_holds_v2 VALUES(?,?,?,?)", d.ID, p.ID, quantity, holdState); e != nil {
				return result, e
			}
			sourceType := "PRODUCT_PUBLISHED"
			content := p.Published
			if kind == "listing" {
				sourceType = "MARKET_LISTING"
				content = p.Body
			}
			if e = s.AppendAssetBindingsV2(ctx, tx, sourceType, p.ID, "ORDER_SNAPSHOT", d.SnapshotID, CatalogAssetsV2(p.Kind, content)); e != nil {
				return result, e
			}
		}
		steps := []CommerceStepV2{commerceReserveStepV2(d)}
		if channel == "OFFICIAL_STORE" && amount == "0.00" {
			steps = []CommerceStepV2{}
		}
		if channel == "OFFICIAL_STORE" {
			plan, _ := catalogObject(d.Body["mailboxPlan"])
			steps = append(steps, CommerceStepV2{Command: "mailbox.create", Payload: map[string]any(plan)})
		}
		operationKind := "STORE_PURCHASE"
		if channel == "PLAYER_MARKET" {
			operationKind = "MARKET_PURCHASE"
		}
		op, e := commerceNewOperationV2(ctx, tx, &d, actor, key, operationKind, "reserve", amount, steps, false, now)
		if e != nil {
			return result, e
		}
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		if e = commerceEventV2(ctx, tx, d, actor, "order.payment.started", "订单已记录，正在核实预付结果。", map[string]string{"operationId": op.ID}); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: d.Kind, OperationID: op.ID}, nil
	})
}

func (s *Store) PrepareCommissionV2(ctx context.Context, actor, key string, content CatalogObjectV2, available bool) (CommerceMutationV2, error) {
	if e := ValidateCommissionContentV2(content); e != nil {
		return CommerceMutationV2{}, e
	}
	amount := catalogString(content, "reward")
	if e := commerceWithinExecutionLimitV2(amount); e != nil {
		return CommerceMutationV2{}, e
	}
	reward, _ := catalogMoney(amount)
	amount = catalogMoneyString(reward)
	var replay int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM commerce_requests_v2 WHERE actor_id=? AND client_request_id=?", actor, key).Scan(&replay); err != nil {
		return CommerceMutationV2{}, err
	}
	if replay == 0 {
		if err := commerceAvailableV2(available); err != nil {
			return CommerceMutationV2{}, err
		}
		if err := s.EnsureAssetsActiveV2(ctx, actor, "", "", []string{catalogString(content, "coverAssetId")}); err != nil {
			return CommerceMutationV2{}, err
		}
	}
	return s.commerceMutateV2(ctx, actor, key, "commission.create", content, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		if e := commerceAvailableV2(available); e != nil {
			return result, e
		}
		owner, e := commerceUserTxV2(ctx, tx, actor, true)
		if e != nil {
			return result, e
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		d := commerceNewResourceV2("COMMISSION", "COMMISSION", owner, amount, now)
		d.Body = CatalogObjectV2{"owner": commercePartyV2(owner), "worker": nil, "content": content, "workDueAt": nil, "acceptanceDueAt": nil, "completionDescription": "", "completionAssetIds": []string{}, "acceptedAt": nil, "completedAt": nil, "confirmedAt": nil}
		snapshot := CatalogObjectV2{"snapshotId": d.SnapshotID, "commissionId": d.ID, "owner": d.Body["owner"], "worker": nil, "content": content, "acceptanceHours": 72, "capturedAt": now}
		d.SnapshotSHA256 = commerceSnapshotHashV2(snapshot)
		snapshot["sha256"] = d.SnapshotSHA256
		if e = commerceSaveV2(ctx, tx, &d, true); e != nil {
			return result, e
		}
		ids := []string{catalogString(content, "coverAssetId")}
		var purpose string
		if e = tx.QueryRowContext(ctx, "SELECT purpose FROM asset_uploads_v2 WHERE asset_id=?", ids[0]).Scan(&purpose); e != nil {
			return result, e
		}
		if purpose != "COMMISSION_COVER" {
			return result, ErrAssetUnavailable
		}
		if e = s.SetAssetBindingsV2(ctx, tx, actor, "COMMISSION", d.ID, ids); e != nil {
			return result, e
		}
		if e = s.CopyAssetBindingsV2(ctx, tx, "COMMISSION", d.ID, "COMMISSION_SNAPSHOT", d.SnapshotID, ids); e != nil {
			return result, e
		}
		if e = commerceInsertSnapshotV2(ctx, tx, d, snapshot, now); e != nil {
			return result, e
		}
		op, e := commerceNewOperationV2(ctx, tx, &d, actor, key, "COMMISSION_PUBLISH", "reserve", amount, []CommerceStepV2{commerceReserveStepV2(d)}, false, now)
		if e != nil {
			return result, e
		}
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		if e = commerceEventV2(ctx, tx, d, actor, "commission.funding.started", "委托预付正在核实，确认成功后才进入大厅。", map[string]string{"operationId": op.ID}); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: d.Kind, OperationID: op.ID}, nil
	})
}

func (s *Store) PrepareCommissionAcceptV2(ctx context.Context, actor, id, key string, expected int64, available bool) (CommerceMutationV2, error) {
	return s.commerceMutateV2(ctx, actor, key, "commission.accept:"+id, map[string]any{"expectedVersion": expected}, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		if e := commerceAvailableV2(available); e != nil {
			return result, e
		}
		d, e := commerceRecordTxV2(ctx, tx, id, "COMMISSION", true)
		if e != nil {
			return result, e
		}
		if e = commerceRequireVersionV2(d, expected); e != nil {
			return result, e
		}
		if d.OwnerID == actor {
			return result, catalogError(422, "SELF_ACCEPT", "不能接取自己发布的委托。")
		}
		if d.State != "OPEN" || d.FundsState != "HELD" || d.PayeeID != "" {
			return result, catalogError(409, "COMMISSION_UNAVAILABLE", "委托已被接取或不再开放。")
		}
		worker, e := commerceUserTxV2(ctx, tx, actor, true)
		if e != nil {
			return result, e
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		d.PayeeID = worker.ID
		d.PayeeUUID = worker.ServerUUID
		d.Body["worker"] = commercePartyV2(worker)
		step := CommerceStepV2{Command: "wallet.escrow.bind", Payload: map[string]any{"escrowRef": d.EscrowRef, "businessRef": d.ID, "payeeUuid": d.PayeeUUID}}
		op, e := commerceNewOperationV2(ctx, tx, &d, actor, key, "COMMISSION_ACCEPT", "bind", d.Amount, []CommerceStepV2{step}, false, now)
		if e != nil {
			return result, e
		}
		d.State = "ACTIVE"
		d.FundsState = "UNKNOWN"
		commerceTouchV2(&d, now)
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		if e = commerceEventV2(ctx, tx, d, actor, "commission.accept.reserved", "正在接取委托，请稍候。", map[string]string{"operationId": op.ID}); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: d.Kind, OperationID: op.ID}, nil
	})
}
