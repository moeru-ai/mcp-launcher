package plugins

import (
	browserbase "github.com/moeru-ai/mcp-launcher/internal/plugins/mcp/sources/github.com/browserbase/plugins"
	"github.com/moeru-ai/mcp-launcher/pkg/pluginregistry"
)

func RegisterPlugins() {
	pluginregistry.Register(browserbase.NewStageHeadMCPServerPlugin())
}
