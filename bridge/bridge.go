package bridge

import (
	config "github.com/ipfs/go-ipfs-config"
	plugin "github.com/ipfs/go-ipfs/plugin"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
)

type BridgePlugin struct{}

var _ plugin.PluginDatastore = (*BridgePlugin)(nil)

func (*BridgePlugin) Name() string {
	return "swhbridge"
}

func (*BridgePlugin) Version() string {
	return "0.1.0"
}

func bridgeSpec() map[string]interface{} {
	return map[string]interface{}{
		"type": "mount",
		"mounts": []interface{}{
			map[string]interface{}{
				"mountpoint": "/blocks",
				"type":       "measure",
				"prefix":     "swhbridge.datastore",
				"child": map[string]interface{}{
					"type": "swhbridge",
				},
			},
			map[string]interface{}{
				"mountpoint": "/",
				"type":       "measure",
				"prefix":     "leveldb.datastore",
				"child": map[string]interface{}{
					"type":        "levelds",
					"path":        "datastore",
					"compression": "none",
				},
			},
		},
	}
}

func (*BridgePlugin) Init(env *plugin.Environment) error {
	config.Profiles["swhbridge"] = config.Profile{
		Description: "Configures the node to act as a bridge to the Software Heritage archive.",
		InitOnly:    true,
		Transform: func(c *config.Config) error {
			c.Datastore.Spec = bridgeSpec()
			return nil
		},
	}
	return nil
}

func (*BridgePlugin) DatastoreTypeName() string {
	return "swhbridge"
}

func (*BridgePlugin) DatastoreConfigParser() fsrepo.ConfigFromMap {
	return func(params map[string]interface{}) (fsrepo.DatastoreConfig, error) {
		return ParseConfig(params)
	}
}
