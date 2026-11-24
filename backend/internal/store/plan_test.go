package store

import (
	"slices"
	"testing"

	"linear-dashboard/backend/internal/report"
)

func TestPlanFrozen(t *testing.T) {
	est := func(task, old, new string) report.FrozenChange {
		return report.FrozenChange{Kind: report.ChangeEstimate, TaskID: task, Old: old, New: new, FrozenBy: "task"}
	}
	pending := []FrozenChangeRow{
		{ID: 1, FrozenChange: est("same", "3", "5")},
		{ID: 2, FrozenChange: est("edited-again", "3", "5")},
		{ID: 3, FrozenChange: est("changed-back", "3", "5")},
		{ID: 4, FrozenChange: est("not-refetched", "3", "5")},
	}
	held := []report.FrozenChange{est("same", "3", "5"), est("edited-again", "3", "8"), est("new", "1", "2")}
	refetched := map[string]bool{"same": true, "edited-again": true, "changed-back": true, "new": true}

	plan := planFrozen(pending, FrozenUpdate{Held: held, Rechecked: func(c report.FrozenChange) bool { return refetched[c.TaskID] }})

	if len(plan.update) != 1 || plan.update[0].ID != 2 || plan.update[0].New != "8" {
		t.Errorf("update = %+v, want change 2 updated to 8", plan.update)
	}
	if !slices.Equal(plan.withdraw, []int64{3}) {
		t.Errorf("withdraw = %v, want [3]: changed back in Linear; 4 was not looked at", plan.withdraw)
	}
	if len(plan.insert) != 1 || plan.insert[0].TaskID != "new" {
		t.Errorf("insert = %+v, want the new change only", plan.insert)
	}

	full := planFrozen(pending, FrozenUpdate{Held: held})
	if !slices.Equal(full.withdraw, []int64{3, 4}) {
		t.Errorf("full sync withdraw = %v, want [3 4]: a full sync looks at every task", full.withdraw)
	}
}
