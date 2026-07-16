package validation

import (
	"fmt"
	"strings"

	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
)

// D2Validator validates D2 diagram output against expected topology
type D2Validator struct {
	expected *model.Cluster
	actual   string
}

// NewD2Validator creates a new validator with expected topology and actual D2 output
func NewD2Validator(expected *model.Cluster, d2Output string) *D2Validator {
	return &D2Validator{
		expected: expected,
		actual:   d2Output,
	}
}

// ValidateSyntax checks that the D2 output has valid syntax.
func (v *D2Validator) ValidateSyntax() error {
	openBraces := strings.Count(v.actual, "{")
	closeBraces := strings.Count(v.actual, "}")
	if openBraces != closeBraces {
		return fmt.Errorf("unbalanced braces: %d open, %d close", openBraces, closeBraces)
	}

	if !strings.Contains(v.actual, "direction:") {
		return fmt.Errorf("missing direction header")
	}

	return nil
}

// ValidateExactRender checks that the actual D2 output matches the current renderer output exactly.
func (v *D2Validator) ValidateExactRender() error {
	var expected strings.Builder
	renderer := render.NewD2Renderer(&expected)
	if err := renderer.Render(v.expected); err != nil {
		return fmt.Errorf("render expected topology: %w", err)
	}

	expectedOutput := expected.String()
	if expectedOutput != v.actual {
		return exactRenderMismatch(expectedOutput, v.actual)
	}

	return nil
}

func exactRenderMismatch(expected, actual string) error {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")
	lineCount := len(expectedLines)
	if len(actualLines) > lineCount {
		lineCount = len(actualLines)
	}

	for i := 0; i < lineCount; i++ {
		expectedLine := "<missing>"
		actualLine := "<missing>"
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		}
		if expectedLine != actualLine {
			return fmt.Errorf(
				"actual D2 output does not match exact renderer output: first difference at line %d (expected %q, actual %q)",
				i+1,
				expectedLine,
				actualLine,
			)
		}
	}

	return fmt.Errorf(
		"actual D2 output does not match exact renderer output: expected %d bytes, actual %d bytes",
		len(expected),
		len(actual),
	)
}
