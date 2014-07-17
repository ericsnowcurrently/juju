// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package simple

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type APIRequest struct {
	Host    *APIHost
	Binding *APIBinding
	EnvUUID string
	Query   *url.Values // We could also use per-APIBinding structs.
	Auth    *HTTPBasicAuth
	Payload *HTTPPayload
}

func NewAPIRequest(hostname string, binding *APIBinding, uuid string) *APIRequest {
	host := NewAPIHost(hostname)
	req := APIRequest{
		Host:    host,
		Binding: binding,
		EnvUUID: uuid,
	}
	return &req
}

func (ar *APIRequest) UpdateFromURL(URL *url.URL) error {
	idx := strings.LastIndex(URL.Path, `/`)
	if idx < 0 {
		return fmt.Errorf("invalid path: %s", URL.Path)
	}
	ar.Host.Scheme = URL.Scheme
	ar.Host.Hostname = URL.Host
	ar.Host.RootPath = URL.Path[0 : idx+1]
	ar.Binding.Name = URL.Path[idx+1 : len(URL.Path)]
	query := URL.Query()
	ar.Query = &query
	return nil
}

func (ar *APIRequest) SetArg(name, value string) {
	if ar.Query == nil {
		ar.Query = &url.Values{}
	}
	ar.Query.Set(name, value)
}

func (ar *APIRequest) SetQuery(query *url.Values) {
	for name := range *query {
		value := query.Get(name)
		ar.SetArg(name, value)
	}
}

func (ar *APIRequest) SetRawQuery(rawquery string) error {
	query, err := url.ParseQuery(rawquery)
	if err != nil {
		return err
	}
	ar.SetQuery(&query)
	return nil
}

func (ar *APIRequest) RawQuery() string {
	if ar.Query == nil {
		return ""
	} else {
		return ar.Query.Encode()
	}
}

func (ar *APIRequest) SetAuth(userid, password string) {
	auth := HTTPBasicAuth{
		UserID:   userid,
		Password: password,
	}
	ar.Auth = &auth
}

func (ar *APIRequest) SetPayload(data io.Reader) {
	payload := HTTPPayload{
		Data:     data,
		Mimetype: ar.Binding.Mimetype,
	}
	ar.Payload = &payload
}

func (ar *APIRequest) URL() *url.URL {
	return ar.Host.URL(ar.EnvUUID, ar.Binding.Name, ar.RawQuery())
}

func (ar *APIRequest) Raw() (*http.Request, error) {
	httpmethod := ar.Binding.HTTPMethod
	url := ar.URL().String()

	var body io.Reader
	if ar.Payload != nil {
		body = ar.Payload.Data
	}

	req, err := http.NewRequest(httpmethod, url, body)
	if err != nil {
		return nil, err
	}
	if ar.Auth != nil {
		ar.Auth.AddHeader(req)
	}
	if ar.Payload != nil {
		ar.Payload.AddHeader(req)
	}
	return req, nil
}

func (ar *APIRequest) Send(client HTTPDoer) (*http.Response, error) {
	req, err := ar.Raw()
	if err != nil {
		return nil, err
	}
	return SendRequest(client, req)
}
