// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build go1.3

package lxdclient

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/juju/errors"
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
)

// addServer adds the given remote info to the provided config.
// The implementation is based loosely on:
//  https://github.com/lxc/lxd/blob/master/lxc/remote.go
func addServer(config *lxd.Config, server string, remoteURL string) error {
	remoteURL, err := fixURL(remoteURL)
	if err != nil {
		return err
	}

	if config.Remotes == nil {
		config.Remotes = make(map[string]lxd.RemoteConfig)
	}

	/* Actually add the remote */
	// TODO(ericsnow) Fail on collision?
	config.Remotes[server] = lxd.RemoteConfig{Addr: remoteURL}

	return nil
}

func fixURL(orig string) (string, error) {
	if orig == "" {
		// TODO(ericsnow) Return lxd.LocalRemote.Addr?
		return orig, nil
	}
	if strings.HasPrefix(orig, "unix:") {
		return "", errors.NewNotValid(nil, fmt.Sprintf("unix socket URLs not supported (got %q)", orig))
	}

	// Fix IPv6 URLs.
	if strings.HasPrefix(orig, ":") {
		parts := strings.SplitN(orig, "/", 2)
		if net.ParseIP(parts[0]) != nil {
			orig = fmt.Sprintf("[%s]", parts[0])
			if len(parts) == 2 {
				orig += "/" + parts[1]
			}
		}
	}

	parsedURL, err := url.Parse(orig)
	if err != nil {
		return "", errors.Trace(err)
	}
	if parsedURL.RawQuery != "" {
		return "", errors.NewNotValid(nil, fmt.Sprintf("URL queries not supported (got %q)", orig))
	}
	if parsedURL.Fragment != "" {
		return "", errors.NewNotValid(nil, fmt.Sprintf("URL fragments not supported (got %q)", orig))
	}
	if parsedURL.Opaque != "" {
		if strings.Contains(parsedURL.Scheme, ".") {
			orig, err := fixURL("https://" + orig)
			if err != nil {
				return "", errors.Trace(err)
			}
			return orig, nil
		}
		return "", errors.NewNotValid(nil, fmt.Sprintf("opaque URLs not supported (got %q)", orig))
	}

	URL := url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
		Path:   strings.TrimRight(parsedURL.Path, "/"),
	}

	// Fix the scheme.
	URL.Scheme = fixScheme(URL)
	if err := validateScheme(URL); err != nil {
		return "", errors.Trace(err)
	}

	// Fix the host.
	if URL.Host == "" {
		if strings.HasPrefix(URL.Path, "/") {
			return "", errors.NewNotValid(nil, fmt.Sprintf("unix socket URLs not supported (got %q)", orig))
		}
		orig = fmt.Sprintf("%s://%s%s", URL.Scheme, URL.Host, URL.Path)
		orig, err := fixURL(orig)
		if err != nil {
			return "", errors.Trace(err)
		}
		return orig, nil
	}
	URL.Host = fixHost(URL.Host, shared.DefaultPort)
	if err := validateHost(URL); err != nil {
		return "", errors.Trace(err)
	}

	return URL.String(), nil
}

func fixScheme(url url.URL) string {
	switch url.Scheme {
	case "https":
		return url.Scheme
	case "http":
		return "https"
	case "":
		return "https"
	default:
		return url.Scheme
	}
}

func validateScheme(url url.URL) error {
	switch url.Scheme {
	case "https":
	default:
		return errors.NewNotValid(nil, fmt.Sprintf("unsupported URL scheme %q", url.Scheme))
	}
	return nil
}

func fixHost(host, defaultPort string) string {
	// Handle IPv6 hosts.
	if strings.Count(host, ":") > 1 {
		if !strings.HasPrefix(host, "[") {
			return fmt.Sprintf("[%s]:%s", host, defaultPort)
		} else if !strings.Contains(host, "]:") {
			return host + ":" + defaultPort
		}
		return host
	}

	// Handle ports.
	if !strings.Contains(host, ":") {
		return host + ":" + defaultPort
	}

	return host
}

func validateHost(url url.URL) error {
	if url.Host == "" {
		return errors.NewNotValid(nil, "URL missing host")
	}

	host, port, err := net.SplitHostPort(url.Host)
	if err != nil {
		return errors.NewNotValid(err, "")
	}

	// Check the host.
	if net.ParseIP(host) == nil {
		if err := validateDomainName(host); err != nil {
			return errors.Trace(err)
		}
	}

	// Check the port.
	if p, err := strconv.Atoi(port); err != nil {
		return errors.NewNotValid(err, fmt.Sprintf("invalid port in host %q", url.Host))
	} else if p <= 0 || p > 0xFFFF {
		return errors.NewNotValid(err, fmt.Sprintf("invalid port in host %q", url.Host))
	}

	return nil
}

func validateDomainName(fqdn string) error {
	// TODO(ericsnow) Do checks for a valid domain name.

	return nil
}
