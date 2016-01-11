// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package server

import (
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/juju/errors"
	charmresource "gopkg.in/juju/charm.v6-unstable/resource"

	"github.com/juju/juju/resource"
)

// UploadDataStore describes the the portion of Juju's "state"
// needed for handling upload requests.
type UploadDataStore interface {
	uploadStorage

	// ListResources returns the resources for the given service.
	ListResources(service string) ([]resource.Resource, error)
}

// UploadedResource holds both the information about an uploaded
// resource and the reader containing its data.
type UploadedResource struct {
	// Service is the name of the service associated with the resource.
	Service string

	// Resource is the information about the resource.
	Resource resource.Resource

	// Data holds the resource blob.
	Data io.ReadCloser
}

// uploadStorage describes the the portion of Juju's "state needed for upload.
type uploadStorage interface {
	// SetResource adds the resource to blob storage and updates the metadata.
	SetResource(serviceID string, res resource.Resource, r io.Reader) error
}

// UploadHandler
type UploadHandler struct {
	Username string
	Store    DataStore

	CurrentTimestamp func() time.Time
}

// HandleRequest handles a resource upload request.
func (uh UploadHandler) HandleRequest(req *http.Request) error {
	defer req.Body.Close()

	uploaded, err := uh.ReadResource(req)
	if err != nil {
		return errors.Trace(err)
	}

	if err := uh.Store.SetResource(uploaded.Service, uploaded.Resource, uploaded.Data); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// ReadResource extracts the relevant info from the request.
func (uh UploadHandler) ReadResource(req *http.Request) (*UploadedResource, error) {
	ctype := req.Header.Get("Content-Type")
	if ctype != "application/octet-stream" {
		return nil, errors.Errorf("unsupported content type %q", ctype)
	}

	// See HTTPEndpoint in server.go and pattern handling in apiserver/apiserver.go.
	service := req.URL.Query().Get(":service")
	name := req.URL.Query().Get(":resource")

	fingerprint := req.URL.Query().Get("fingerprint")
	size := req.Header.Get("Content-Length")

	res, err := getResource(uh.Store, service, name)
	if err != nil {
		return nil, errors.Trace(err)
	}

	res, err = uh.updateResource(res, fingerprint, size)
	if err != nil {
		return nil, errors.Trace(err)
	}

	uploaded := &UploadedResource{
		Service:  service,
		Resource: res,
		Data:     req.Body,
	}
	return uploaded, nil
}

// updateResource returns a copy of the provided resource, updated with
// the given information.
func (uh UploadHandler) updateResource(res resource.Resource, fingerprint, size string) (resource.Resource, error) {
	data, err := hex.DecodeString(fingerprint)
	if err != nil {
		return res, errors.Annotate(err, "invalid fingerprint")
	}
	fp, err := charmresource.NewFingerprint(data)
	if err != nil {
		return res, errors.Annotate(err, "invalid fingerprint")
	}

	sizeInt, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		return res, errors.Annotate(err, "invalid size")
	}

	res.Origin = charmresource.OriginUpload
	res.Revision = 0
	res.Fingerprint = fp
	res.Size = sizeInt
	res.Username = uh.Username
	res.Timestamp = uh.CurrentTimestamp().UTC()

	if err := res.Validate(); err != nil {
		return res, errors.Trace(err)
	}
	return res, nil
}