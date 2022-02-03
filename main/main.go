package main

import (
	swhid "github.com/obsidiansystems/go-ipfs-swh-plugin"
)

// Plugins is an exported list of plugins that will be loaded by go-ipfs.
var Plugins = swhid.Plugins
