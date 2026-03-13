package domain

import "time"

type ModuleInfo struct {
	Name       string       `json:"name"`
	GoVersion  string       `json:"go_version"`
	Deps       []Dependency `json:"dependencies"`
	AnalyzedAt time.Time    `json:"analyzed_at"`
}

type Dependency struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpdateAvail    bool   `json:"update_available"`
}
