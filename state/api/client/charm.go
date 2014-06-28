// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package client

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/juju/charm"

	ec "github.com/juju/juju/state/api/client.errorcodes"
	"github.com/juju/juju/state/api/params"
)

// CharmInfo holds information about a charm.
type CharmInfo struct {
	Revision int
	URL      string
	Config   *charm.Config
	Meta     *charm.Meta
}

// CharmInfo returns information about the requested charm.
func (c *Client) CharmInfo(charmURL string) (*CharmInfo, error) {
	args := params.CharmInfo{CharmURL: charmURL}
	info := new(CharmInfo)
	if err := c.call("CharmInfo", args, info); err != nil {
		return nil, err
	}
	return info, nil
}

type tempFile io.File

func (tf *tempFile) Close() error {
	err := os.Remove(tf.file.Name())
	if err != nil {
		return err
	}
	return tf.file.Close()
}

func (c *Client) bundleCharm(ch charm.Charm) (archive io.ReadCloser, err error) {
	switch ch := ch.(type) {
	case *charm.Dir:
		// XXX Use api.Error for these errors?
		if archive, err = ioutil.TempFile("", "charm"); err != nil {
			err = fmt.Errorf("cannot create temp file: %v", err)
		} else {
			if err := ch.BundleTo(archive); err != nil {
				err = fmt.Errorf("cannot repackage charm: %v", err)
			} else if _, err := archive.Seek(0, 0); err != nil {
				err = fmt.Errorf("cannot rewind packaged charm: %v", err)
			}
			archive = tempfile(archive)
			if err != nil {
				archive.Close() // Ignore any error here.
			}
		}
	case *charm.Bundle:
		if archive, err = os.Open(ch.Path); err != nil {
			err = fmt.Errorf("cannot read charm archive: %v", err)
		}
	default:
		err = fmt.Errorf("unknown charm type %T", ch)
	}
	//    return
}

// AddLocalCharm prepares the given charm with a local: schema in its
// URL, and uploads it via the API server, returning the assigned
// charm URL. If the API server does not support charm uploads, an
// error satisfying params.IsCodeNotImplemented() is returned.
func (c *Client) AddLocalCharm(curl *charm.URL, ch charm.Charm) (*charm.URL, error) {
	if curl.Schema != "local" {
		msg := fmt.Sprintf("expected charm URL with local: schema, got %q", curl.String())
		return nil, ec.PreprocessingError.Err(nil, msg)
	}

	// Package the charm for uploading.
	archive, err := c.bundleCharm(ch)
	if err != nil {
		return nil, ec.PreprocessingError.Err(err, "")
	}
	defer archive.Close()

	// Send the request.
	var result params.CharmsResponse
	args := url.Values{}
	args.Set("series", curl.Series)
	data := rawRPCData{File: archive, Mimetype: "application/zip"}
	err := c.callRaw("charm", &args, &data, &result)
	if err != nil {
		return nil, err
	}

	return charm.MustParseURL(result.CharmURL), nil
}

// AddCharm adds the given charm URL (which must include revision) to
// the environment, if it does not exist yet. Local charms are not
// supported, only charm store URLs. See also AddLocalCharm() in the
// client-side API.
func (c *Client) AddCharm(curl *charm.URL) error {
	args := params.CharmURL{URL: curl.String()}
	return c.call("AddCharm", args, nil)
}

// ResolveCharm resolves the best available charm URLs with series, for charm
// locations without a series specified.
func (c *Client) ResolveCharm(ref charm.Reference) (*charm.URL, error) {
	args := params.ResolveCharms{References: []charm.Reference{ref}}
	result := new(params.ResolveCharmResults)
	if err := c.st.Call("Client", "", "ResolveCharms", args, result); err != nil {
		return nil, err
	}
	if len(result.URLs) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}
	urlInfo := result.URLs[0]
	if urlInfo.Error != "" {
		return nil, fmt.Errorf("%v", urlInfo.Error)
	}
	return urlInfo.URL, nil
}
