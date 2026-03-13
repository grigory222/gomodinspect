package domain

import "time"

// AnalysisRecord -- модель результата анализа для репозитория, чтобы сохранять её в Redis
type AnalysisRecord struct {
	RepoURL    string    `json:"repo_url"`
	ModuleName string    `json:"module_name"`
	GoVersion  string    `json:"go_version"`
	AnalyzedAt time.Time `json:"analyzed_at"`
	DepsJSON   string    `json:"deps_json"`
}
