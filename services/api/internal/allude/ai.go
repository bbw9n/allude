package allude

import (
	"math"
	"strconv"
	"strings"
)

type AnalysisResult struct {
	Embedding []float64
	Concepts  []string
	Notes     []string
}

type AIProvider interface {
	AnalyzeThought(content string) (*AnalysisResult, error)
	EmbedQuery(content string) ([]float64, error)
}

type StubAIProvider struct{}

var stopWords = map[string]struct{}{
	"about": {}, "after": {}, "again": {}, "against": {}, "being": {},
	"between": {}, "because": {}, "could": {}, "first": {}, "found": {},
	"ideas": {}, "their": {}, "there": {}, "these": {}, "those": {},
	"thought": {}, "through": {}, "under": {}, "using": {}, "while": {},
	"would": {},
}

func (provider *StubAIProvider) AnalyzeThought(content string) (*AnalysisResult, error) {
	concepts := extractKeywords(content)
	notes := []string{"Related-thought candidates scored by semantic similarity"}
	if len(concepts) > 0 {
		notes = append([]string{strings.Join([]string{"Extracted", strconv.Itoa(len(concepts)), "concepts"}, " ")}, notes...)
	} else {
		notes = []string{"No strong concepts detected yet"}
	}
	return &AnalysisResult{
		Embedding: createEmbedding(content),
		Concepts:  concepts,
		Notes:     notes,
	}, nil
}

func (provider *StubAIProvider) EmbedQuery(content string) ([]float64, error) {
	return createEmbedding(content), nil
}

func extractKeywords(content string) []string {
	parts := strings.FieldsFunc(strings.ToLower(content), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	seen := map[string]struct{}{}
	var concepts []string
	for _, token := range parts {
		if len(token) <= 3 {
			continue
		}
		if _, blocked := stopWords[token]; blocked {
			continue
		}
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		concepts = append(concepts, token)
		if len(concepts) == 6 {
			break
		}
	}
	return concepts
}

func createEmbedding(content string) []float64 {
	vector := make([]float64, 16)
	input := strings.ToLower(content)
	for index, character := range input {
		bucket := index % len(vector)
		vector[bucket] += float64(character) / 255.0
	}
	var magnitude float64
	for _, value := range vector {
		magnitude += value * value
	}
	if magnitude == 0 {
		return vector
	}
	magnitude = math.Sqrt(magnitude)
	for index, value := range vector {
		vector[index] = value / magnitude
	}
	return vector
}
