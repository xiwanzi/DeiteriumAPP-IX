package supply

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type Artwork struct {
	Hero     string  `json:"hero"`
	Rail     string  `json:"rail"`
	Detail   string  `json:"detail"`
	Fallback string  `json:"fallback"`
	Theme    string  `json:"theme"`
	HeroX    float64 `json:"heroX"`
	HeroY    float64 `json:"heroY"`
}

// Item identity and revision are frozen at publication. The client only receives display fields.
type Reward struct {
	Key             string   `json:"key"`
	Name            string   `json:"name"`
	GunID           string   `json:"gunId"`
	ModelKey        string   `json:"modelKey"`
	Stars           int      `json:"stars"`
	Weight          int      `json:"weight"`
	UP              bool     `json:"up"`
	ItemRef         string   `json:"itemRef"`
	ItemRevision    int64    `json:"itemRevision"`
	PayloadSHA256   string   `json:"payloadSha256"`
	InventoryDomain string   `json:"inventoryDomain"`
	Servers         []string `json:"servers"`
	Art             Artwork  `json:"art"`
}

type Definition struct {
	Name         string          `json:"name"`
	Title        string          `json:"title"`
	Subtitle     string          `json:"subtitle"`
	Description  string          `json:"description"`
	Kind         string          `json:"kind"`
	SeriesID     string          `json:"seriesId"`
	StartsAt     *time.Time      `json:"startsAt"`
	EndsAt       *time.Time      `json:"endsAt"`
	VisibleAt    *time.Time      `json:"visibleAt"`
	Sort         int             `json:"sort"`
	Art          Artwork         `json:"art"`
	TicketID     string          `json:"ticketId,omitempty"`
	TicketArt    string          `json:"ticketArt,omitempty"`
	EventGoldPPM int             `json:"eventGoldPpm"`
	EventPity    int             `json:"eventPity"`
	Labels       *PoolLabels     `json:"labels,omitempty"`
	Sections     []PoolSection   `json:"sections,omitempty"`
	Rewards      []Reward        `json:"rewards"`
	Rules        map[string]Rule `json:"rules,omitempty"`
}

type PoolLabels struct {
	Rail   string `json:"rail"`
	Single string `json:"single"`
	Ten    string `json:"ten"`
}

type PoolSection struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

const EditorArtworkPrefix = "gacha-art-"

func EditorArtworkHash(id string) string {
	value := strings.TrimPrefix(id, EditorArtworkPrefix)
	if value != id && hash.MatchString(value) {
		return value
	}
	return ""
}

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)
var namespaced = regexp.MustCompile(`^[a-z0-9_.-]+:[a-z0-9_./-]+$`)
var hash = regexp.MustCompile(`^[a-f0-9]{64}$`)

func ValidID(value string) bool       { return identifier.MatchString(value) }
func ValidCurrency(value string) bool { return ValidID(value) }

func LegacyCurrency(value string) bool { return value == "STANDARD" || value == "LIMITED" }

func (d Definition) Currency() string {
	if d.TicketID != "" {
		return d.TicketID
	}
	if d.Kind == "STANDARD" {
		return "STANDARD"
	}
	return "LIMITED"
}

func (d Definition) Open(at time.Time) bool {
	return (d.StartsAt == nil || !at.Before(*d.StartsAt)) && (d.EndsAt == nil || at.Before(*d.EndsAt))
}

func (d Definition) Visible(at time.Time) bool {
	visible := d.VisibleAt
	if visible == nil {
		visible = d.StartsAt
	}
	return (visible == nil || !at.Before(*visible)) && (d.EndsAt == nil || at.Before(*d.EndsAt))
}

func (d Definition) Schemes() []string {
	if d.Kind == "LIMITED" {
		return []string{"80", "120"}
	}
	if d.Kind == "EVENT" {
		return []string{"event"}
	}
	return []string{"80"}
}

func validText(value string, min, max int) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(value))
	return utf8.ValidString(value) && n >= min && n <= max && !strings.ContainsRune(value, '\x00')
}

func (a Artwork) Validate(required bool) error {
	if a.Theme != "" && a.Theme != "violet" && a.Theme != "red" && a.Theme != "blue" {
		return errors.New("请选择已支持的主题配色")
	}
	if a.HeroX < 0 || a.HeroX > 1 || a.HeroY < 0 || a.HeroY > 1 {
		return errors.New("图片主体位置必须在 0 到 1 之间")
	}
	for _, id := range []string{a.Hero, a.Rail, a.Detail, a.Fallback} {
		if id != "" && !ValidID(id) {
			return errors.New("图片必须选择已上传的素材")
		}
		if strings.HasPrefix(id, "builtin-") && BuiltinArtwork()[id] == "" {
			return errors.New("内置素材不存在")
		}
	}
	if required && (a.Hero == "" || a.Rail == "") {
		return errors.New("发布前请配置首页海报和侧栏封面")
	}
	return nil
}

func (d Definition) Validate(publishing bool) error {
	if !validText(d.Name, 1, 64) || !validText(d.Title, 1, 12) || !validText(d.Subtitle, 0, 120) || !validText(d.Description, 0, 4000) {
		return errors.New("请填写名称和标题，并检查文案长度")
	}
	if d.Kind != "STANDARD" && d.Kind != "LIMITED" && d.Kind != "EVENT" {
		return errors.New("卡池类型无效")
	}
	if d.TicketID != "" && !ValidCurrency(d.TicketID) {
		return errors.New("请选择有效的抽奖券")
	}
	if d.Sort < -100000 || d.Sort > 100000 || len(d.Rewards) > 100 {
		return errors.New("排序或奖品数量超出范围")
	}
	if d.Labels != nil && (!validText(d.Labels.Rail, 0, 32) || !validText(d.Labels.Single, 0, 16) || !validText(d.Labels.Ten, 0, 16)) {
		return errors.New("卡池列表或抽取按钮文案过长")
	}
	if len(d.Sections) > 32 {
		return errors.New("补给详情最多允许 32 段")
	}
	for _, section := range d.Sections {
		if !validText(section.Title, 1, 80) || !validText(section.Body, 1, 2000) {
			return errors.New("请检查补给详情的标题和正文")
		}
	}
	if err := d.Art.Validate(publishing); err != nil {
		return err
	}
	if d.TicketArt != "" {
		if err := (Artwork{Detail: d.TicketArt}).Validate(false); err != nil {
			return err
		}
	}
	if d.Kind == "STANDARD" {
		if d.EndsAt != nil || d.SeriesID != "" {
			return errors.New("常驻池长期开放，不设置结束时间或活动编号")
		}
	} else {
		if publishing && (d.StartsAt == nil || d.EndsAt == nil) {
			return errors.New("限定和活动池必须设置开始、结束时间")
		}
		if d.StartsAt != nil && d.EndsAt != nil && !d.EndsAt.After(*d.StartsAt) {
			return errors.New("结束时间必须晚于开始时间")
		}
	}
	if d.VisibleAt != nil && d.StartsAt != nil && d.VisibleAt.After(*d.StartsAt) {
		return errors.New("预告时间不能晚于开始时间")
	}
	if d.Kind == "EVENT" {
		if !ValidID(d.SeriesID) || d.EventGoldPPM < 1 || d.EventGoldPPM > Million || d.EventPity < 1 || d.EventPity > 10000 {
			return errors.New("请填写活动概率、保底和稳定活动编号")
		}
	} else if d.EventGoldPPM != 0 || d.EventPity != 0 || d.SeriesID != "" {
		return errors.New("普通卡池使用固定规则，不接受活动配置")
	}
	seen := map[string]bool{}
	top, other := 0, 0
	for _, r := range d.Rewards {
		if !ValidID(r.Key) || seen[r.Key] || !validText(r.Name, 1, 128) || r.Stars < 4 || r.Stars > 6 || r.Weight < 1 || r.Weight > Million {
			return errors.New("奖品名称、品质、权重或重复编号不正确")
		}
		seen[r.Key] = true
		if r.ModelKey != "" && !ValidID(r.ModelKey) || r.GunID != "" && !namespaced.MatchString(r.GunID) {
			return errors.New("武器展示资源标识无效")
		}
		if err := r.Art.Validate(false); err != nil {
			return err
		}
		if r.Stars == 6 {
			top++
			if d.Kind != "STANDARD" && !r.UP {
				return errors.New("限定和活动的全部 6 星奖品必须属于 UP")
			}
		} else {
			other++
			if r.UP {
				return errors.New("UP 奖品必须是最高品质")
			}
		}
		if publishing && (!namespaced.MatchString(r.ItemRef) || r.ItemRevision < 1 || !hash.MatchString(r.PayloadSHA256) || r.InventoryDomain == "" || len(r.Servers) == 0 || len(r.Servers) > 32) {
			return errors.New("发布前请选择有效的物品库版本和交付范围")
		}
	}
	if publishing && (top == 0 || other == 0) {
		return errors.New("请同时配置6 星奖品和普通品质奖品")
	}
	return nil
}
