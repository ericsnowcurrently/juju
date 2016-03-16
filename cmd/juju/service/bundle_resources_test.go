// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service_test

import (
	"strings"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	charmresource "gopkg.in/juju/charm.v6-unstable/resource"

	"github.com/juju/juju/cmd/juju/service"
	"github.com/juju/juju/component/all"
	"github.com/juju/juju/resource"
	"github.com/juju/juju/testcharms"
)

func init() {
	if err := all.RegisterForServer(); err != nil {
		panic(err)
	}
}

type ResourcesBundleSuite struct {
	service.DeployRepoCharmStoreSuite
}

var _ = gc.Suite(&ResourcesBundleSuite{})

func (s *ResourcesBundleSuite) TestDeployBundleResources(c *gc.C) {
	testcharms.UploadCharm(c, s.Client, "trusty/starsay-42", "starsay")
	bundleMeta := `
        services:
            starsay:
                charm: cs:starsay
                num_units: 1
                resources:
                    for-store: 3
                    for-install: 17
                    for-upload: 42
    `
	output, err := s.DeployBundleYAML(c, bundleMeta)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(output, gc.Equals, strings.TrimSpace(`
added charm cs:trusty/starsay-42
service starsay deployed (charm: cs:trusty/starsay-42)
added resource for-install
added resource for-store
added resource for-upload
added starsay/0 unit to new machine
deployment of bundle "local:bundle/example-0" completed
    `))
	s.checkResources(c, "starsay", []resource.Resource{{
		Resource: charmresource.Resource{
			Meta: charmresource.Meta{
				Name:        "for-install",
				Type:        charmresource.TypeFile,
				Path:        "initial.tgz",
				Description: "get things started",
			},
			Origin:   charmresource.OriginStore,
			Revision: 17,
		},
		ID:        "starsay/for-install",
		ServiceID: "starsay",
	}, {
		Resource: charmresource.Resource{
			Meta: charmresource.Meta{
				Name:        "for-store",
				Type:        charmresource.TypeFile,
				Path:        "dummy.tgz",
				Description: "One line that is useful when operators need to push it.",
			},
			Origin:   charmresource.OriginStore,
			Revision: 3,
		},
		ID:        "starsay/for-store",
		ServiceID: "starsay",
	}, {
		Resource: charmresource.Resource{
			Meta: charmresource.Meta{
				Name:        "for-upload",
				Type:        charmresource.TypeFile,
				Path:        "config.xml",
				Description: "Who uses xml anymore?",
			},
			Origin:   charmresource.OriginStore,
			Revision: 42,
		},
		ID:        "starsay/for-upload",
		ServiceID: "starsay",
	}})
}

func (s *ResourcesBundleSuite) checkResources(c *gc.C, serviceName string, expected []resource.Resource) {
	_, err := s.State.Service("starsay")
	c.Check(err, jc.ErrorIsNil)
	st, err := s.State.Resources()
	c.Assert(err, jc.ErrorIsNil)
	svcResources, err := st.ListResources("starsay")
	c.Assert(err, jc.ErrorIsNil)
	resources := svcResources.Resources
	resource.Sort(resources)
	c.Assert(resources, jc.DeepEquals, expected)
}
