package bridge

import (
	"fmt"
	"net/url"

	"github.com/ipfs/kubo/repo"
	"github.com/ipfs/kubo/repo/fsrepo"
)

type SwhClientConfig struct {
	base_url   *url.URL
	auth_token *string
}

func (c *SwhClientConfig) DiskSpec() fsrepo.DiskSpec {
	return nil
}

func (cfg *SwhClientConfig) Create(string) (repo.Datastore, error) {
	return BridgeDs{c: SwhClient{cfg}}, nil
}

func ParseConfig(params map[string]interface{}) (*SwhClientConfig, error) {
	base_url_v, ok := params["base-url"]
	base_url_s := "https://archive.softwareheritage.org"
	if ok {
		// Overwrite default with explicit config field if it exists.
		base_url_s, ok = base_url_v.(string)
		if !ok {
			return nil, fmt.Errorf("base-url %q is not a string", base_url_v)
		}
	}

	base_url, err := url.Parse(base_url_s)
	if err != nil {
		return nil, err
	}

	// Default is null pointer, i.e. no auth token. This does not work
	// too well.
	var auth_token *string = nil
	auth_token_v, ok := params["auth-token"]
	if ok {
		// Overwrite default with explicit config field if it exists.
		auth_token_s, ok := auth_token_v.(string)
		if !ok {
			return nil, fmt.Errorf("auth-token %q is not a string", auth_token_v)
		}
		auth_token = &auth_token_s
	}

	return &SwhClientConfig{
		base_url,
		auth_token,
	}, nil
}
