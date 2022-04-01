package bridge

import (
	ctx "context"
	"fmt"
	"io/ioutil"
	"net/http"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	config "github.com/ipfs/go-ipfs-config"
	plugin "github.com/ipfs/go-ipfs/plugin"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
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
	config map[string]interface{} // Go moment
}

// TODO: Investigate possibility of caching fetched objects locally à la
// flatfs?
func (c *bridgeDatastoreConfig) DiskSpec() fsrepo.DiskSpec {
	return nil
}

type BridgeDs struct {
}

var _ repo.Datastore = (*BridgeDs)(nil)

func (c BridgeDs) Close() error {
	return nil
}

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

func (b BridgeDs) fetchHash(hash string, key ds.Key) ([]byte, error) {
	// TODO: Hit the "/api/1/known" endpoint with a POST request with the
	// set of possible SWHIDs for the given hash, then fetch only the
	// SWHID in the object which was indicated to exist.

	/* Attempt 1: Fetch the given hash as a blob. We hit the "content" SWH
	 * API endpoint, and use that as the contents. */
	fmt.Printf("swh bridge: fetching hash: %s\n", hash)
	url := fmt.Sprintf("https://archive.softwareheritage.org/api/1/content/sha1_git:%s/raw/", hash)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, ds.ErrNotFound
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("swh bridge: hash fetched: %s\n", hash)

	/* We need to add the git blob header (so the string "blob", the byte
	 * length as decimal digits, and a zero byte) to the returned contents,
	 * otherwise other IPFS nodes have no reason to believe that the data
	 * we just gave them is what we said we gave them. */
	head := fmt.Sprintf("blob %d\x00", len(buf))
	buffer := make([]byte, len(head))
	copy(buffer, head)
	buffer = append(buffer, buf...)

	return buffer, nil
}

func (b BridgeDs) Get(ctx ctx.Context, key ds.Key) (value []byte, err error) {
	// Try to parse the key as a Git hash
	hash, err := keyToGit(key)

	// TODO: Work out how to differentiate between different object types
	// (content, snapshot, tree, commit) without hitting the API 4 times
	// per Get. I don't think we have access to the codec in this method..
	if err == nil {
		return b.fetchHash(hash, key)
	}

	return nil, nil
}

func (b BridgeDs) Has(ctx ctx.Context, key ds.Key) (exists bool, err error) {
	return false, nil
}

func (b BridgeDs) GetSize(ctx ctx.Context, key ds.Key) (size int, err error) {
	return 0, ds.ErrNotFound
}

func (b BridgeDs) Query(ctx ctx.Context, q query.Query) (query.Results, error) {
	fmt.Printf("query: %s\n", q)
	return nil, nil
}

func (b BridgeDs) Put(ctx ctx.Context, key ds.Key, value []byte) error {
	return nil
}

func (b BridgeDs) Delete(ctx ctx.Context, key ds.Key) error {
	return nil
}

func (b BridgeDs) Sync(ctx ctx.Context, prefix ds.Key) error {
	return nil
}

func (b BridgeDs) Batch(ctx ctx.Context) (ds.Batch, error) {
	return ds.NewBasicBatch(b), nil
}

func (c *bridgeDatastoreConfig) Create(string) (repo.Datastore, error) {
	return BridgeDs{}, nil
}

func (*BridgePlugin) DatastoreConfigParser() fsrepo.ConfigFromMap {
	return func(cfg map[string]interface{}) (fsrepo.DatastoreConfig, error) { // Go moment
		return &bridgeDatastoreConfig{cfg}, nil
	}
}
