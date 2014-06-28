// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package client

import (
	"net/url"
	"os"
	"strings"

	ec "github.com/juju/juju/state/api/client.errorcodes"
	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/tools"
	"github.com/juju/juju/version"
)

// FindTools returns a List containing all tools matching the specified parameters.
func (c *Client) FindTools(majorVersion, minorVersion int,
	series, arch string) (result params.FindToolsResults, err error) {

	args := params.FindToolsParams{
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
		Arch:         arch,
		Series:       series,
	}
	err = c.call("FindTools", args, &result)
	return result, err
}

func (c *Client) UploadTools(
	toolsFilename string, vers version.Binary, fakeSeries ...string,
) (
	tools *tools.Tools, err error,
) {
	// Prepare the tool for uploading.
	toolsTarball, err := os.Open(toolsFilename)
	if err != nil {
		return nil, ec.PreprocessingError(err, "")
	}
	defer toolsTarball.Close()

	// Send the request.
	var result params.ToolsResult
	args := url.Values{}
	args.Set("binaryVersion", vers)
	args.Set("series", strings.Join(fakeSeries, ","))
	data := rawRPCData{File: toolsTarball, Mimetype: "application/x-tar-gz"}
	err := c.callRaw("tools", &args, &data, &result)
	if err != nil {
		return nil, err
	}

	return result.Tools, nil
}
