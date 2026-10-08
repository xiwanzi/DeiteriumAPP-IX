package store

import (
	"math/rand"
	"testing"
)

func TestCheckoutCouponAssignmentMatchesExhaustiveOptimum(t *testing.T) {
	r := rand.New(rand.NewSource(221))
	for n := 1; n <= 5; n++ {
		for m := 0; m <= 5; m++ {
			for trial := 0; trial < 30; trial++ {
				weights := make([][]int64, n)
				for i := range weights {
					weights[i] = make([]int64, m)
					for j := range weights[i] {
						weights[i][j] = int64(r.Intn(9))
					}
				}
				best := int64(0)
				var visit func(int, int, int64)
				visit = func(row, used int, total int64) {
					if row == n {
						if total > best {
							best = total
						}
						return
					}
					visit(row+1, used, total)
					for c, w := range weights[row] {
						if used&(1<<c) == 0 {
							visit(row+1, used|(1<<c), total+w)
						}
					}
				}
				visit(0, 0, 0)
				got, used := int64(0), map[int]bool{}
				for i, c := range checkoutCouponAssignmentV221(weights) {
					if c < 0 {
						continue
					}
					if used[c] {
						t.Fatal("coupon reused")
					}
					used[c] = true
					got += weights[i][c]
				}
				if got != best {
					t.Fatalf("weights=%v got=%d want=%d", weights, got, best)
				}
			}
		}
	}
}
