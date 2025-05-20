// Package censor provides lexical content filtering and validation.

// Important notice: Test data files contain examples of explicit language
// and offensive terms required for pattern validation. These examples:
// - Are intentionally provocative to test edge cases
// - Do not represent the author's views
// - Should be treated as technical test artifacts only

// If you find such content disturbing or prefer to avoid exposure
// to sensitive language patterns:
// 1. Do not inspect the 'test_data' directory
// 2. Avoid reviewing test case literals
package censor

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Word struct {
	Text       string   `json:"text"`
	Pattern    string   `json:"pattern"`
	Exceptions []string `json:"exceptions"`

	regexPattern *regexp.Regexp
}

type Censor struct {
	bannedWords []Word
}

// New returns an empty Censor instance.
func New() *Censor {
	return &Censor{}
}

// LoadFromJSON loads banned words from a JSON file and compiles regexes.
func (c *Censor) LoadFromJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var words []Word
	if err := json.Unmarshal(data, &words); err != nil {
		return err
	}

	for i, word := range words {
		words[i].regexPattern, err = regexp.Compile(word.Pattern)
		if err != nil {
			return fmt.Errorf("failed to compile pattern %q: %w", word.Pattern, err)
		}
	}

	c.bannedWords = words
	return nil
}

func normalize(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "ё", "е")
	return strings.TrimSpace(text)
}

// Check scans comment for banned vocabulary using case-insensitive matching and Unicode normalization.
// Returns true if any word:
//   - Matches prohibited pattern(s)
//   - Isn't explicitly allowed in exceptions
func (c *Censor) Check(comment string) bool {
	normalized := normalize(comment)
	words := strings.Fields(normalized)

	for _, w := range words {
		for _, banned := range c.bannedWords {
			match := banned.regexPattern.FindString(w)
			if match == "" {
				continue
			}

			isException := false
			for _, exc := range banned.Exceptions {
				if exc == match {
					isException = true
					break
				}
			}

			if !isException {
				return true
			}
		}
	}

	return false
}
