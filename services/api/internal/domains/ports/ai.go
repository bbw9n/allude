package ports

type AnalysisResult struct {
	Embedding []float64
	Concepts  []string
	Notes     []string
}

type AIProvider interface {
	AnalyzeThought(content string) (*AnalysisResult, error)
	EmbedQuery(content string) ([]float64, error)
}
