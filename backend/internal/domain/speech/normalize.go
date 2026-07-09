// Package speech holds pure scoring rules for spoken and written answers
// (research R8): WER-based similarity and per-word typo tolerance.
package speech

import "strings"

// contractions expanded before scoring so "I'm" == "I am" (research R8).
var contractions = map[string]string{
	"i'm": "i am", "you're": "you are", "he's": "he is", "she's": "she is",
	"it's": "it is", "we're": "we are", "they're": "they are",
	"i've": "i have", "you've": "you have", "we've": "we have", "they've": "they have",
	"i'll": "i will", "you'll": "you will", "he'll": "he will", "she'll": "she will",
	"we'll": "we will", "they'll": "they will", "it'll": "it will",
	"i'd": "i would", "you'd": "you would", "he'd": "he would", "she'd": "she would",
	"we'd": "we would", "they'd": "they would",
	"don't": "do not", "doesn't": "does not", "didn't": "did not",
	"can't": "cannot", "couldn't": "could not", "won't": "will not",
	"wouldn't": "would not", "shouldn't": "should not", "isn't": "is not",
	"aren't": "are not", "wasn't": "was not", "weren't": "were not",
	"haven't": "have not", "hasn't": "has not", "hadn't": "had not",
	"let's": "let us", "that's": "that is", "there's": "there is",
	"what's": "what is", "who's": "who is", "where's": "where is",
}

// Normalize lowercases, expands common contractions, strips punctuation and
// collapses whitespace, returning the token list used for scoring.
func Normalize(s string) []string {
	fields := strings.Fields(strings.ToLower(s))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.Trim(f, ".,!?;:\"()[]{}…—–")
		if exp, ok := contractions[f]; ok {
			out = append(out, strings.Fields(exp)...)
			continue
		}
		f = strings.Map(func(r rune) rune {
			if r == '\'' || r == '’' {
				return -1
			}
			return r
		}, f)
		f = strings.Trim(f, ".,!?;:\"()[]{}…—–")
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

// levenshtein is the character-level edit distance between two words.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
