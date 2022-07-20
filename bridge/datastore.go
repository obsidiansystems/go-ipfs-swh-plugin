package bridge

import (
	"bytes"
	ctx "context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/ipfs/go-ipfs/repo"
	logging "github.com/ipfs/go-log"
	"github.com/multiformats/go-base32"
	mb "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
)

type BridgeDs struct {
	cfg *BridgeDatastoreConfig
}

var _ repo.Datastore = (*BridgeDs)(nil)

func (c BridgeDs) Close() error {
	return nil
}

// swhlog is the logger for the SWH Bridge
var swhlog = logging.Logger("swh-bridge")

type nonGitHash struct {
	Code uint64
}

func (e nonGitHash) Error() string {
	return fmt.Sprintf("Data was a multihash encoded with %d, but expected %d (SHA1)", e.Code, mh.SHA1)
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
		return "", nonGitHash{Code: myh.Code}
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
	swhlog.Infof("lookup up hash: %s\n", hash)
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
			swhlog.Infof("found SWHID %s", s)
			return &s, nil
			break
		}
	}
	swhlog.Infof("no SWHID found for %s", hash)
	return nil, ds.ErrNotFound
}

func (b BridgeDs) fetchSwhid(swhid string, key ds.Key) ([]byte, error) {
	/* Fetch the given hash as a blob. We hit the "content" SWH API
	 * endpoint, and use that as the contents. */
	swhlog.Infof("fetching SWHID: %s\n", swhid)

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

	swhlog.Infof("SWHID fetched: %s\n", swhid)

	return buf, nil
}

func (b BridgeDs) Get(ctx ctx.Context, key ds.Key) (value []byte, err error) {
	// Try to parse the key as a Git hash
	hash, err := keyToGit(key)
	if err != nil {
		e, ok := err.(nonGitHash)
		if ok {
			// Non-git is not an error, just something we don't have.
			swhlog.Debugf("Requested key bridge can't get: %s", e.Error())
			return nil, ds.ErrNotFound
		} else {
			return nil, err
		}
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
		// Non-git is not an error, just something we don't have.
		return false, nil
		e, ok := err.(nonGitHash)
		if ok {
			// Non-git is not an error, just something we don't have.
			swhlog.Debugf("Requested key bridge doesn't have: %s", e.Error())
			return false, nil
		} else {
			return false, err
		}
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
	swhlog.Infof("query: %s\n", q)
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
