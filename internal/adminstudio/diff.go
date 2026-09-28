package adminstudio

import (
	"errors"
	"sort"
)

var ErrDiffTooLarge = errors.New("revision diff exceeds comparison limit")

type DiffKind string

const (
	DiffContext DiffKind = "context"
	DiffAdded   DiffKind = "added"
	DiffRemoved DiffKind = "removed"
)

type DiffLine struct {
	Kind DiffKind
	Text string
}

// CompareLines uses bounded LCS so rendering untrusted revision snapshots has
// predictable memory use. The 400-line limit caps the matrix at 160k cells.
func CompareLines(before, after []string) ([]DiffLine, error) {
	if len(before) > 400 || len(after) > 400 {
		return nil, ErrDiffTooLarge
	}
	width := len(after) + 1
	lcs := make([]uint16, (len(before)+1)*width)
	for i := len(before) - 1; i >= 0; i-- {
		for j := len(after) - 1; j >= 0; j-- {
			index := i*width + j
			if before[i] == after[j] {
				lcs[index] = lcs[(i+1)*width+j+1] + 1
			} else if lcs[(i+1)*width+j] >= lcs[i*width+j+1] {
				lcs[index] = lcs[(i+1)*width+j]
			} else {
				lcs[index] = lcs[i*width+j+1]
			}
		}
	}
	result := make([]DiffLine, 0, len(before)+len(after))
	for i, j := 0, 0; i < len(before) || j < len(after); {
		switch {
		case i < len(before) && j < len(after) && before[i] == after[j]:
			result = append(result, DiffLine{Kind: DiffContext, Text: before[i]})
			i++
			j++
		case i < len(before) && (j == len(after) || lcs[(i+1)*width+j] >= lcs[i*width+j+1]):
			result = append(result, DiffLine{Kind: DiffRemoved, Text: before[i]})
			i++
		default:
			result = append(result, DiffLine{Kind: DiffAdded, Text: after[j]})
			j++
		}
	}
	return result, nil
}

type FieldDiff struct{ Field, Before, After string }

func CompareFields(before, after map[string]string) []FieldDiff {
	keys := make([]string, 0, len(before)+len(after))
	seen := map[string]bool{}
	for key := range before {
		seen[key] = true
		keys = append(keys, key)
	}
	for key := range after {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	result := make([]FieldDiff, 0, len(keys))
	for _, key := range keys {
		if before[key] != after[key] {
			result = append(result, FieldDiff{Field: key, Before: before[key], After: after[key]})
		}
	}
	return result
}
