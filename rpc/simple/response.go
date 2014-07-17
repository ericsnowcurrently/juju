// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package simple

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/juju/juju/rpc/errors"
)

func ReadResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func ReadJSONResponse(resp *http.Response, result interface{}) (error, error) {
	defer resp.Body.Close()

	err := json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		err = fmt.Errorf("could not unpack result: %v", err)
		return nil, err
	}

	errorResult, ok := result.(errors.FailableResult)
	if !ok {
		return nil, nil
	}

	failure := errorResult.Err()
	return failure, nil
}
