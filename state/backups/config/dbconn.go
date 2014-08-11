// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/utils/tar"

	"github.com/juju/juju/environmentserver/authentication"
	"github.com/juju/juju/mongo"
)

// DBConnInfo is a simplification of authentication.MongoInfo.
type DBConnInfo interface {
	// Address returns the connection address.
	Address() string
	// Username returns the connection username.
	Username() string
	// Password returns the connection password.
	Password() string
	// Checked returns the address, username, and password after
	// ensuring they are valid.
	Checked() (address, username, password string, err error)
}

type dbConnInfo struct {
	address  string
	username string
	password string
}

// NewDBConnInfo returns a new DBConnInfo.
func NewDBConnInfo(addr, user, pw string) DBConnInfo {
	dbinfo := dbConnInfo{
		address:  addr,
		username: user,
		password: pw,
	}
	return &dbinfo
}

func (ci *dbConnInfo) Address() string {
	return ci.address
}

func (ci *dbConnInfo) Username() string {
	return ci.username
}

func (ci *dbConnInfo) Password() string {
	return ci.password
}

// UpdateFromMongoInfo pulls in the provided connection info.
func (ci *dbConnInfo) UpdateFromMongoInfo(mgoInfo *authentication.MongoInfo) {
	ci.address = mgoInfo.Addrs[0]
	ci.password = mgoInfo.Password

	if mgoInfo.Tag != nil {
		ci.username = mgoInfo.Tag.String()
	}
}

func (ci *dbConnInfo) Checked() (addr, user, pw string, err error) {
	addr = ci.Address()
	user = ci.Username()
	pw = ci.Password()

	if addr == "" {
		err = errors.Errorf("missing address")
	} else if user == "" {
		err = errors.Errorf("missing username")
	} else if pw == "" {
		err = errors.Errorf("missing password")
	}

	return addr, user, pw, err
}
