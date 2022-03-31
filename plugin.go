package example

import (
	"github.com/ipfs/go-ipfs/plugin"

	bridge "github.com/obsidiansystems/go-ipfs-swh-plugin/bridge"
)

// Plugins is an exported list of plugins that will be loaded by go-ipfs.
var Plugins = []plugin.Plugin{
	&bridge.BridgePlugin{},
}
