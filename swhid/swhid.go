package swhid

import (
	plugin "github.com/ipfs/go-ipfs/plugin"
	logging "github.com/ipfs/go-log"
	"github.com/ipld/go-ipld-prime/multicodec"
	"github.com/obsidiansystems/go-ipld-swh"
)

// swhidlog is the logger for the non-git SWH object support.
var swhidlog = logging.Logger("swhid")

type SwhidPlugin struct{}

var _ plugin.PluginIPLD = (*SwhidPlugin)(nil)

// Name returns the plugin's name, satisfying the plugin.Plugin interface.
func (*SwhidPlugin) Name() string {
	return "swhid"
}

// Version returns the plugin's version, satisfying the plugin.Plugin interface.
func (*SwhidPlugin) Version() string {
	return "0.1.0"
}

// Init initializes plugin, satisfying the plugin.Plugin interface. Put any
// initialization logic here.
func (*SwhidPlugin) Init(env *plugin.Environment) error {
	return nil
}

func (*SwhidPlugin) Register(reg multicodec.Registry) error {
	swhidlog.Debugf("SWHID plugin loaded!\n")
	reg.RegisterEncoder(ipldswh.Swh1Snp, ipldswh.EncodeGeneric)
	reg.RegisterDecoder(ipldswh.Swh1Snp, ipldswh.DecodeGeneric)
	return nil
}
