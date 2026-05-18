package story

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
)

var errEmptyBranchName = errors.New("branch_name_template produced an empty branch name")

// branchNameData is the template data available in branch_name_template.
type branchNameData struct {
	Name string
}

// BranchFromTemplate evaluates tpl as a Go text/template with .Name set to
// storyName and returns the result. An empty tpl uses the default template
// "feat/{{.Name}}". Returns an error if the template is syntactically invalid
// or evaluates to an empty string.
func BranchFromTemplate(tpl, storyName string) (string, error) {
	if tpl == "" {
		tpl = config.DefaultBranchNameTemplate
	}

	t, err := template.New("branch").Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("invalid branch_name_template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, branchNameData{Name: storyName}); err != nil {
		return "", fmt.Errorf("evaluating branch_name_template: %w", err)
	}

	result := buf.String()
	if result == "" {
		return "", errEmptyBranchName
	}

	return result, nil
}
