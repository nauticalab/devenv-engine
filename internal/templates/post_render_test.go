package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunPostRender_NoOp(t *testing.T) {
	err := RunPostRender(t.TempDir(), RenderPlan{TemplateNames: []string{"namespace"}}, NewPostRenderOptions())
	require.NoError(t, err)
}
