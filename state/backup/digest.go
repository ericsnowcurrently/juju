// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backup

// We could move these to juju/utils/hash

import (
	"fmt"
	"net/http"
	"strings"
)

// See RFC3230.

// For now we support only 1 digest algorithm: SHA-1 ("SHA").

func IsSupportedDigestAlgorithm(algorithm string) bool {
	return algorithm == DigestAlgorithm
}

func DigestTokensAreQuoted(algorithm string) bool {
	return false
}

func AddDigestHeader(header http.Header, algorithm, token string) error {
	if !IsSupportedDigestAlgorithm(algorithm) {
		return fmt.Errorf("unsupported digest algorithm: %s", algorithm)
	}
	value := fmt.Sprintf("%s=%s", algorithm, token)
	if header.Get("Digest") != "" {
		// We could simply append (with a comma) and check for dupes...
		return fmt.Errorf("multiple digests is not supported")
	}
	header.Set("Digest", value)
	return nil
}

func ParseDigest(digest string) (string, string, error) {
	parts := strings.SplitN(digest, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("malformed digest: %s", digest)
	}
	algorithm, token := parts[0], parts[1]
	if DigestTokensAreQuoted(algorithm) {
		err := fmt.Errorf("quoted digest tokens not supported")
		return algorithm, token, err
	}
	return algorithm, token, nil
}

func ParseDigestHeader(header http.Header) (digests map[string]string, err error) {
	digests = map[string]string{}

	raw := header.Get("Digest")
	if raw == "" {
		return
	}

	// For now we do not support quoted digest tokens which contain commas.
	for _, value := range strings.Split(raw, ",") {
		var algorithm, token string
		algorithm, token, err = ParseDigest(value)
		if err != nil {
			return
		}
		_, exists := digests[algorithm]
		if exists {
			err = fmt.Errorf("duplicate digest: %s", algorithm)
			return
		}
		digests[algorithm] = token
	}
	return
}
