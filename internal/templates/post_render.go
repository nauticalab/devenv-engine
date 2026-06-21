package templates

// PostRenderOptions reserves space for future post-render behaviors.
type PostRenderOptions struct{}

func NewPostRenderOptions() PostRenderOptions {
	return PostRenderOptions{}
}

// RunPostRender executes post-render steps for a completed render pass.
func RunPostRender(outputDir string, plan RenderPlan, opts PostRenderOptions) error {
	return nil
}
