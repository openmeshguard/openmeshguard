package output

import (
	"fmt"
	"io"
	"strings"
)

// WriteScore renders the score fields already present in canonical JSON.
func WriteScore(writer io.Writer, reader io.Reader, namespace string) error {
	canonical, err := readCanonicalReport(reader)
	if err != nil {
		return err
	}

	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		if _, err := fmt.Fprintf(writer, "OpenMeshGuard score: %s/100\n", formatScore(canonical.Scores.Overall)); err != nil {
			return fmt.Errorf("write score: %w", err)
		}
	} else {
		found := false
		for _, item := range canonical.Scores.Namespaces {
			if item.Namespace != namespace {
				continue
			}
			found = true
			capNotice := ""
			if item.Capped {
				capNotice = " (critical cap applied)"
			}
			if _, err := fmt.Fprintf(
				writer,
				"OpenMeshGuard score for namespace %s: %s/100%s\n",
				namespace,
				formatScore(item.Score),
				capNotice,
			); err != nil {
				return fmt.Errorf("write score: %w", err)
			}
			break
		}
		if !found {
			return fmt.Errorf("score namespace %q is not present in canonical JSON", namespace)
		}
	}

	if _, err := fmt.Fprintln(writer, "Category grades:"); err != nil {
		return fmt.Errorf("write score: %w", err)
	}
	for _, category := range canonical.Scores.Categories {
		if _, err := fmt.Fprintf(
			writer,
			"  %-12s %s  pass=%s evaluated=%d unknown=%d\n",
			category.Category,
			category.Grade,
			formatPercent(category.PassRate),
			category.Evaluated,
			category.Unknown,
		); err != nil {
			return fmt.Errorf("write score: %w", err)
		}
	}
	unknowns := 0
	for _, item := range canonical.Findings {
		if item.Status == "unknown" {
			unknowns++
		}
	}
	if _, err := fmt.Fprintf(
		writer,
		"Unknown findings: %d (excluded from score and exit status by default)\n",
		unknowns,
	); err != nil {
		return fmt.Errorf("write score: %w", err)
	}
	return nil
}

// FailsThreshold evaluates CI status strictly from canonical finding fields.
func FailsThreshold(reader io.Reader, threshold string, failOnUnknown bool) (bool, error) {
	canonical, err := readCanonicalReport(reader)
	if err != nil {
		return false, err
	}
	threshold = strings.TrimSpace(strings.ToLower(threshold))
	if threshold != "" && severityRank(threshold) == 0 {
		return false, fmt.Errorf(
			"invalid --fail-on severity %q; want critical, high, medium, low, or info",
			threshold,
		)
	}
	for _, item := range canonical.Findings {
		if item.Status == "unknown" {
			if failOnUnknown {
				return true, nil
			}
			continue
		}
		if threshold == "" || item.Status != "open" {
			continue
		}
		if severityRank(item.Severity) == 0 {
			return false, fmt.Errorf("finding %s has invalid severity %q", item.ID, item.Severity)
		}
		if severityRank(item.Severity) >= severityRank(threshold) {
			return true, nil
		}
	}
	return false, nil
}
