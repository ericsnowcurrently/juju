// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api

import (
	"net"
	"sort"
	"strconv"

	"github.com/juju/names"

	"github.com/juju/juju/network"
	"github.com/juju/juju/state/api/params"
)

// Login authenticates as the entity with the given name and password.
// Subsequent requests on the state will act as that entity.  This
// method is usually called automatically by Open. The machine nonce
// should be empty unless logging in as a machine agent.
func (st *State) Login(tag, password, nonce string) error {
	var result params.LoginResult
	err := st.Call("Admin", "", "Login", &params.Creds{
		AuthTag:  tag,
		Password: password,
		Nonce:    nonce,
	}, &result)
	if err == nil {
		authtag, err := names.ParseTag(tag)
		if err != nil {
			return err
		}
		st.authTag = authtag
		hostPorts, err := addAddress(result.Servers, st.addr)
		if err != nil {
			st.Close()
			return err
		}
		st.hostPorts = hostPorts
		st.environTag = result.EnvironTag
		st.facadeVersions = make(map[string][]int, len(result.Facades))
		for _, facade := range result.Facades {
			// The API will likely return versions in sorted order,
			// but we sort again so we don't have to trust that all
			// implementations always will.
			sort.Ints(facade.Versions)
			st.facadeVersions[facade.Name] = facade.Versions
		}
	}
	return err
}

// slideAddressToFront moves the address at the location (serverIndex, addrIndex) to be
// the first address of the first server.
func slideAddressToFront(servers [][]network.HostPort, serverIndex, addrIndex int) {
	server := servers[serverIndex]
	hostPort := server[addrIndex]
	// Move the matching address to be the first in this server
	for ; addrIndex > 0; addrIndex-- {
		server[addrIndex] = server[addrIndex-1]
	}
	server[0] = hostPort
	for ; serverIndex > 0; serverIndex-- {
		servers[serverIndex] = servers[serverIndex-1]
	}
	servers[0] = server
}

// addAddress appends a new server derived from the given
// address to servers if the address is not already found
// there.
func addAddress(servers [][]network.HostPort, addr string) ([][]network.HostPort, error) {
	for i, server := range servers {
		for j, hostPort := range server {
			if hostPort.NetAddr() == addr {
				slideAddressToFront(servers, i, j)
				return servers, nil
			}
		}
	}
	host, portString, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		return nil, err
	}
	hostPort := network.HostPort{
		Address: network.NewAddress(host, network.ScopeUnknown),
		Port:    port,
	}
	result := make([][]network.HostPort, 0, len(servers)+1)
	result = append(result, []network.HostPort{hostPort})
	result = append(result, servers...)
	return result, nil
}

// Client returns an object that can be used
// to access client-specific functionality.
func (st *State) Client() *Client {
	return &Client{st}
}

/*
// Agent returns a version of the state that provides
// functionality required by the agent code.
func (st *State) Agent() *agent.State {
	return agent.NewState(st)
}
*/
