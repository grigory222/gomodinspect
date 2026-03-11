package domain

// ModuleInfo содержит основную информацию о Go-модуле
type ModuleInfo struct {
	Name      string       `json:"name"`
	GoVersion string       `json:"go_version"`
	Deps      []Dependency `json:"dependencies"`
}

// Dependency описывает одну зависимость Go-модуля.
type Dependency struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpdateAvail    bool   `json:"update_available"`
}
