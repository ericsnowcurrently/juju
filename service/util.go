// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"strings"
)

//TODO(ericsnow) Move hasPrefix to the utils repo.

func hasPrefix(name string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func contains(strList []string, str string) bool {
	for _, contained := range strList {
		if str == contained {
			return true
		}
	}
	return false
}

// fromSlash is borrowed from cloudinit/renderers.go.
func fromSlash(path string, initSystem string) string {
	// If initSystem is "" then just do the default.

	if initSystem == InitSystemWindows {
		return strings.Replace(path, "/", `\`, -1)
	}
	return path
}
