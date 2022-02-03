package example

import (
	"github.com/ipfs/go-ipfs/plugin"

	swhid "github.com/obsidiansystems/go-ipfs-swh-plugin/swhid"
)

// Plugins is an exported list of plugins that will be loaded by go-ipfs.
var Plugins = []plugin.Plugin{
	&swhid.SwhidPlugin{},
}
