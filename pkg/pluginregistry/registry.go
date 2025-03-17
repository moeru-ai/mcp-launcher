package pluginregistry

import (
	"context"
	"log/slog"
	"sync"

	"github.com/moeru-ai/mcp-launcher/pkg/plugins"
)

var (
	DefaultRegistry *PluginRegistry
)

func init() {
	DefaultRegistry = NewPluginRegistry()
}

func Register(plugin plugins.Plugin) {
	DefaultRegistry.Register(plugin)
}

func RegisterBeforeClone(plugin plugins.PluginBeforeClone) {
	DefaultRegistry.RegisterBeforeClone(plugin)
}

func BeforeClone(ctx context.Context) error {
	return DefaultRegistry.BeforeClone(ctx)
}

func RegisterAfterClone(plugin plugins.PluginAfterClone) {
	DefaultRegistry.RegisterAfterClone(plugin)
}

func AfterClone(ctx context.Context) error {
	return DefaultRegistry.AfterClone(ctx)
}

func RegisterBeforeBuild(plugin plugins.PluginBeforeBuild) {
	DefaultRegistry.RegisterBeforeBuild(plugin)
}

func BeforeBuild(ctx context.Context) error {
	return DefaultRegistry.BeforeBuild(ctx)
}

func RegisterAfterBuild(plugin plugins.PluginAfterBuild) {
	DefaultRegistry.RegisterAfterBuild(plugin)
}

func AfterBuild(ctx context.Context) error {
	return DefaultRegistry.AfterBuild(ctx)
}

type PluginRegistry struct {
	beforeClone []plugins.PluginBeforeClone
	afterClone  []plugins.PluginAfterClone
	beforeBuild []plugins.PluginBeforeBuild
	afterBuild  []plugins.PluginAfterBuild

	mutex sync.Mutex
}

func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		beforeClone: make([]plugins.PluginBeforeClone, 0),
		afterClone:  make([]plugins.PluginAfterClone, 0),
		beforeBuild: make([]plugins.PluginBeforeBuild, 0),
		afterBuild:  make([]plugins.PluginAfterBuild, 0),
	}
}

func (r *PluginRegistry) Register(plugin plugins.Plugin) {
	if bc, ok := plugin.(plugins.PluginBeforeClone); ok {
		r.RegisterBeforeClone(bc)
	}
	if ac, ok := plugin.(plugins.PluginAfterClone); ok {
		r.RegisterAfterClone(ac)
	}
	if bb, ok := plugin.(plugins.PluginBeforeBuild); ok {
		r.RegisterBeforeBuild(bb)
	}
	if ab, ok := plugin.(plugins.PluginAfterBuild); ok {
		r.RegisterAfterBuild(ab)
	}
}

func (r *PluginRegistry) RegisterBeforeClone(plugin plugins.PluginBeforeClone) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.beforeClone = append(r.beforeClone, plugin)
}

func (r *PluginRegistry) BeforeClone(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, plugin := range r.beforeClone {
		if err := plugin.BeforeClone(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *PluginRegistry) RegisterAfterClone(plugin plugins.PluginAfterClone) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.afterClone = append(r.afterClone, plugin)
}

func (r *PluginRegistry) AfterClone(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, plugin := range r.afterClone {
		slog.Info("After clone plugin", "plugin", plugin)

		if err := plugin.AfterClone(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *PluginRegistry) RegisterBeforeBuild(plugin plugins.PluginBeforeBuild) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.beforeBuild = append(r.beforeBuild, plugin)
}

func (r *PluginRegistry) BeforeBuild(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, plugin := range r.beforeBuild {
		if err := plugin.BeforeBuild(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *PluginRegistry) RegisterAfterBuild(plugin plugins.PluginAfterBuild) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.afterBuild = append(r.afterBuild, plugin)
}

func (r *PluginRegistry) AfterBuild(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, plugin := range r.afterBuild {
		if err := plugin.AfterBuild(ctx); err != nil {
			return err
		}
	}

	return nil
}
