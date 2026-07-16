package engine

import (
	"fmt"
	"strconv"
	"strings"
)

func parseEvidencePath(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("path is empty")
	}

	index := 0
	readIdentifier := func() (string, error) {
		if index >= len(path) || !isIdentifierStart(path[index]) {
			return "", fmt.Errorf("expected identifier at byte %d", index+1)
		}
		start := index
		index++
		for index < len(path) && isIdentifierPart(path[index]) {
			index++
		}
		return path[start:index], nil
	}

	first, err := readIdentifier()
	if err != nil {
		return nil, err
	}
	segments := []string{first}
	for index < len(path) {
		switch path[index] {
		case '.':
			index++
			segment, readErr := readIdentifier()
			if readErr != nil {
				return nil, readErr
			}
			segments = append(segments, segment)
		case '[':
			index++
			if index >= len(path) || path[index] != '"' {
				return nil, fmt.Errorf("expected double-quoted literal map key at byte %d", index+1)
			}
			start := index
			index++
			escaped := false
			for index < len(path) {
				current := path[index]
				index++
				if escaped {
					escaped = false
					continue
				}
				if current == '\\' {
					escaped = true
					continue
				}
				if current == '"' {
					break
				}
			}
			if index > len(path) || index == 0 || path[index-1] != '"' {
				return nil, fmt.Errorf("unterminated literal map key")
			}
			key, unquoteErr := strconv.Unquote(path[start:index])
			if unquoteErr != nil {
				return nil, fmt.Errorf("invalid literal map key: %w", unquoteErr)
			}
			if key == "" {
				return nil, fmt.Errorf("literal map key must not be empty")
			}
			if index >= len(path) || path[index] != ']' {
				return nil, fmt.Errorf("expected ] after literal map key at byte %d", index+1)
			}
			index++
			segments = append(segments, key)
		default:
			return nil, fmt.Errorf("expected . or [ at byte %d", index+1)
		}
	}
	return segments, nil
}

func formatEvidencePath(segments []string) string {
	if len(segments) == 0 {
		return ""
	}
	var formatted strings.Builder
	formatted.WriteString(segments[0])
	for _, segment := range segments[1:] {
		if evidenceIdentifier(segment) {
			formatted.WriteByte('.')
			formatted.WriteString(segment)
			continue
		}
		formatted.WriteByte('[')
		formatted.WriteString(strconv.Quote(segment))
		formatted.WriteByte(']')
	}
	return formatted.String()
}

func appendEvidenceSegment(path, segment string) string {
	segments, err := parseEvidencePath(path)
	if err != nil {
		return path
	}
	return formatEvidencePath(append(segments, segment))
}

func evidenceIdentifier(value string) bool {
	if value == "" || !isIdentifierStart(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !isIdentifierPart(value[index]) {
			return false
		}
	}
	return true
}

func evidencePathHasPrefix(path, prefix string) bool {
	pathSegments, pathErr := parseEvidencePath(path)
	prefixSegments, prefixErr := parseEvidencePath(prefix)
	if pathErr != nil || prefixErr != nil || len(pathSegments) < len(prefixSegments) {
		return false
	}
	for index := range prefixSegments {
		if pathSegments[index] != prefixSegments[index] {
			return false
		}
	}
	return true
}

func evidencePathsOverlap(left, right string) bool {
	return evidencePathHasPrefix(left, right) || evidencePathHasPrefix(right, left)
}
