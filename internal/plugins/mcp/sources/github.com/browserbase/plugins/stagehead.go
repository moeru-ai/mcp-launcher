package plugins

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/moeru-ai/mcp-launcher/pkg/jsonpatch"
	"github.com/moeru-ai/mcp-launcher/pkg/plugins"
	"github.com/moeru-ai/mcp-launcher/pkg/rules/repositoryurlrules"
	"github.com/samber/mo"
)

var (
	_ plugins.PluginAfterClone = (*StageHeadMCPServerPlugin)(nil)
)

// StageHeadMCPServerPlugin help to build browserbase/stagehead mcp server
type StageHeadMCPServerPlugin struct {
	repositoryurlrules.RulesPlugin
}

func NewStageHeadMCPServerPlugin() *StageHeadMCPServerPlugin {
	return &StageHeadMCPServerPlugin{
		RulesPlugin: repositoryurlrules.Rules(
			repositoryurlrules.ForExact("https://github.com/browserbase/mcp-server-browserbase"),
		),
	}
}

func (p *StageHeadMCPServerPlugin) ShouldHandleAfterClone(ctx context.Context) (bool, error) {
	repoURL, ok := ctx.Value("repoURL").(string)
	if !ok {
		return false, nil
	}

	return p.ShouldHandle(ctx, repoURL), nil
}

func (p *StageHeadMCPServerPlugin) AfterClone(ctx context.Context) error {
	// Only proceed if rules match
	repoURL, ok := ctx.Value("repoURL").(string)
	if !ok {
		return nil
	}

	if !p.ShouldHandle(ctx, repoURL) {
		return nil
	}

	clonedPath, ok := ctx.Value("clonedPath").(string)
	if !ok {
		return nil
	}

	directory, ok := ctx.Value("directory").(string)
	if !ok {
		directory = "stagehead"
	}

	packageJSONPath := filepath.Join(clonedPath, directory, "package.json")

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
