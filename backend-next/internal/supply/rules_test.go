package supply

import "testing"

var rewards = []Reward{{Key: "up", Stars: 6, Weight: 1, UP: true}, {Key: "ordinary", Stars: 4, Weight: 1}}

func TestEightyForcesGoldAndTenDrawsContinueAfterReset(t *testing.T) {
	rule, _, _, _ := (Definition{Kind: "LIMITED"}).Rule("80")
	rolls, pity, err := Draw(rule, Pity{Count: 77, Target: 80}, rewards, 10, func(n int) (int, error) { return n - 1, nil })
	if err != nil || len(rolls) != 10 || rolls[2].Reward.Key != "up" || !rolls[2].Guaranteed || pity.Count != 7 {
		t.Fatalf("%+v %+v %v", rolls, pity, err)
	}
}

func TestGoldChanceNeverIncreasesBeforeGuarantee(t *testing.T) {
	for _, kind := range []string{"STANDARD", "LIMITED"} {
		rule, _, _, _ := (Definition{Kind: kind}).Rule("80")
		for _, count := range []int{0, 69, 70, 78} {
			rolls, _, err := Draw(rule, Pity{Count: count, Target: 80}, rewards, 1, func(n int) (int, error) {
				if n == Million {
					return 5000, nil
				}
				return 0, nil
			})
			if err != nil || rolls[0].Reward.Key != "ordinary" {
				t.Fatalf("chance increased at %d: %+v %v", count, rolls, err)
			}
		}
	}
}

func TestHundredTwentyTargetIncludesBothEndpointsAndPersists(t *testing.T) {
	rule, scope, _, _ := (Definition{Kind: "LIMITED"}).Rule("120")
	if scope != "limited:120" || rule.GoldPPM != 12000 {
		t.Fatal(rule, scope)
	}
	for _, pick := range []int{0, 20} {
		calls := 0
		random := func(n int) (int, error) {
			if n == 21 {
				calls++
				return pick, nil
			}
			return n - 1, nil
		}
		_, state, err := Draw(rule, Pity{}, rewards, 1, random)
		if err != nil || state.Target != 100+pick || calls != 1 {
			t.Fatal(state, calls, err)
		}
		_, state, err = Draw(rule, state, rewards, 10, random)
		if err != nil || state.Count != 11 || state.Target != 100+pick || calls != 1 {
			t.Fatal("target rerolled", state, calls, err)
		}
		state.Count = state.Target - 1
		rolls, state, err := Draw(rule, state, rewards, 1, random)
		if err != nil || !rolls[0].Guaranteed || state.Count != 0 || calls != 2 {
			t.Fatal(rolls, state, calls, err)
		}
	}
}

func TestEarlyGoldResetsOnlyTheSuppliedGroup(t *testing.T) {
	rule, _, _, _ := (Definition{Kind: "LIMITED"}).Rule("120")
	rolls, state, err := Draw(rule, Pity{Count: 30, Target: 117, Cycle: 4}, rewards, 1, func(int) (int, error) { return 0, nil })
	if err != nil || rolls[0].Guaranteed || state.Count != 0 || state.Target != 100 || state.Cycle != 5 {
		t.Fatal(rolls, state, err)
	}
}

func TestPoolsShareSchemeScopesWhileEventsRemainSeparate(t *testing.T) {
	a, b := Definition{Kind: "LIMITED", Name: "A"}, Definition{Kind: "LIMITED", Name: "B"}
	_, a80, _, _ := a.Rule("80")
	_, b80, _, _ := b.Rule("80")
	_, a120, _, _ := a.Rule("120")
	if a80 != b80 || a80 == a120 {
		t.Fatal(a80, b80, a120)
	}
	event := Definition{Kind: "EVENT", SeriesID: "anniversary", EventGoldPPM: 10000, EventPity: 80}
	_, scope, currency, err := event.Rule("event")
	if err != nil || scope != "event:anniversary" || currency != "LIMITED" {
		t.Fatal(scope, currency, err)
	}
}
