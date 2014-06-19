// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// These are state API client methods that use their own raw connections
// to the state machine rather than the state's RPC client.

package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"code.google.com/p/go.net/websocket"
	"github.com/juju/charm"
	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/utils"

	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/state/api/rawrpc"
	"github.com/juju/juju/tools"
	"github.com/juju/juju/version"
)

// XXX Replace with Client.callWithData() that uses the state's RPC client.
func (c *Client) sendRawRPC(
	method string, args rawrpc.Args, data *rawrpc.RequestData, result rawrpc.RemoteResult,
) rawrpc.Error {
	// Prepare the HTTP request.
	req, err := rawrpc.GetRequest(c.st.serverRoot, method, args, data)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.st.tag, c.st.password)

	// Send the HTTP request.
	client := c.rawHTTPClient()
	resp, err := rawrpc.SendRequest(client, req)
	if err != nil {
		return err
	}

	// Handle the HTTP response.
	err = rawrpc.HandleResponse(resp, result)
	if err != nil {
		return err
	}

	return nil
}

//----------------------
// charms

type CharmsResult params.CharmsResponse

func (r *CharmsResult) RemoteError() error {
	return fmt.Errorf(r.Error)
}

func (c *Client) bundleCharm(
	ch charm.Charm,
) (
	archive *os.File, istemp bool, err error,
) {
	switch ch := ch.(type) {
	case *charm.Dir:
		istemp = true
		if archive, err = ioutil.TempFile("", "charm"); err != nil {
			err = fmt.Errorf("cannot create temp file: %v", err)
		} else if err = ch.BundleTo(archive); err != nil {
			err = fmt.Errorf("cannot repackage charm: %v", err)
		} else if _, err := archive.Seek(0, 0); err != nil {
			err = fmt.Errorf("cannot rewind packaged charm: %v", err)
		}
	case *charm.Bundle:
		if archive, err = os.Open(ch.Path); err != nil {
			err = fmt.Errorf("cannot read charm archive: %v", err)
		}
	default:
		err = fmt.Errorf("unknown charm type %T", ch)
	}
	return
}

// AddLocalCharm prepares the given charm with a local: schema in its
// URL, and uploads it via the API server, returning the assigned
// charm URL. If the API server does not support charm uploads, an
// error satisfying params.IsCodeNotImplemented() is returned.
func (c *Client) AddLocalCharm(curl *charm.URL, ch charm.Charm) (*charm.URL, error) {
	if curl.Schema != "local" {
		return nil, fmt.Errorf("expected charm URL with local: schema, got %q", curl.String())
	}
	// Package the charm for uploading.
	archive, istemp, err := c.bundleCharm(ch)
	if archive != nil {
		if istemp {
			defer os.Remove(archive.Name())
		}
		defer archive.Close()
	}
	if err != nil {
		return nil, err
	}

	// Send the request.
	var result CharmsResult
	args := params.CharmsArgs{Series: curl.Series}
	data := RequestData{Stream: archive, Mimetype: "application/zip"}
	err = c.sendRawRPC("charms", args, data, &result)
	if err != nil {
		return nil, err
	}
	return charm.MustParseURL(result.CharmURL), nil
}

//----------------------
// tools

type ToolsResult params.ToolsResult

func (r *ToolsResult) RemoteError() error {
	return fmt.Errorf(r.Error)
}

func (c *Client) UploadTools(
	toolsFilename string, vers version.Binary, fakeSeries ...string,
) (
	tools *tools.Tools, err error,
) {
	toolsTarball, err := os.Open(toolsFilename)
	if err != nil {
		return nil, err
	}
	defer toolsTarball.Close()

	// Send the request.
	var result ToolsResult
	series := strings.Join(fakeSeries, ",")
	args := params.ToolsArgs{BinaryVersion: vers, Series: series}
	data := RequestData{Stream: toolsTarball, Mimetype: "application/x-tar-gz"}
	err = c.sendRawRPC("tools", args, data, &result)
	if err != nil {
		return nil, err
	}
	return result.Tools, nil
}

//----------------------
// logs

// websocketDialConfig is called instead of websocket.DialConfig so we can
// override it in tests.
var websocketDialConfig = func(config *websocket.Config) (io.ReadCloser, error) {
	return websocket.DialConfig(config)
}

// DebugLogParams holds parameters for WatchDebugLog that control the
// filtering of the log messages. If the structure is zero initialized, the
// entire log file is sent back starting from the end, and until the user
// closes the connection.
type DebugLogParams struct {
	// IncludeEntity lists entity tags to include in the response. Tags may
	// finish with a '*' to match a prefix e.g.: unit-mysql-*, machine-2. If
	// none are set, then all lines are considered included.
	IncludeEntity []string
	// IncludeModule lists logging modules to include in the response. If none
	// are set all modules are considered included.  If a module is specified,
	// all the submodules also match.
	IncludeModule []string
	// ExcludeEntity lists entity tags to exclude from the response. As with
	// IncludeEntity the values may finish with a '*'.
	ExcludeEntity []string
	// ExcludeModule lists logging modules to exclude from the resposne. If a
	// module is specified, all the submodules are also excluded.
	ExcludeModule []string
	// Limit defines the maximum number of lines to return. Once this many
	// have been sent, the socket is closed.  If zero, all filtered lines are
	// sent down the connection until the client closes the connection.
	Limit uint
	// Backlog tells the server to try to go back this many lines before
	// starting filtering. If backlog is zero and replay is false, then there
	// may be an initial delay until the next matching log message is written.
	Backlog uint
	// Level specifies the minimum logging level to be sent back in the response.
	Level loggo.Level
	// Replay tells the server to start at the start of the log file rather
	// than the end. If replay is true, backlog is ignored.
	Replay bool
}

// WatchDebugLog returns a ReadCloser that the caller can read the log
// lines from. Only log lines that match the filtering specified in
// the DebugLogParams are returned. It returns an error that satisfies
// errors.IsNotImplemented when the API server does not support the
// end-point.
//
// TODO(dimitern) We already have errors.IsNotImplemented - why do we
// need to define a different error for this purpose here?
func (c *Client) WatchDebugLog(args DebugLogParams) (io.ReadCloser, error) {
	// The websocket connection just hangs if the server doesn't have the log
	// end point. So do a version check, as version was added at the same time
	// as the remote end point.
	_, err := c.AgentVersion()
	if err != nil {
		return nil, errors.NotSupportedf("WatchDebugLog")
	}
	// Prepare URL.
	attrs := url.Values{}
	if args.Replay {
		attrs.Set("replay", fmt.Sprint(args.Replay))
	}
	if args.Limit > 0 {
		attrs.Set("maxLines", fmt.Sprint(args.Limit))
	}
	if args.Backlog > 0 {
		attrs.Set("backlog", fmt.Sprint(args.Backlog))
	}
	if args.Level != loggo.UNSPECIFIED {
		attrs.Set("level", fmt.Sprint(args.Level))
	}
	attrs["includeEntity"] = args.IncludeEntity
	attrs["includeModule"] = args.IncludeModule
	attrs["excludeEntity"] = args.ExcludeEntity
	attrs["excludeModule"] = args.ExcludeModule

	target := url.URL{
		Scheme:   "wss",
		Host:     c.st.addr,
		Path:     "/log",
		RawQuery: attrs.Encode(),
	}
	cfg, err := websocket.NewConfig(target.String(), "http://localhost/")
	cfg.Header = utils.BasicAuthHeader(c.st.tag, c.st.password)
	cfg.TlsConfig = &tls.Config{RootCAs: c.st.certPool, ServerName: "anything"}
	connection, err := websocketDialConfig(cfg)
	if err != nil {
		return nil, err
	}
	// Read the initial error and translate to a real error.
	// Read up to the first new line character. We can't use bufio here as it
	// reads too much from the reader.
	line := make([]byte, 4096)
	n, err := connection.Read(line)
	if err != nil {
		return nil, fmt.Errorf("unable to read initial response: %v", err)
	}
	line = line[0:n]

	logger.Debugf("initial line: %q", line)
	var errResult params.ErrorResult
	err = json.Unmarshal(line, &errResult)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal initial response: %v", err)
	}
	if errResult.Error != nil {
		return nil, errResult.Error
	}
	return connection, nil
}
