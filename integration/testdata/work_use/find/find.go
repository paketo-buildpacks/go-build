package find

import (
	"github.com/sahilm/fuzzy"
)

func Fuzzy(pattern string, data ...string) []string {
	var matches []string
	for _, match := range fuzzy.Find(pattern, data) {
		matches = append(matches, match.Str)
	}
	return matches
}
