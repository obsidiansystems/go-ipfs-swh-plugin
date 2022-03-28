package bridge

import (
	ctx "context"
	"fmt"
	"io/ioutil"
	"net/http"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
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

func (*BridgePlugin) Init(env *plugin.Environment) error {
	return nil
}

func (*BridgePlugin) DatastoreTypeName() string {
	return "swhds"
}

type bridgeDatastoreConfig struct {
	config map[string]interface{} // Go moment
}

// TODO: Investigate possibility of caching fetched objects locally à la
// flatfs?
func (c *bridgeDatastoreConfig) DiskSpec() fsrepo.DiskSpec {
	return nil
}

// We need a map to store puts/has-es; IPFS will always try to store the
// "empty directory" CID.
type BridgeDs struct {
	backing map[ds.Key][]byte
}

var _ repo.Datastore = (*BridgeDs)(nil)

func (c BridgeDs) Close() error {
	return nil
}

// Parse a datastore key as a SHA1 multihash, and encode it in hex
// (codec 'f'), dropping the signifier byte.
func keyToGit(key ds.Key) (string, error) {
	data, err := base32.RawStdEncoding.DecodeString(key.String()[1:])
	if err != nil {
		return "", err
	}
	myh, err := mh.Decode(data)
	if err != nil {
		return "", err
	}

	if myh.Code != mh.SHA1 {
		return "", fmt.Errorf("data was a multihash encoded with %d, but expected %d (SHA1)", myh.Code, mh.SHA1)
	}

	str, err := mb.Encode('f', myh.Digest)
	if err != nil {
		return "", err
	}
	return str[1:], nil
}

func (b BridgeDs) Get(ctx ctx.Context, key ds.Key) (value []byte, err error) {
	fmt.Printf("get: key=%s\n", key)

	val, found := b.backing[key]
	if found {
		return val, nil
	}

	// Try to parse the key as a Git hash
	hash, err := keyToGit(key)

	// TODO: Work out how to differentiate between different object types
	// (content, snapshot, tree, commit) without hitting the API 4 times
	// per Get. I don't think we have access to the codec in this method..
	if err == nil {
		fmt.Printf("getting hash: %s\n", hash)
		url := fmt.Sprintf("https://archive.softwareheritage.org/api/1/content/sha1_git:%s/raw/", hash)

		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		b.backing[key] = buf // use the put storage for cheap caching
		return buf, nil
	}

	return nil, fmt.Errorf("swhds: dunno how to get %s", key)
}

func (b BridgeDs) Has(ctx ctx.Context, key ds.Key) (exists bool, err error) {
	fmt.Printf("has: key=%s\n", key)
	_, found := b.backing[key]
	return found, nil
}

func (b BridgeDs) GetSize(ctx ctx.Context, key ds.Key) (size int, err error) {
	return 0, nil
}

func (b BridgeDs) Query(ctx ctx.Context, q query.Query) (query.Results, error) {
	fmt.Printf("query: %s\n", q)
	return nil, nil
}

func (b BridgeDs) Put(ctx ctx.Context, key ds.Key, value []byte) error {
	fmt.Printf("put: key=%s, value=%s\n", key, value)
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
	return BridgeDs{make(map[ds.Key][]byte)}, nil
}

func (*BridgePlugin) DatastoreConfigParser() fsrepo.ConfigFromMap {
	return func(cfg map[string]interface{}) (fsrepo.DatastoreConfig, error) { // Go moment
		return &bridgeDatastoreConfig{cfg}, nil
	}
}
