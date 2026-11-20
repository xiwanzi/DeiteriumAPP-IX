// Package supply defines the published weapon supply rules independently of transport and storage.
package supply

import (
	"crypto/rand"
	"errors"
	"math/big"
)

const Million = 1_000_000
const MaxBalance int64 = 1_000_000_000
const RulesVersion = "2026-11-17"

type Rule struct {
	GoldPPM int `json:"goldPpm"`
	Minimum int `json:"minimum"`
	Maximum int `json:"maximum"`
}

type Pity struct {
	Count  int   `json:"count"`
	Target int   `json:"target"`
	Cycle  int64 `json:"cycle"`
}

type Roll struct {
	Reward     Reward `json:"reward"`
	Before     int    `json:"before"`
	After      int    `json:"after"`
	Target     int    `json:"target"`
	Guaranteed bool   `json:"guaranteed"`
	Cycle      int64  `json:"cycle"`
}

// Random must return an unbiased value in [0, limit). An entropy failure aborts the transaction.
type Random func(limit int) (int, error)

func SecureRandom(limit int) (int, error) {
	if limit <= 0 {
		return 0, errors.New("invalid random limit")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(limit)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

func (d Definition) Rule(scheme string) (Rule, string, string, error) {
	switch d.Kind {
	case "STANDARD":
		if scheme == "80" {
			return d.frozenRule(scheme, Rule{5000, 80, 80}), "standard", "STANDARD", nil
		}
	case "LIMITED":
		if scheme == "80" {
			return d.frozenRule(scheme, Rule{5000, 80, 80}), "limited:80", "LIMITED", nil
		}
		if scheme == "120" {
			return d.frozenRule(scheme, Rule{12000, 100, 120}), "limited:120", "LIMITED", nil
		}
	case "EVENT":
		if scheme == "event" && d.SeriesID != "" {
			return d.frozenRule(scheme, Rule{d.EventGoldPPM, d.EventPity, d.EventPity}), "event:" + d.SeriesID, "LIMITED", nil
		}
	}
	return Rule{}, "", "", errors.New("抽取方案不属于当前卡池")
}

func (d Definition) frozenRule(scheme string, fallback Rule) Rule {
	if rule, ok := d.Rules[scheme]; ok {
		return rule
	}
	return fallback
}

// Published revisions retain the actual numerical rules even after a later program release.
func (d *Definition) FreezeRules() {
	d.Rules = map[string]Rule{}
	for _, scheme := range d.Schemes() {
		rule, _, _, err := d.Rule(scheme)
		if err == nil {
			d.Rules[scheme] = rule
		}
	}
}

func (r Rule) NextTarget(random Random) (int, error) {
	if r.Minimum < 1 || r.Maximum < r.Minimum || r.Maximum > 10000 {
		return 0, errors.New("保底配置无效")
	}
	if r.Minimum == r.Maximum {
		return r.Minimum, nil
	}
	n, err := random(r.Maximum - r.Minimum + 1)
	if err != nil {
		return 0, err
	}
	if n < 0 || n > r.Maximum-r.Minimum {
		return 0, errors.New("invalid random result")
	}
	return r.Minimum + n, nil
}

// Draw advances one shared pity group in order. The caller persists the returned state atomically
// with the ticket debit, results and delivery intents; it must not redraw after a lost response.
func Draw(rule Rule, state Pity, rewards []Reward, quantity int, random Random) ([]Roll, Pity, error) {
	if quantity != 1 && quantity != 10 || rule.GoldPPM < 1 || rule.GoldPPM > Million {
		return nil, state, errors.New("抽取数量或概率无效")
	}
	if state.Target == 0 {
		var err error
		state.Target, err = rule.NextTarget(random)
		if err != nil {
			return nil, state, err
		}
	}
	if state.Target < rule.Minimum || state.Target > rule.Maximum || state.Count < 0 || state.Count >= state.Target {
		return nil, state, errors.New("保底进度与已发布规则不一致")
	}
	rolls := make([]Roll, 0, quantity)
	for i := 0; i < quantity; i++ {
		before, target, cycle := state.Count, state.Target, state.Cycle
		guaranteed := before+1 >= target
		gold := guaranteed
		if !gold {
			n, err := random(Million)
			if err != nil {
				return nil, state, err
			}
			if n < 0 || n >= Million {
				return nil, state, errors.New("invalid random result")
			}
			gold = n < rule.GoldPPM
		}
		reward, err := choose(rewards, gold, random)
		if err != nil {
			return nil, state, err
		}
		state.Count++
		if gold {
			state.Count = 0
			state.Cycle++
			state.Target, err = rule.NextTarget(random)
			if err != nil {
				return nil, state, err
			}
		}
		rolls = append(rolls, Roll{reward, before, state.Count, target, guaranteed, cycle})
	}
	return rolls, state, nil
}

func choose(rewards []Reward, gold bool, random Random) (Reward, error) {
	total := 0
	for _, r := range rewards {
		if (r.Stars == 6) == gold {
			total += r.Weight
		}
	}
	if total <= 0 {
		return Reward{}, errors.New("当前品质没有可用奖品")
	}
	n, err := random(total)
	if err != nil {
		return Reward{}, err
	}
	if n < 0 || n >= total {
		return Reward{}, errors.New("invalid random result")
	}
	for _, r := range rewards {
		if (r.Stars == 6) != gold {
			continue
		}
		if n < r.Weight {
			return r, nil
		}
		n -= r.Weight
	}
	return Reward{}, errors.New("奖品权重无效")
}
