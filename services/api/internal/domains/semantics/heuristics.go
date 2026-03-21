package semantics

import (
	"strings"

	"github.com/bbw9n/allude/services/api/internal/domains/models"
	"github.com/bbw9n/allude/services/api/internal/pkgs/shared"
)

func LexicalScore(query, content string, concepts []*models.Concept) float64 {
	if query == "" {
		return 0
	}

	score := 0.0
	contentLower := strings.ToLower(content)
	queryLower := strings.ToLower(query)
	if strings.Contains(contentLower, queryLower) {
		score += 1.0
	}

	for _, token := range strings.Fields(queryLower) {
		if strings.Contains(contentLower, token) {
			score += 0.2
		}
		for _, concept := range concepts {
			if strings.Contains(strings.ToLower(concept.CanonicalName), token) {
				score += 0.15
			}
		}
	}

	if score > 1 {
		return 1
	}
	return score
}

func QualityScore(thought *models.Thought) float64 {
	score := 0.2
	score += float64(len(thought.Collections)) * 0.2
	score += float64(len(thought.Links)) * 0.1
	if score > 1 {
		return 1
	}
	return score
}

func DominantConceptName(thought *models.Thought) string {
	if len(thought.Concepts) == 0 {
		return ""
	}
	return thought.Concepts[0].CanonicalName
}

func UniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := shared.NormalizeConceptName(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, value)
	}
	return result
}

func AppendUniqueString(values []string, candidate string) []string {
	for _, value := range values {
		if value == candidate {
			return values
		}
	}
	return append(values, candidate)
}

func Slugify(value string) string {
	return strings.ReplaceAll(shared.NormalizeConceptName(value), " ", "-")
}

func NormalizedPair(left, right string) string {
	if left < right {
		return left + ":" + right
	}
	return right + ":" + left
}

func MaxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func CombinedRelationshipScore(source, target *models.Thought) float64 {
	base := shared.CosineSimilarity(source.CurrentVersion.Embedding, target.CurrentVersion.Embedding)
	overlap := ConceptOverlap(source, target)
	return (base * 0.7) + (overlap * 0.3)
}

func ConceptOverlap(source, target *models.Thought) float64 {
	if len(source.Concepts) == 0 || len(target.Concepts) == 0 {
		return 0
	}
	sourceConcepts := map[string]struct{}{}
	for _, concept := range source.Concepts {
		sourceConcepts[concept.CanonicalName] = struct{}{}
	}
	matches := 0
	for _, concept := range target.Concepts {
		if _, exists := sourceConcepts[concept.CanonicalName]; exists {
			matches++
		}
	}
	if matches == 0 {
		return 0
	}
	total := len(source.Concepts)
	if len(target.Concepts) > total {
		total = len(target.Concepts)
	}
	return float64(matches) / float64(total)
}

func RelationTypeForThoughts(source, target *models.Thought) models.RelationType {
	sourceContent := strings.ToLower(source.CurrentVersion.Content)
	targetContent := strings.ToLower(target.CurrentVersion.Content)
	if strings.Contains(sourceContent, "not") || strings.Contains(targetContent, "not") || strings.Contains(sourceContent, "against") {
		return models.RelationContradict
	}
	sourceConcepts := map[string]struct{}{}
	for _, concept := range source.Concepts {
		sourceConcepts[concept.ID] = struct{}{}
	}
	for _, concept := range target.Concepts {
		if _, exists := sourceConcepts[concept.ID]; exists {
			return models.RelationExtends
		}
	}
	return models.RelationRelated
}
