package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	ds "github.com/ipfs/go-datastore"
	mh "github.com/multiformats/go-multihash"
)

type SwhClient struct {
	cfg *SwhClientConfig
}

type nonGitHash struct {
	Code uint64
}

func (e nonGitHash) Error() string {
	return fmt.Sprintf("Data was a multihash encoded with %d, but expected %d (SHA1)", e.Code, mh.SHA1)
}

func (b SwhClient) customHeaderReq() http.Request {
	var req http.Request
	req.Header = map[string][]string{}
	if b.cfg.auth_token != nil {
		req.Header["Authorization"] = []string{fmt.Sprintf("Bearer %s", *b.cfg.auth_token)}
	}
	req.URL = new(url.URL)
	*req.URL = *b.cfg.base_url
	return req
}

func (b SwhClient) findSwhidFromGit(types []string, hash string) (*string, error) {
	// Hit the "/api/1/known" endpoint with a POST request with the set
	// of possible SWHIDs for the given hash to find which one exists.
	swhlog.Infof("looking up hash: %s", hash)

	// Build request headers
	req := b.customHeaderReq()
	req.Method = "POST"
	req.Header["Content-Type"] = []string{"application/json"}
	req.URL.Path = "/api/1/known/"

	// Build JSON request body, with each possible SWHID.
	var reqJson []string
	for _, t := range types {
		reqJson = append(reqJson, fmt.Sprintf("swh:1:%s:%s", t, hash))
	}
	req2, err := json.Marshal(reqJson)
	if err != nil {
		return nil, err
	}

	// Issue the request
	req.Body = io.NopCloser(bytes.NewReader(req2))
	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, ds.ErrNotFound
	}

	// Parse the response
	var respParsed map[string]struct{ Known bool }
	if err := json.NewDecoder(resp.Body).Decode(&respParsed); err != nil {
		return nil, err
	}

	// Take the first one that is "known", i.e. that exists.
	for s, v := range respParsed {
		if v.Known {
			swhlog.Infof("found SWHID %s", s)
			return &s, nil
			break
		}
	}

	// Return error if none were known.
	swhlog.Infof("no SWHID found for %s", hash)
	return nil, ds.ErrNotFound
}

func (b SwhClient) fetchSwhid(swhid string) ([]byte, error) {
	// Fetch the given hash as a blob. We hit the "content" SWH API
	// endpoint, and use that as the contents.
	swhlog.Infof("fetching SWHID: %s", swhid)

	// Build request headers
	req := b.customHeaderReq()
	req.Method = "GET"
	req.Header["Content-Type"] = []string{"application/octet-stream"}
	req.URL.Path = fmt.Sprintf("/api/1/raw/%s/", swhid)

	// Issue the request
	resp1, err := http.DefaultClient.Do(&req)
	if err != nil {
		return nil, err
	}
	if resp1.StatusCode != 200 {
		return nil, ds.ErrNotFound
	}

	// Read response into array
	buf, err := ioutil.ReadAll(resp1.Body)
	if err != nil {
		return nil, err
	}

	swhlog.Infof("SWHID fetched: %s", swhid)

	return buf, nil
}
