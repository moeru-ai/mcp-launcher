package plugins

import "context"

type Plugin interface {
	isPlugin()
}

type IsPlugin struct{}

func (IsPlugin) isPlugin() {}

type PluginBeforeClone interface {
	Plugin

	ShouldHandleBeforeClone(ctx context.Context) (bool, error)
	BeforeClone(ctx context.Context) error
}

type PluginAfterClone interface {
	Plugin

	ShouldHandleAfterClone(ctx context.Context) (bool, error)
	AfterClone(ctx context.Context) error
}

type PluginBeforeBuild interface {
	Plugin

	ShouldHandleBeforeBuild(ctx context.Context) (bool, error)
	BeforeBuild(ctx context.Context) error
}

type PluginAfterBuild interface {
	Plugin

	ShouldHandleAfterBuild(ctx context.Context) (bool, error)
	AfterBuild(ctx context.Context) error
}
