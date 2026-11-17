// Command fixture exports an anonymized test fixture from the synced database:
// the segment-tagged projects, their tasks and the labels, users and initiatives
// they reference. Titles and names are replaced; segment labels, dates,
// estimates, states and structure are kept, so the counting rules see real data.
//
//	DATABASE_URL=... go run ./cmd/fixture > internal/report/testdata/fixture.json
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"linear-dashboard/backend/internal/report"
	"linear-dashboard/backend/internal/store"
)

// Fixture is the on-disk shape read by the golden test.
type Fixture struct {
	Teams       []report.Team       `json:"teams"`
	Users       []report.User       `json:"users"`
	Labels      []report.Label      `json:"labels"`
	Initiatives []report.Initiative `json:"initiatives"`
	Projects    []report.Project    `json:"projects"`
	Tasks       []report.Task       `json:"tasks"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fixture:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	st, err := store.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer st.Close()
	ds, err := st.Load(ctx)
	if err != nil {
		return err
	}
	if ds == nil {
		return fmt.Errorf("no synced data")
	}

	var f Fixture
	projectIDs := map[string]bool{}
	var ids []string
	for id, p := range ds.Projects {
		if len(p.Segments) > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	usedLabels, usedInits, usedUsers, usedTeams := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for n, id := range ids {
		p := *ds.Projects[id]
		projectIDs[id] = true
		p.Name = fmt.Sprintf("Project %d", n+1)
		p.URL = ""
		p.Segments = nil
		for _, l := range p.LabelIDs {
			usedLabels[l] = true
		}
		for _, i := range p.InitiativeIDs {
			usedInits[i] = true
		}
		f.Projects = append(f.Projects, p)
	}
	for _, t := range ds.Tasks {
		if !projectIDs[t.ProjectID] {
			continue
		}
		t.Title = "Task " + t.Identifier
		t.URL = ""
		for _, l := range t.LabelIDs {
			usedLabels[l] = true
		}
		usedUsers[t.AssigneeID] = true
		usedTeams[t.TeamID] = true
		t.HasEstimatedDescendant, t.EffectivePts, t.MissingEstimate, t.TaskSegments = false, 0, false, nil
		f.Tasks = append(f.Tasks, t)
	}
	sort.Slice(f.Tasks, func(i, j int) bool { return f.Tasks[i].ID < f.Tasks[j].ID })

	n := 0
	for _, id := range sortedKeys(usedUsers) {
		if u, ok := ds.Users[id]; ok {
			n++
			f.Users = append(f.Users, report.User{ID: u.ID, Name: fmt.Sprintf("User %d", n), DisplayName: fmt.Sprintf("user.%d", n), Active: u.Active})
		}
	}
	for _, id := range sortedKeys(usedTeams) {
		if t, ok := ds.Teams[id]; ok {
			f.Teams = append(f.Teams, report.Team{ID: t.ID, Key: t.Key, Name: t.Key, EstimationType: t.EstimationType})
		}
	}
	n = 0
	for _, id := range sortedKeys(usedLabels) {
		l, ok := ds.Labels[id]
		if !ok {
			continue
		}
		if report.SegmentOf(l.Name) == "" {
			n++
			l.Name = fmt.Sprintf("Label %d", n)
			if l.Group != "" {
				l.Group = "Group"
			}
		}
		f.Labels = append(f.Labels, l)
	}
	n = 0
	for _, id := range sortedKeys(usedInits) {
		ini, ok := ds.Initiatives[id]
		if !ok {
			continue
		}
		n++
		name := fmt.Sprintf("Initiative %d", n)
		if ini.Visible {
			name = "[W] " + name
		}
		f.Initiatives = append(f.Initiatives, report.Initiative{ID: ini.ID, Name: name, Status: ini.Status, CreatedAt: ini.CreatedAt})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	return enc.Encode(f)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
