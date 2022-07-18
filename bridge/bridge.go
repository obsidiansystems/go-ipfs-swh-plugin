package bridge

import (
	"bytes"
	ctx "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	config "github.com/ipfs/go-ipfs-config"
	plugin "github.com/ipfs/go-ipfs/plugin"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	logging "github.com/ipfs/go-log"
	"github.com/multiformats/go-base32"
	mb "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
)

type BridgePlugin struct{}

var _ plugin.PluginDatastore = (*BridgePlugin)(nil)

func (*BridgePlugin) Name() string {
	return "swhds"
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

type bridgeDatastoreConfig struct {
	base_url   *url.URL
	auth_token *string
}

func (c *bridgeDatastoreConfig) DiskSpec() fsrepo.DiskSpec {
	return nil
}

type BridgeDs struct {
	cfg *bridgeDatastoreConfig
}

var _ repo.Datastore = (*BridgeDs)(nil)

func (c BridgeDs) Close() error {
	return nil
}

// swhlog is the logger for the SWH Bridge
var swhlog = logging.Logger("swh-bridge")

// Parse a datastore key as a SHA1 multihash, and encode it in hex
// (codec 'f'), dropping the signifier byte.
func keyToGit(key ds.Key) (string, error) {
	// Parse the key as base32-encoded data
	data, err := base32.RawStdEncoding.DecodeString(key.String()[1:])
	if err != nil {
		return "", err
	}

	// Decode the data as a multihash
	myh, err := mh.Decode(data)
	if err != nil {
		return "", err
	}

	if myh.Code != mh.SHA1 {
		return "", fmt.Errorf("data was a multihash encoded with %d, but expected %d (SHA1)", myh.Code, mh.SHA1)
	}

	// Re-encode it in hex
	str, err := mb.Encode('f', myh.Digest)
	if err != nil {
		return "", err
	}

	return str[1:], nil
}

func (b BridgeDs) customHeaderReq() http.Request {
	var req http.Request
	req.Header = map[string][]string{}
	if b.cfg.auth_token != nil {
		req.Header["Authorization"] = []string{fmt.Sprintf("Bearer %s", *b.cfg.auth_token)}
	}
	req.URL = new(url.URL)
	*req.URL = *b.cfg.base_url
	return req
}

func (b BridgeDs) findSwhidFromGit(hash string) (*string, error) {
	/* Hit the "/api/1/known" endpoint with a POST request with the set of
	 * possible SWHIDs for the given hash to find which one exists. */
	swhlog.Debugf("lookup up hash: %s\n", hash)
	req := b.customHeaderReq()
	req.Method = "POST"
	req.Header["Content-Type"] = []string{"application/json"}
	req.URL.Path = "/api/1/known/"

	req2, err := json.Marshal([]string{
		fmt.Sprintf("swh:1:cnt:%s", hash),
		fmt.Sprintf("swh:1:dir:%s", hash),
		fmt.Sprintf("swh:1:rev:%s", hash),
		fmt.Sprintf("swh:1:rel:%s", hash),
		fmt.Sprintf("swh:1:snp:%s", hash),
	})
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(req2))
	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, ds.ErrNotFound
	}
	var respParsed map[string]struct{ Known bool }
	if err := json.NewDecoder(resp.Body).Decode(&respParsed); err != nil {
		return nil, err
	}

	for s, v := range respParsed {
		if v.Known {
			swhlog.Debugf("found SWHID %s", s)
			return &s, nil
			break
		}
	}
	swhlog.Debugf("no SWHID found for %s", hash)
	return nil, ds.ErrNotFound
}

func (b BridgeDs) fetchSwhid(swhid string, key ds.Key) ([]byte, error) {
	/* Fetch the given hash as a blob. We hit the "content" SWH API
	 * endpoint, and use that as the contents. */
	swhlog.Debugf("fetching SWHID: %s\n", swhid)

	req := b.customHeaderReq()
	req.Method = "GET"
	req.Header["Content-Type"] = []string{"application/octet-stream"}
	req.URL.Path = fmt.Sprintf("/api/1/raw/%s/", swhid)

	resp1, err := http.DefaultClient.Do(&req)
	if err != nil {
		return nil, err
	}
	if resp1.StatusCode != 200 {
		return nil, ds.ErrNotFound
	}

	buf, err := ioutil.ReadAll(resp1.Body)
	if err != nil {
		return nil, err
	}

	swhlog.Debugf("SWHID fetched: %s\n", swhid)

	return buf, nil
}

func (b BridgeDs) Get(ctx ctx.Context, key ds.Key) (value []byte, err error) {
	// Try to parse the key as a Git hash
	hash, err := keyToGit(key)
	if err != nil {
		return nil, err
	}

	var swhid string = ""
	if p, err := b.findSwhidFromGit(hash); p != nil && err == nil {
		swhid = *p
	} else {
		return nil, err
	}

	return b.fetchSwhid(swhid, key)
}

func (b BridgeDs) Has(ctx ctx.Context, key ds.Key) (exists bool, err error) {
	// Try to parse the key as a Git hash
	hash, err := keyToGit(key)
	if err != nil {
		// Non-git is not and error, just something we don't have.
		return false, nil
	}

	if p, err := b.findSwhidFromGit(hash); err == nil {
		return p != nil, nil
	} else {
		return false, err
	}
}

func (b BridgeDs) GetSize(ctx ctx.Context, key ds.Key) (size int, err error) {
	// TODO Don't actually fetch the data.
	return ds.GetBackedSize(ctx, b, key)
}

func (b BridgeDs) Query(ctx ctx.Context, q query.Query) (query.Results, error) {
	swhlog.Debugf("query: %s\n", q)
	return nil, errors.New("Query is not supported")
}

func (b BridgeDs) Put(ctx ctx.Context, key ds.Key, value []byte) error {
	return errors.New("Put is not supported")
}

func (b BridgeDs) Delete(ctx ctx.Context, key ds.Key) error {
	return errors.New("Delete is not supported")
}

func (b BridgeDs) Sync(ctx ctx.Context, prefix ds.Key) error {
	return errors.New("Sync is not supported")
}

func (b BridgeDs) Batch(ctx ctx.Context) (ds.Batch, error) {
	return ds.NewBasicBatch(b), nil
}

func (cfg *bridgeDatastoreConfig) Create(string) (repo.Datastore, error) {
	return BridgeDs{cfg}, nil
}

func (*BridgePlugin) DatastoreConfigParser() fsrepo.ConfigFromMap {
	return func(params map[string]interface{}) (fsrepo.DatastoreConfig, error) {
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

		return &bridgeDatastoreConfig{
			base_url,
			auth_token,
		}, nil
	}
}
