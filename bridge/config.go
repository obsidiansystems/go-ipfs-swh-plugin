package bridge

import (
	"fmt"
	"net/url"

	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
)

type BridgeDatastoreConfig struct {
	base_url   *url.URL
	auth_token *string
}

func (c *BridgeDatastoreConfig) DiskSpec() fsrepo.DiskSpec {
	return nil
}

func (cfg *BridgeDatastoreConfig) Create(string) (repo.Datastore, error) {
	return BridgeDs{cfg}, nil
}

func ParseConfig(params map[string]interface{}) (*BridgeDatastoreConfig, error) {
	base_url_v, ok := params["base-url"]
	base_url_s := "https://archive.softwareheritage.org"
	if ok {
		base_url_s, ok = base_url_v.(string)
		if !ok {
			return nil, fmt.Errorf("base-url %q is not a string", base_url_v)
		}
	}

	base_url, err := url.Parse(base_url_s)
	if err != nil {
		return nil, err
	}

	var auth_token *string
	auth_token_v, ok := params["auth-token"]
	if ok {
		auth_token_s, ok := auth_token_v.(string)
		if !ok {
			return nil, fmt.Errorf("auth-token %q is not a string", auth_token_v)
		}
		auth_token = &auth_token_s
	}

	return &BridgeDatastoreConfig{
		base_url,
		auth_token,
	}, nil
}
