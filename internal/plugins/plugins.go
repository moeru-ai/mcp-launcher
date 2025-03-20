package plugins

import (
	browserbase "github.com/moeru-ai/mcp-launcher/internal/plugins/mcp/sources/github.com/browserbase/plugins"
	fatwang2 "github.com/moeru-ai/mcp-launcher/internal/plugins/mcp/sources/github.com/fatwang2/plugins"
	"github.com/moeru-ai/mcp-launcher/pkg/pluginregistry"
)

func RegisterPlugins() {
	pluginregistry.Register(browserbase.NewStageHeadMCPServerPlugin())
	pluginregistry.Register(fatwang2.NewSearchAPIMCPServerPlugin())
}
