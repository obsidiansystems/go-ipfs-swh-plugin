package bridge

import (
	ctx "context"
	"fmt"

	blocks "github.com/ipfs/go-block-format"
	bs "github.com/ipfs/go-ipfs-blockstore"
	ds "github.com/ipfs/go-datastore"
	cid "github.com/ipfs/go-cid"
	mb "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
	mc "github.com/multiformats/go-multicodec"
	ipld "github.com/ipfs/go-ipld-format"
	uatomic "go.uber.org/atomic"
)

type BridgeBs struct {
	c SwhClient
	rehash *uatomic.Bool
}

var _ bs.Blockstore = (*BridgeBs)(nil)

func (c BridgeBs) Close() error {
	return nil
}

type nonSwhCodec struct {
	Code mc.Code
}

func (e nonSwhCodec) Error() string {
	return fmt.Sprintf("Data has codec %d, but expected %d (git-raw) or %d (swhid-1-snp)", e.Code, (uint64)(mc.GitRaw), (uint64)(mc.Swhid1Snp))
}

// Parse a blockstore key as a rought SWHID type and SHA1 multihash,
// and encoding the latter in hex (codec 'f'), dropping the string signifier
// byte.
func bsKeyToGit(key cid.Cid) (string, *mc.Code, error) {
	switch c := mc.Code(key.Type()); c {
		case mc.GitRaw:
		case mc.Swhid1Snp:
			break;
		default:
			return "", nil, nonSwhCodec{Code: c}

	}

	// Decode the multihash
	myh, err := mh.Decode(key.Hash())
	if err != nil {
	        return "", nil, err
	}

	// Return error of the resulting multihash is not something we can
	// relay to SWH.
	if myh.Code != mh.SHA1 {
		return "", nil, nonGitHash{Code: myh.Code}
	}

	// Re-encode it in hex
	str, err := mb.Encode('f', myh.Digest)
	if err != nil {
		return "", nil, err
	}

	var c = mc.Code(key.Type());
	return str[1:], &c, nil
}

func (b BridgeBs) GetRaw(ctx ctx.Context, key cid.Cid) (value []byte, err error) {
	// Try to parse the key as a rough swhid type + hash
	hash, swhid_type, err := bsKeyToGit(key)
	if err != nil {
		_, ok0 := err.(nonSwhCodec)
		_, ok1 := err.(nonGitHash)
		if ok0 || ok1 {
			// Non-git is not an error, just something we don't have.
			swhlog.Debugf("Requested key bridge can't get: %s", err.Error())
			return nil, ipld.ErrNotFound{Cid: key}
		} else {
			// Other error actually is an errors, probably indicate
			// malformed keys.
			return nil, err
		}
	}

	var swhid string = ""
	switch *swhid_type {
	case mc.GitRaw:
		if p, err := b.c.findSwhidFromGit([]string{"cnt", "dir", "rev", "rel"}, hash); p != nil && err == nil {
			swhid = *p
		} else {
			return nil, err
		}
	case mc.Swhid1Snp:
		swhid = fmt.Sprintf("swh:1:snp:%s", hash)
    }

	return b.c.fetchSwhid(swhid)
}

func (b BridgeBs) Get(ctx ctx.Context, key cid.Cid) (blocks.Block, error) {
	bdata, err := b.GetRaw(ctx, key)

	// TODO Copied from go-ipfs-blockstore, should de-dup
	if err == ds.ErrNotFound {
		return nil, ipld.ErrNotFound{Cid: key}
	}
	if err != nil {
		return nil, err
	}

	if b.rehash.Load() {
		rbcid, err := key.Prefix().Sum(bdata)
		if err != nil {
			return nil, err
		}

		if !rbcid.Equals(key) {
			return nil, bs.ErrHashMismatch
		}

		return blocks.NewBlockWithCid(bdata, rbcid)
	}
	return blocks.NewBlockWithCid(bdata, key)
}

func (b BridgeBs) Has(ctx ctx.Context, key cid.Cid) (exists bool, err error) {
	// Try to parse the key as a Git hash
	hash, swhid_type, err := bsKeyToGit(key)
	if err != nil {
		return false, nil
		_, ok0 := err.(nonSwhCodec)
		_, ok1:= err.(nonGitHash)
		if ok0 || ok1 {
			// Non-git is not an error, just something we don't have.
			swhlog.Debugf("Requested key bridge doesn't have: %s", err.Error())
			return false, nil
		} else {
			// Other error actually is an errors, probably indicate
			// malformed keys.
			return false, err
		}
	}

	var types []string
	switch *swhid_type {
	case mc.GitRaw:
		types = []string{"cnt", "dir", "rev", "rel"}
	case mc.Swhid1Snp:
		types = []string{"snp"}
	}

	if p, err := b.c.findSwhidFromGit(types, hash); err == nil {
		// We don't care what a found SWHID is, just whether one was in
		// fact found (null pointer vs string).
		return p != nil, nil
	} else {
		return false, err
	}
}

func (b BridgeBs) GetSize(ctx ctx.Context, key cid.Cid) (size int, err error) {
	// TODO Don't actually fetch the data.
	value, err := b.GetRaw(ctx, key)
	if err == nil {
		return len(value), nil
	}
	return -1, err
}

func (b BridgeBs) Put(ctx.Context, blocks.Block) error {
	return nil
}

func (b BridgeBs) PutMany(ctx.Context, []blocks.Block) error {
	return nil
}

func (b BridgeBs) DeleteBlock(ctx.Context, cid.Cid) error {
	return nil
}

func (b BridgeBs) AllKeysChan(ctx ctx.Context) (<-chan cid.Cid, error) {
	swhlog.Infof("AllKeysChan")
	return nil, nil
}

func (b BridgeBs) HashOnRead(enabled bool) {
	b.rehash.Store(enabled)
}
