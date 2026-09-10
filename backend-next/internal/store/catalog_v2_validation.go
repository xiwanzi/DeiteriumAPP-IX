package store

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"unicode/utf8"
)

// CatalogObjectV2 is a validated contract object. Unknown and missing fields are
// rejected before persistence, including nested content and attachment IDs.
type CatalogObjectV2 map[string]any
type CatalogErrorV2 struct {
	Status        int
	Code, Message string
}

func (e *CatalogErrorV2) Error() string { return e.Code }
func catalogError(status int, code, message string) error {
	return &CatalogErrorV2{status, code, message}
}
func catalogInvalid() error {
	return catalogError(400, "INVALID_REQUEST", "字段格式、数量或内容不符合要求。")
}
func catalogNotFound() error { return catalogError(404, "NOT_FOUND", "内容不存在或不可见。") }
func catalogDenied() error {
	return catalogError(403, "FORBIDDEN", "没有管理此内容的权限。")
}
func catalogVersion() error {
	return catalogError(409, "STATE_VERSION_CONFLICT", "内容已更新，请刷新后再操作。")
}

var catalogRefPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)
var catalogAmountPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]{1,2})?$`)
var catalogQQPattern = regexp.MustCompile(`^[0-9]{5,12}$`)
var catalogColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
var catalogCategories = []string{"MATERIALS", "EQUIPMENT", "SUPPLIES", "DECORATION", "CONSTRUCTION", "OTHER"}

func CatalogReferenceV2(v string) bool { return catalogRefPattern.MatchString(v) }
func catalogText(v any, min, max int) bool {
	s, ok := v.(string)
	return ok && utf8.ValidString(s) && !strings.ContainsRune(s, 0) && utf8.RuneCountInString(strings.TrimSpace(s)) >= min && utf8.RuneCountInString(s) <= max
}
func catalogString(o CatalogObjectV2, k string) string { v, _ := o[k].(string); return v }
func catalogInt(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) || n > 2147483647 || n < -2147483647 {
			return 0, false
		}
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, e := n.Int64()
		return i, e == nil
	}
	return 0, false
}
func catalogNumber(o CatalogObjectV2, k string) int64 { n, _ := catalogInt(o[k]); return n }
func catalogRange(v any, min, max int64) bool {
	n, ok := catalogInt(v)
	return ok && n >= min && n <= max
}
func catalogObject(v any) (CatalogObjectV2, bool) {
	switch o := v.(type) {
	case map[string]any:
		return CatalogObjectV2(o), true
	case CatalogObjectV2:
		return o, true
	}
	return nil, false
}
func catalogFields(o CatalogObjectV2, required, optional string) bool {
	allowed := map[string]bool{}
	for _, k := range strings.Fields(required + " " + optional) {
		allowed[k] = true
	}
	for _, k := range strings.Fields(required) {
		if _, ok := o[k]; !ok {
			return false
		}
	}
	for k := range o {
		if !allowed[k] {
			return false
		}
	}
	return true
}
func catalogEnum(v any, options ...string) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	for _, x := range options {
		if s == x {
			return true
		}
	}
	return false
}
func catalogStrings(v any, min, max, eachMin, eachMax int, unique bool) ([]string, bool) {
	if strings, ok := v.([]string); ok {
		items := make([]any, len(strings))
		for i, value := range strings {
			items[i] = value
		}
		v = items
	}
	a, ok := v.([]any)
	if !ok || len(a) < min || len(a) > max {
		return nil, false
	}
	out := make([]string, 0, len(a))
	seen := map[string]bool{}
	for _, x := range a {
		if !catalogText(x, eachMin, eachMax) {
			return nil, false
		}
		s := x.(string)
		if unique && seen[s] {
			return nil, false
		}
		seen[s] = true
		out = append(out, s)
	}
	return out, true
}
func catalogRefs(v any, min, max int) ([]string, bool) {
	a, ok := catalogStrings(v, min, max, 1, 128, true)
	if !ok {
		return nil, false
	}
	for _, x := range a {
		if !CatalogReferenceV2(x) {
			return nil, false
		}
	}
	return a, true
}
func catalogIDs(o CatalogObjectV2, k string) []string { a, _ := catalogRefs(o[k], 0, 100); return a }
func catalogNullableRef(v any) bool {
	return v == nil || (catalogText(v, 0, 128) && (v == "" || CatalogReferenceV2(v.(string))))
}
func catalogMoney(v any) (*big.Int, bool) {
	s, ok := v.(string)
	if !ok || len(s) > 20 || !catalogAmountPattern.MatchString(s) {
		return nil, false
	}
	p := strings.Split(s, ".")
	fraction := "00"
	if len(p) == 2 {
		fraction = p[1] + strings.Repeat("0", 2-len(p[1]))
	}
	n, ok := new(big.Int).SetString(p[0]+fraction, 10)
	return n, ok && n.Sign() > 0 && len(catalogMoneyString(n)) <= 20
}
func catalogMoneyString(n *big.Int) string {
	s := n.String()
	for len(s) < 3 {
		s = "0" + s
	}
	return s[:len(s)-2] + "." + s[len(s)-2:]
}

func ValidateCatalogContentV2(kind string, o CatalogObjectV2) error {
	switch kind {
	case "store":
		if !catalogFields(o, "name intro logoAssetId coverAssetId contactQq serviceHours notice", "") || !catalogText(o["name"], 1, 60) || !catalogText(o["intro"], 0, 300) || !catalogNullableRef(o["logoAssetId"]) || !catalogNullableRef(o["coverAssetId"]) || !catalogQQPattern.MatchString(catalogString(o, "contactQq")) || !catalogText(o["serviceHours"], 0, 100) || !catalogText(o["notice"], 0, 1000) {
			return catalogInvalid()
		}
	case "brand", "category":
		required := "name sortOrder active"
		if kind == "brand" {
			required += " logoAssetId"
		}
		max := 40
		sortMax := int64(2147483647)
		if kind == "brand" {
			max = 60
			sortMax = 9999
		}
		if !catalogFields(o, required, "") || !catalogText(o["name"], 1, max) || !catalogRange(o["sortOrder"], 0, sortMax) {
			return catalogInvalid()
		}
		if _, ok := o["active"].(bool); !ok {
			return catalogInvalid()
		}
		if kind == "brand" && !catalogNullableRef(o["logoAssetId"]) {
			return catalogInvalid()
		}
	case "product":
		if !catalogFields(o, "title subtitle description brandId categoryId price coverAssetId galleryAssetIds galleryAltTexts contentBlocks includedItems deliveryTemplateRef deliverySummary estimatedDelivery inventoryPolicy stock limitPerOrder posterTone accentColor badges sortOrder", "mailTitle mailBody discountRate purchaseLimits deliveryCredits") {
			return catalogInvalid()
		}
		if e := validateProductPromotionV209(o); e != nil {
			return e
		}
		if e := validateProductMail(o); e != nil {
			return e
		}
		if !catalogText(o["title"], 1, 80) || !catalogText(o["subtitle"], 1, 200) || !catalogText(o["description"], 1, 10000) || !CatalogReferenceV2(catalogString(o, "brandId")) || !CatalogReferenceV2(catalogString(o, "categoryId")) || !CatalogReferenceV2(catalogString(o, "deliveryTemplateRef")) {
			return catalogInvalid()
		}
		if _, ok := catalogMoney(o["price"]); !ok {
			return catalogInvalid()
		}
		ids, ok := catalogRefs(o["galleryAssetIds"], 1, 20)
		if !ok || ids[0] != catalogString(o, "coverAssetId") {
			return catalogInvalid()
		}
		alts, ok := catalogStrings(o["galleryAltTexts"], 1, 20, 0, 200, false)
		if !ok || len(alts) != len(ids) {
			return catalogInvalid()
		}
		if _, ok := catalogStrings(o["includedItems"], 1, 40, 1, 200, false); !ok {
			return catalogInvalid()
		}
		if _, ok := catalogStrings(o["badges"], 0, 4, 1, 20, false); !ok {
			return catalogInvalid()
		}
		if !catalogText(o["deliverySummary"], 1, 500) || !catalogText(o["estimatedDelivery"], 1, 150) || !catalogEnum(o["inventoryPolicy"], "FINITE", "UNLIMITED") || !catalogRange(o["stock"], 0, 999999) || !catalogRange(o["limitPerOrder"], 1, 999) || !catalogRange(o["sortOrder"], 0, 99999) || !catalogEnum(o["posterTone"], "LIGHT", "DARK") || !catalogColorPattern.MatchString(catalogString(o, "accentColor")) {
			return catalogInvalid()
		}
		blocks, ok := o["contentBlocks"].([]any)
		if !ok || len(blocks) > 80 {
			return catalogInvalid()
		}
		seen := map[string]bool{}
		for _, v := range blocks {
			b, ok := catalogObject(v)
			if !ok || !catalogFields(b, "blockId type", "heading text assetId altText rows") || !CatalogReferenceV2(catalogString(b, "blockId")) || seen[catalogString(b, "blockId")] {
				return catalogInvalid()
			}
			seen[catalogString(b, "blockId")] = true
			for k, max := range map[string]int{"heading": 100, "text": 3000, "altText": 200} {
				if x, has := b[k]; has && !catalogText(x, 0, max) {
					return catalogInvalid()
				}
			}
			if asset, has := b["assetId"]; has && !catalogNullableRef(asset) {
				return catalogInvalid()
			}
			if rawRows, has := b["rows"]; has {
				rows, ok := rawRows.([]any)
				if !ok || len(rows) > 30 {
					return catalogInvalid()
				}
				for _, v := range rows {
					row, ok := catalogObject(v)
					if !ok || !catalogFields(row, "label value", "") || !catalogText(row["label"], 1, 80) || !catalogText(row["value"], 1, 500) {
						return catalogInvalid()
					}
				}
			}
			switch b["type"] {
			case "HEADING":
				if !catalogText(b["heading"], 1, 100) {
					return catalogInvalid()
				}
			case "PARAGRAPH":
				if !catalogText(b["text"], 1, 3000) {
					return catalogInvalid()
				}
			case "IMAGE":
				if !CatalogReferenceV2(catalogString(b, "assetId")) {
					return catalogInvalid()
				}
			case "KEY_VALUE_LIST":
				rows, ok := b["rows"].([]any)
				if !ok || len(rows) < 1 || len(rows) > 30 {
					return catalogInvalid()
				}
				for _, v := range rows {
					row, ok := catalogObject(v)
					if !ok || !catalogFields(row, "label value", "") || !catalogText(row["label"], 1, 80) || !catalogText(row["value"], 1, 500) {
						return catalogInvalid()
					}
				}
			default:
				return catalogInvalid()
			}
		}
	case "listing":
		if !catalogFields(o, "title subtitle description categoryCode price stock photoAssetIds contactQq deliveryMethods pickupLocation workHours", "") || !catalogText(o["title"], 1, 80) || !catalogText(o["subtitle"], 1, 200) || !catalogText(o["description"], 1, 10000) || !catalogEnum(o["categoryCode"], catalogCategories...) || !catalogRange(o["stock"], 1, 999) || !catalogQQPattern.MatchString(catalogString(o, "contactQq")) || !catalogText(o["pickupLocation"], 0, 120) || !catalogRange(o["workHours"], 1, 8760) {
			return catalogInvalid()
		}
		if _, ok := catalogMoney(o["price"]); !ok {
			return catalogInvalid()
		}
		if _, ok := catalogRefs(o["photoAssetIds"], 1, 5); !ok {
			return catalogInvalid()
		}
		methods, ok := catalogStrings(o["deliveryMethods"], 1, 2, 1, 16, true)
		if !ok {
			return catalogInvalid()
		}
		construction := o["categoryCode"] == "CONSTRUCTION"
		for _, m := range methods {
			if construction {
				if m != "WORKSITE" || len(methods) != 1 {
					return catalogInvalid()
				}
			} else if m != "DOOR" && m != "PICKUP" {
				return catalogInvalid()
			}
			if m == "PICKUP" && !catalogText(o["pickupLocation"], 1, 120) {
				return catalogInvalid()
			}
		}
	case "homepage":
		if !catalogFields(o, "intro sections brandIds categoryIds", "") || !catalogText(o["intro"], 0, 300) {
			return catalogInvalid()
		}
		if _, ok := catalogRefs(o["brandIds"], 0, 30); !ok {
			return catalogInvalid()
		}
		if _, ok := catalogRefs(o["categoryIds"], 0, 30); !ok {
			return catalogInvalid()
		}
		sections, ok := o["sections"].([]any)
		if !ok || len(sections) < 1 || len(sections) > 20 {
			return catalogInvalid()
		}
		seen := map[string]bool{}
		for _, v := range sections {
			sec, ok := catalogObject(v)
			if !ok || !catalogFields(sec, "sectionId title layout productIds bannerAssetId sortOrder", "") || !CatalogReferenceV2(catalogString(sec, "sectionId")) || seen[catalogString(sec, "sectionId")] || !catalogText(sec["title"], 1, 80) || !catalogEnum(sec["layout"], "HERO_CAROUSEL", "PRODUCT_GRID", "BANNER") || !catalogNullableRef(sec["bannerAssetId"]) || !catalogRange(sec["sortOrder"], 0, 2147483647) {
				return catalogInvalid()
			}
			seen[catalogString(sec, "sectionId")] = true
			if _, ok := catalogRefs(sec["productIds"], 0, 40); !ok {
				return catalogInvalid()
			}
			if sec["layout"] == "BANNER" && !CatalogReferenceV2(catalogString(sec, "bannerAssetId")) {
				return catalogInvalid()
			}
		}
	default:
		return catalogInvalid()
	}
	return nil
}

func CatalogAssetsV2(kind string, o CatalogObjectV2) []string {
	out := []string{}
	switch kind {
	case "product":
		out = append(out, catalogIDs(o, "galleryAssetIds")...)
		blocks, _ := o["contentBlocks"].([]any)
		for _, v := range blocks {
			if b, ok := catalogObject(v); ok && b["type"] == "IMAGE" {
				out = append(out, catalogString(b, "assetId"))
			}
		}
	case "listing":
		out = append(out, catalogIDs(o, "photoAssetIds")...)
	case "store", "brand":
		for _, k := range []string{"logoAssetId", "coverAssetId"} {
			if id := catalogString(o, k); id != "" {
				out = append(out, id)
			}
		}
	case "homepage":
		sections, _ := o["sections"].([]any)
		for _, v := range sections {
			if b, ok := catalogObject(v); ok {
				if id := catalogString(b, "bannerAssetId"); id != "" {
					out = append(out, id)
				}
			}
		}
	}
	unique := []string{}
	seen := map[string]bool{}
	for _, id := range out {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	return unique
}
func CatalogMutationInputV2(input CatalogObjectV2, content bool, version bool, extra string) (string, int64, CatalogObjectV2, error) {
	required := "clientRequestId"
	if version {
		required += " expectedVersion"
	}
	if content {
		required += " content"
	}
	if !catalogFields(input, required, extra) || !CatalogReferenceV2(catalogString(input, "clientRequestId")) {
		return "", 0, nil, catalogInvalid()
	}
	if version && !catalogRange(input["expectedVersion"], 1, 2147483647) {
		return "", 0, nil, catalogInvalid()
	}
	var obj CatalogObjectV2
	if content {
		var ok bool
		obj, ok = catalogObject(input["content"])
		if !ok {
			return "", 0, nil, catalogInvalid()
		}
	}
	return catalogString(input, "clientRequestId"), catalogNumber(input, "expectedVersion"), obj, nil
}
func CatalogQuoteInputV2(input CatalogObjectV2) error {
	if !catalogFields(input, "channel items delivery", "source") || !catalogEnum(input["channel"], "OFFICIAL_STORE", "PLAYER_MARKET") {
		return catalogInvalid()
	}
	if source, has := input["source"]; has && !catalogEnum(source, "CART", "DIRECT") {
		return catalogInvalid()
	}
	items, ok := input["items"].([]any)
	if !ok || len(items) < 1 || len(items) > 100 {
		return catalogInvalid()
	}
	seen := map[string]bool{}
	for _, v := range items {
		o, ok := catalogObject(v)
		if !ok || !catalogFields(o, "productId quantity expectedProductVersion", "") || !CatalogReferenceV2(catalogString(o, "productId")) || seen[catalogString(o, "productId")] || !catalogRange(o["quantity"], 1, 999) || !catalogRange(o["expectedProductVersion"], 1, 2147483647) {
			return catalogInvalid()
		}
		seen[catalogString(o, "productId")] = true
	}
	d, ok := catalogObject(input["delivery"])
	if !ok || !catalogFields(d, "method", "location projectName") || !catalogEnum(d["method"], "DOOR", "PICKUP", "WORKSITE", "MAILBOX") {
		return catalogInvalid()
	}
	for k, max := range map[string]int{"location": 120, "projectName": 80} {
		if v, has := d[k]; has && !catalogText(v, 0, max) {
			return catalogInvalid()
		}
	}
	if input["channel"] == "OFFICIAL_STORE" {
		if d["method"] != "MAILBOX" {
			return catalogInvalid()
		}
	} else {
		if d["method"] == "MAILBOX" || !catalogText(d["location"], 1, 120) || (d["method"] == "WORKSITE" && !catalogText(d["projectName"], 1, 80)) {
			return catalogInvalid()
		}
		if len(items) != 1 {
			return catalogError(422, "INVALID_REQUEST", "市场报价每次只支持一件商品，避免混用卖家和交付方式。")
		}
	}
	return nil
}
func catalogJSON(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		panic(fmt.Sprintf("validated catalog JSON: %v", e))
	}
	return string(b)
}
