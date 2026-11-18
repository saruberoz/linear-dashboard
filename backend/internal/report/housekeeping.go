package report

import (
	"sort"

	pb "linear-dashboard/backend/api/dashboard/v1"
)

// ProjectCanceled is Linear's project state for canceled projects. Project
// housekeeping pages hide them by default, so menu counts leave them out.
const ProjectCanceled = "canceled"

// notCanceled counts the projects that are not canceled.
func notCanceled(ps []*Project) int32 {
	var n int32
	for _, p := range ps {
		if p.State != ProjectCanceled {
			n++
		}
	}
	return n
}

func (d *Dataset) tasksWithoutProject(r Range) []*Task {
	var out []*Task
	for _, t := range d.tasksByProject[""] {
		if t.Status() != StatusCanceled && r.InRange(t) {
			out = append(out, t)
		}
	}
	return out
}

// TasksWithoutProject groups the open (not canceled) tasks in r that have no project by assignee.
func (d *Dataset) TasksWithoutProject(r Range) *pb.ListTasksWithoutProjectResponse {
	tasks := d.tasksWithoutProject(r)
	return &pb.ListTasksWithoutProjectResponse{Total: int32(len(tasks)), Groups: d.byAssignee(tasks)}
}

func (d *Dataset) projectsWithoutTimeline(r Range) []*Project {
	var out []*Project
	for _, p := range d.Projects {
		if (p.StartDate == "" || p.TargetDate == "") && d.ProjectVisible(p, r) {
			out = append(out, p)
		}
	}
	return out
}

// initiativesWithoutTarget returns the open [W]/[WS] initiatives with no target date.
func (d *Dataset) initiativesWithoutTarget() []*Initiative {
	var out []*Initiative
	for _, ini := range d.Initiatives {
		if ini.Visible && ini.TargetDate == "" && ini.Status != "Completed" {
			out = append(out, ini)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// MissingTimeline lists projects active in r without a start or target date,
// largest effort first, and open [W]/[WS] initiatives without a target date.
func (d *Dataset) MissingTimeline(r Range) *pb.ListMissingTimelineResponse {
	res := &pb.ListMissingTimelineResponse{}
	for _, p := range d.projectsWithoutTimeline(r) {
		res.Projects = append(res.Projects, d.projectRow(p, r, ""))
	}
	sort.SliceStable(res.Projects, func(i, j int) bool {
		a, b := res.Projects[i], res.Projects[j]
		if a.Pts != b.Pts {
			return a.Pts > b.Pts
		}
		return a.Name < b.Name
	})
	for _, ini := range d.initiativesWithoutTarget() {
		res.Initiatives = append(res.Initiatives, &pb.TimelineInitiative{Id: ini.ID, Name: ini.Name, Url: ini.URL,
			Status: ini.Status, Projects: int32(len(ini.ProjectIDs))})
	}
	return res
}

func (d *Dataset) projectsWithoutTasks() []*Project {
	var out []*Project
	for _, p := range d.Projects {
		if len(d.tasksByProject[p.ID]) == 0 {
			out = append(out, p)
		}
	}
	sortProjects(out)
	return out
}

// ProjectsWithoutTasks lists projects with no tasks at all, whatever the range.
// r only shapes the rows, which are all zero.
func (d *Dataset) ProjectsWithoutTasks(r Range) *pb.ListProjectsWithoutTasksResponse {
	res := &pb.ListProjectsWithoutTasksResponse{}
	for _, p := range d.projectsWithoutTasks() {
		res.Projects = append(res.Projects, d.projectRow(p, r, ""))
	}
	return res
}
