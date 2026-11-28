package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

type samplePoolSeed struct {
	ID         string
	Definition supply.Definition
}

func samplePoolCommonRewards() []supply.Reward {
	type spec struct {
		key, name, gunId string
		stars, weight    int
	}
	list := []spec{
		{"tacz-ai_awp", "精密国际 AWM 狙击步枪", "tacz:ai_awp", 5, 200},
		{"tacz-m249", "M249 机枪", "tacz:m249", 5, 200},
		{"tacz-scar_h", "SCAR-H 战斗步枪", "tacz:scar_h", 5, 200},
		{"tacz-m1014", "M1014 战斗霰弹枪", "tacz:m1014", 5, 200},
		{"tacz-ak47", "AKM 突击步枪", "tacz:ak47", 4, 1500},
		{"tacz-m4a1", "M4A1 卡宾枪", "tacz:m4a1", 4, 1500},
		{"tacz-hk_mp5a5", "MP5A5 冲锋枪", "tacz:hk_mp5a5", 4, 1500},
		{"tacz-glock_17", "格洛克 17 手枪", "tacz:glock_17", 4, 1500},
		{"tacz-ump45", "UMP45 冲锋枪", "tacz:ump45", 4, 1500},
		{"tacz-sks_tactical", "SKS 战术步枪", "tacz:sks_tactical", 4, 1500},
	}
	rewards := make([]supply.Reward, 0, len(list))
	for _, item := range list {
		rewards = append(rewards, supply.Reward{
			Key:      item.key,
			Name:     item.name,
			GunID:    item.gunId,
			ModelKey: item.key,
			Stars:    item.stars,
			Weight:   item.weight,
			Art: supply.Artwork{
				Detail:   "builtin-detail-" + item.key,
				Fallback: "builtin-fallback-" + item.key,
			},
		})
	}
	return rewards
}

func SampleSupplyPools(now time.Time) []samplePoolSeed {
	start := now.UTC().Truncate(time.Second)
	end := start.Add(90 * 24 * time.Hour)
	common := samplePoolCommonRewards()

	up191 := supply.Definition{
		Name:        "墨影流光",
		Title:       "墨影流光",
		Subtitle:    "当期限定 · 突击步枪 UP",
		Description: "QBZ-191「印花集」为本期主推枪械。使用限定补给兑换券参与抽取，支持 80 抽 / 120 抽两套共享保底方案。",
		Kind:        "LIMITED",
		StartsAt:    &start,
		EndsAt:      &end,
		Sort:        300,
		TicketID:    "LIMITED",
		TicketArt:   "builtin-ticket",
		Labels:      &supply.PoolLabels{Rail: ""},
		Art: supply.Artwork{
			Hero:  "builtin-hero-up",
			Rail:  "builtin-rail-up",
			Theme: "violet",
			HeroX: 0.5,
			HeroY: 0.5,
		},
		Sections: []supply.PoolSection{
			{Title: "抽取规则与保底", Body: "每抽 6 星概率 {goldProb}，最多 {maxPity} 抽必得当期 6 星 UP（限定 120 方案在 {minPity}～{maxPity} 抽间随机确定保底）；同方案跨期共享保底进度。\n当前使用「{ticketName}」，单抽消耗 1 张、十连消耗 10 张；兑换单价：{price}。"},
			{Title: "奖励交付与记录", Body: "每件奖励优先进入当前背包，空间不足或玩家离线时自动按抽取订单合并寄送至游戏邮箱；可在上方「补给记录」查看跨服历史抽取记录。"},
		},
		Rewards: append([]supply.Reward{{
			Key:      "printstream",
			Name:     "QBZ-191「印花集」",
			GunID:    "deuterium_ix:qbz_191_printstream",
			ModelKey: "printstream",
			Stars:    6,
			Weight:   100,
			UP:       true,
			Art: supply.Artwork{
				Detail:   "builtin-detail-printstream",
				Fallback: "builtin-fallback-printstream",
			},
		}}, common...),
	}
	up191.FreezeRules()

	upVector := supply.Definition{
		Name:        "绯红回响",
		Title:       "绯红回响",
		Subtitle:    "当期限定 · 冲锋枪 UP",
		Description: "Vector .45「绯红禁域」为本期主推枪械。使用限定补给兑换券参与抽取，与同档限定池共享保底计数。",
		Kind:        "LIMITED",
		StartsAt:    &start,
		EndsAt:      &end,
		Sort:        200,
		TicketID:    "LIMITED",
		TicketArt:   "builtin-ticket",
		Labels:      &supply.PoolLabels{Rail: ""},
		Art: supply.Artwork{
			Hero:  "builtin-hero-vector",
			Rail:  "builtin-rail-vector",
			Theme: "red",
			HeroX: 0.5,
			HeroY: 0.5,
		},
		Sections: []supply.PoolSection{
			{Title: "抽取规则与保底", Body: "本池 6 星 100% 为当期主推。每抽 6 星概率 {goldProb}，最多 {maxPity} 抽必得当期 6 星 UP（限定 120 方案在 {minPity}～{maxPity} 抽间随机确定保底）；同方案跨期共享保底进度。\n当前使用「{ticketName}」，单抽消耗 1 张、十连消耗 10 张；兑换单价：{price}。"},
			{Title: "奖励交付与记录", Body: "每件奖励优先进入当前背包，空间不足或玩家离线时自动按抽取订单合并寄送至游戏邮箱；可在上方「补给记录」查看跨服历史抽取记录。"},
		},
		Rewards: append([]supply.Reward{{
			Key:      "scarlet",
			Name:     "Vector .45「绯红禁域」",
			GunID:    "deuterium_ix:vector45_flandre",
			ModelKey: "scarlet",
			Stars:    6,
			Weight:   100,
			UP:       true,
			Art: supply.Artwork{
				Detail:   "builtin-detail-scarlet",
				Fallback: "builtin-fallback-scarlet",
			},
		}}, common...),
	}
	upVector.FreezeRules()

	standard := supply.Definition{
		Name:        "常驻补给",
		Title:       "雨夜鸣金",
		Subtitle:    "常驻双主推武器补给",
		Description: "DB-4 乌萨斯「雨夜鎏金」与 FN EVOLYS「完美冻结」为常驻双主推枪械。长期开放，使用常驻补给兑换券参与抽取。",
		Kind:        "STANDARD",
		StartsAt:    &start,
		Sort:        100,
		TicketID:    "STANDARD",
		TicketArt:   "builtin-ticket",
		Labels:      &supply.PoolLabels{Rail: ""},
		Art: supply.Artwork{
			Hero:  "builtin-hero-standard",
			Rail:  "builtin-rail-standard",
			Theme: "blue",
			HeroX: 0.5,
			HeroY: 0.5,
		},
		Sections: []supply.PoolSection{
			{Title: "抽取规则与保底", Body: "常驻补给每抽 6 星概率 {goldProb}，最多 {maxPity} 抽必得 6 星，获得 6 星后重新计数。\n当前使用「{ticketName}」，单抽消耗 1 张、十连消耗 10 张；兑换单价：{price}。"},
			{Title: "奖励交付与记录", Body: "每件奖励优先进入当前背包，空间不足或玩家离线时自动按抽取订单合并寄送至游戏邮箱；可在上方「补给记录」查看跨服历史抽取记录。"},
		},
		Rewards: append([]supply.Reward{
			{
				Key:      "gilded-rain",
				Name:     "DB-4 乌萨斯「雨夜鎏金」",
				GunID:    "deuterium_ix:db4_gilded_rain",
				ModelKey: "gilded-rain",
				Stars:    6,
				Weight:   100,
				UP:       false,
				Art: supply.Artwork{
					Detail:   "builtin-detail-gilded-rain",
					Fallback: "builtin-fallback-gilded-rain",
				},
			},
			{
				Key:      "perfect-freeze",
				Name:     "FN EVOLYS「完美冻结」",
				GunID:    "cirno_ice:fn_evolys_cirno",
				ModelKey: "perfect-freeze",
				Stars:    6,
				Weight:   100,
				UP:       false,
				Art: supply.Artwork{
					Detail:   "builtin-detail-perfect-freeze",
					Fallback: "builtin-fallback-perfect-freeze",
				},
			},
		}, common...),
	}
	standard.FreezeRules()

	return []samplePoolSeed{
		{ID: "sp_sample_up_191", Definition: up191},
		{ID: "sp_sample_up_vector", Definition: upVector},
		{ID: "sp_sample_standard", Definition: standard},
	}
}

// SeedSampleSupplyPools idempotently inserts the 3 default editable sample pools without overwriting any existing pool.
func (s *Store) SeedSampleSupplyPools(ctx context.Context, actor string) ([]string, error) {
	if actor == "" {
		actor = "system:sample-seed"
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var settingsVersion int64
	if err := tx.QueryRowContext(ctx, "SELECT version FROM supply_settings WHERE id=1 FOR UPDATE").Scan(&settingsVersion); err != nil {
		return nil, err
	}
	var activeStandard int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM supply_pools WHERE kind='STANDARD' AND enabled=TRUE").Scan(&activeStandard); err != nil {
		return nil, err
	}
	created := []string{}
	now := time.Now().UTC()
	for _, seed := range SampleSupplyPools(now) {
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM supply_pools WHERE id=?", seed.ID).Scan(&exists); err != nil {
			return nil, err
		}
		if exists > 0 {
			continue
		}
		raw, err := json.Marshal(seed.Definition)
		if err != nil {
			return nil, err
		}
		enabled := true
		if seed.Definition.Kind == "STANDARD" && activeStandard > 0 {
			enabled = false
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO supply_pools(id,kind,series_id,draft_json,version,published_revision,enabled,starts_at,ends_at,visible_at,created_at,updated_at)
			VALUES(?,?,?,?,1,1,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`,
			seed.ID, seed.Definition.Kind, seed.Definition.SeriesID, raw, enabled, seed.Definition.StartsAt, seed.Definition.EndsAt, seed.Definition.VisibleAt); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO supply_revisions(pool_id,revision,definition_json,actor_id,created_at) VALUES(?,1,?,?,UTC_TIMESTAMP(6))`,
			seed.ID, raw, actor); err != nil {
			return nil, err
		}
		if seed.Definition.Kind == "STANDARD" && enabled {
			activeStandard++
		}
		created = append(created, seed.ID)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}
