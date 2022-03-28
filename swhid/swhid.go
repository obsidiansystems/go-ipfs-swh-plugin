package swhid

import (
	"fmt"
	"io"
	"io/ioutil"

	plugin "github.com/ipfs/go-ipfs/plugin"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/multicodec"
)

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
	fmt.Printf("SWHID plugin loaded!\n")
	reg.RegisterEncoder(0x01f0, func(a datamodel.Node, b io.Writer) error {
		return fmt.Errorf("test error (encode)")
	})
	reg.RegisterDecoder(0x01f0, func(a datamodel.NodeAssembler, b io.Reader) error {
		bytes, err := ioutil.ReadAll(b)
		if err != nil {
			return err
		}
		fmt.Printf("returned data: %s\n", string(bytes))
		return fmt.Errorf("SWH decode is NYI")
	})
	return nil
}
