package plugins

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/moeru-ai/mcp-launcher/internal/metadata"
	"github.com/moeru-ai/mcp-launcher/pkg/jsonpatch"
	"github.com/moeru-ai/mcp-launcher/pkg/plugins"
	"github.com/moeru-ai/mcp-launcher/pkg/rules/repositoryurlrules"
	"github.com/samber/mo"
)

var (
	_ plugins.PluginAfterClone = (*StageHandMCPServerPlugin)(nil)
)

// StageHandMCPServerPlugin help to build browserbase/stagehead mcp server
type StageHandMCPServerPlugin struct {
	repositoryurlrules.RulesPlugin
}

func NewStageHeadMCPServerPlugin() *StageHandMCPServerPlugin {
	return &StageHandMCPServerPlugin{
		RulesPlugin: repositoryurlrules.Rules(
			repositoryurlrules.ForExact("https://github.com/browserbase/mcp-server-browserbase"),
		),
	}
}

func (p *StageHandMCPServerPlugin) ShouldHandleAfterClone(ctx context.Context) (bool, error) {
	return p.ShouldHandle(ctx, metadata.FromContext(ctx).RepositoryURL), nil
}

func (p *StageHandMCPServerPlugin) AfterClone(ctx context.Context) error {
	md := metadata.FromContext(ctx)

	if !p.ShouldHandle(ctx, md.RepositoryURL) {
		return nil
	}

	packageJSONPath := filepath.Join(md.RepositoryClonedPath, md.SubDirectory, "package.json")

	packageJSONContent, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return err
	}

	patchedPackageJSONContent := jsonpatch.ApplyPatches(
		packageJSONContent,
		mo.Some(jsonpatch.ApplyOptions{AllowMissingPathOnRemove: true}),
		jsonpatch.NewRemove("/scripts/prepare"),
		jsonpatch.NewAdd("/scripts/start", "node dist/index.js"),
	)
	if patchedPackageJSONContent.IsError() {
		return patchedPackageJSONContent.Error()
	}

	slog.Info("Patching package.json", "packageJSONPath", packageJSONPath)

	err = os.WriteFile(packageJSONPath, patchedPackageJSONContent.MustGet(), 0600) //nolint:mnd
	if err != nil {
		return err
	}

	return nil
}
