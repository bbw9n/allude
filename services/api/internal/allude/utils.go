package allude

import (
	"crypto/rand"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"
)

const ViewerID = "00000000-0000-0000-0000-000000000001"

var idCounter uint64

func createID(prefix string) string {
	value := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s_%x", prefix, value)
}

func newUUID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16],
	)
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func normalizeConceptName(input string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(input))), " ")
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot float64
	var normA float64
	var normB float64
	for index := range a {
		dot += a[index] * b[index]
		normA += a[index] * a[index]
		normB += b[index] * b[index]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func clamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
