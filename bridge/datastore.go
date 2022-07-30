package bridge

import (
	ctx "context"

	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	"github.com/ipfs/kubo/repo"
	"github.com/multiformats/go-base32"
	mb "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
)

type BridgeDs struct {
	c SwhClient
}

var _ repo.Datastore = (*BridgeDs)(nil)

func (c BridgeDs) Close() error {
	return nil
}

// Parse a datastore key as a SHA1 multihash, and encode it in hex
// (codec 'f'), dropping the signifier byte.
func dsKeyToGit(key ds.Key) (string, error) {
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

	// Return error of the resulting multihash is not something we can
	// relay to SWH.
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

func (b BridgeDs) Get(ctx ctx.Context, key ds.Key) (value []byte, err error) {
	// Try to parse the key as a Git hash
	hash, err := dsKeyToGit(key)
	if err != nil {
		e, ok := err.(nonGitHash)
		if ok {
			// Non-git is not an error, just something we don't have.
			swhlog.Debugf("Requested key bridge can't get: %s", e.Error())
			return nil, ds.ErrNotFound
		} else {
			// Other error actually is an errors, probably indicate
			// malformed keys.
			return nil, err
		}
	}

	var swhid string = ""
	if p, err := b.c.findSwhidFromGit([]string{"cnt", "dir", "rev", "rel", "snp"}, hash); p != nil && err == nil {
		swhid = *p
	} else {
		return nil, err
	}

	return b.c.fetchSwhid(swhid)
}

func (b BridgeDs) Has(ctx ctx.Context, key ds.Key) (exists bool, err error) {
	// Try to parse the key as a Git hash
	hash, err := dsKeyToGit(key)
	if err != nil {
		return false, nil
		e, ok := err.(nonGitHash)
		if ok {
			// Non-git is not an error, just something we don't have.
			swhlog.Debugf("Requested key bridge doesn't have: %s", e.Error())
			return false, nil
		} else {
			// Other error actually is an errors, probably indicate
			// malformed keys.
			return false, err
		}
	}

	if p, err := b.c.findSwhidFromGit([]string{"cnt", "dir", "rev", "rel", "snp"}, hash); err == nil {
		// We don't care what a found SWHID is, just whether one was in
		// fact found (null pointer vs string).
		return p != nil, nil
	} else {
		return false, err
	}
}

func (b BridgeDs) GetSize(ctx ctx.Context, key ds.Key) (size int, err error) {
	// TODO Don't actually fetch the data.
	return ds.GetBackedSize(ctx, b, key)
}

func (b BridgeDs) Query(ctx ctx.Context, q dsq.Query) (dsq.Results, error) {
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
