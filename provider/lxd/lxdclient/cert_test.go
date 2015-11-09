// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build go1.3

package lxdclient_test

import (
	"bytes"
	"crypto/tls"
	"encoding/pem"
	"os"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/provider/lxd/lxdclient"
)

var (
	_ = gc.Suite(&certSuite{})
	_ = gc.Suite(&certFunctionalSuite{})
)

type certSuite struct {
	lxdclient.BaseSuite

	certPEM []byte
	keyPEM  []byte

	hostname string
}

func (s *certSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)

	s.certPEM = []byte("<a valid PEM-encoded x.509 cert>")
	s.keyPEM = []byte("<a valid PEM-encoded x.509 key>")
}

func (s *certSuite) osHostname() (string, error) {
	s.Stub.AddCall("osHostname")
	if err := s.Stub.NextErr(); err != nil {
		return "", errors.Trace(err)
	}

	return s.hostname, nil
}

func (s *certSuite) TestNewCert(c *gc.C) {
	cert := lxdclient.NewCert(s.certPEM, s.keyPEM)

	checkCert(c, cert, s.certPEM, s.keyPEM)
}

func (s *certSuite) TestWithDefaultsMissingName(c *gc.C) {
	s.hostname = "a.b.c"

	orig := lxdclient.NewCert(s.certPEM, s.keyPEM)
	cert, err := lxdclient.CertWithDefaults(orig, s.osHostname)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(cert, jc.DeepEquals, lxdclient.Cert{
		Name:    s.hostname,
		CertPEM: s.certPEM,
		KeyPEM:  s.keyPEM,
	})
}

func (s *certSuite) TestWithDefaultsNameError(c *gc.C) {
	failure := errors.New("<failed>")
	s.Stub.SetErrors(failure)

	orig := lxdclient.NewCert(s.certPEM, s.keyPEM)
	_, err := lxdclient.CertWithDefaults(orig, s.osHostname)

	c.Check(errors.Cause(err), gc.Equals, failure)
}

func (s *certSuite) TestValidateOkay(c *gc.C) {
	cert := lxdclient.NewCert(s.certPEM, s.keyPEM)
	cert.Name = "a.b.c"
	err := cert.Validate()

	c.Check(err, jc.ErrorIsNil)
}

func (s *certSuite) TestValidateMissingName(c *gc.C) {
	cert := lxdclient.NewCert(s.certPEM, s.keyPEM)
	cert.Name = ""
	err := cert.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
}

func (s *certSuite) TestValidateMissingCertPEM(c *gc.C) {
	cert := lxdclient.NewCert(nil, s.keyPEM)
	cert.Name = "a.b.c"
	err := cert.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
}

func (s *certSuite) TestValidateMissingKeyPEM(c *gc.C) {
	cert := lxdclient.NewCert(s.certPEM, nil)
	cert.Name = "a.b.c"
	err := cert.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
}

func (s *certSuite) TestWriteCertPEM(c *gc.C) {
	cert := lxdclient.NewCert(s.certPEM, s.keyPEM)
	var pemfile bytes.Buffer
	err := cert.WriteCertPEM(&pemfile)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(pemfile.String(), gc.Equals, string(s.certPEM))
}

func (s *certSuite) TestWriteKeyPEM(c *gc.C) {
	cert := lxdclient.NewCert(s.certPEM, s.keyPEM)
	var pemfile bytes.Buffer
	err := cert.WriteKeyPEM(&pemfile)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(pemfile.String(), gc.Equals, string(s.keyPEM))
}

func (s *certSuite) TestWritePEMs(c *gc.C) {
	cert := lxdclient.NewCert(s.certPEM, s.keyPEM)
	var pemfile bytes.Buffer
	err := cert.WriteCertPEM(&pemfile)
	c.Assert(err, jc.ErrorIsNil)
	err = cert.WriteKeyPEM(&pemfile)
	c.Assert(err, jc.ErrorIsNil)

	expected := string(s.certPEM) + string(s.keyPEM)
	c.Check(pemfile.String(), gc.Equals, expected)
}

func (s *certSuite) TestFingerprint(c *gc.C) {
	certPEM := []byte(testCertPEM)
	cert := lxdclient.NewCert(certPEM, nil)
	fingerprint, err := cert.Fingerprint()
	c.Assert(err, jc.ErrorIsNil)

	c.Check(fingerprint, gc.Equals, testCertFingerprint)
}

func (s *certSuite) TestX509Okay(c *gc.C) {
	certPEM := []byte(testCertPEM)
	cert := lxdclient.NewCert(certPEM, nil)
	x509Cert, err := cert.X509()
	c.Assert(err, jc.ErrorIsNil)

	block, _ := pem.Decode(certPEM)
	c.Assert(block, gc.NotNil)
	c.Check(string(x509Cert.Raw), gc.Equals, string(block.Bytes))
}

func (s *certSuite) TestX509ZeroValue(c *gc.C) {
	var cert lxdclient.Cert
	_, err := cert.X509()

	c.Check(err, gc.ErrorMatches, `invalid cert PEM \(0 bytes\)`)
}

func (s *certSuite) TestX509BadPEM(c *gc.C) {
	cert := lxdclient.NewCert(s.certPEM, s.keyPEM)
	_, err := cert.X509()

	c.Check(err, gc.ErrorMatches, `invalid cert PEM \(\d+ bytes\)`)
}

type certFunctionalSuite struct {
	lxdclient.BaseSuite
}

func (s *certFunctionalSuite) TestWithDefaultsMissingName(c *gc.C) {
	hostname, err := os.Hostname()
	c.Assert(err, jc.ErrorIsNil)

	certPEM, keyPEM := []byte("a cert"), []byte("a key")
	orig := lxdclient.NewCert(certPEM, keyPEM)
	cert, err := orig.WithDefaults()
	c.Assert(err, jc.ErrorIsNil)

	c.Check(cert, jc.DeepEquals, lxdclient.Cert{
		Name:    hostname,
		CertPEM: certPEM,
		KeyPEM:  keyPEM,
	})
}

func (s *certFunctionalSuite) TestGenerateCert(c *gc.C) {
	// This test involves the filesystem.
	certPEM, keyPEM, err := lxdclient.GenCertAndKey()
	c.Assert(err, jc.ErrorIsNil)
	cert := lxdclient.NewCert(certPEM, keyPEM)

	checkValidCert(c, &cert)
}

func checkCert(c *gc.C, cert lxdclient.Cert, certPEM, keyPEM []byte) {
	c.Check(cert, jc.DeepEquals, lxdclient.Cert{
		CertPEM: certPEM,
		KeyPEM:  keyPEM,
	})
	c.Check(string(cert.CertPEM), gc.Equals, string(certPEM))
	c.Check(string(cert.KeyPEM), gc.Equals, string(keyPEM))
}

func checkValidCert(c *gc.C, cert *lxdclient.Cert) {
	c.Assert(cert, gc.NotNil)

	_, err := tls.X509KeyPair(cert.CertPEM, cert.KeyPEM)
	c.Check(err, jc.ErrorIsNil)

	block, remainder := pem.Decode(cert.CertPEM)
	c.Check(block.Type, gc.Equals, "CERTIFICATE")
	c.Check(remainder, gc.HasLen, 0)

	block, remainder = pem.Decode(cert.KeyPEM)
	c.Check(block.Type, gc.Equals, "RSA PRIVATE KEY")
	c.Check(remainder, gc.HasLen, 0)
}

const (
	testCertFingerprint = "1c5156027fe71cfd0f7db807123e6873879f0f9754e08eab151f224783b2bff0"
	testCertPEM         = `
-----BEGIN CERTIFICATE-----
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
zxBNMbP3
-----END CERTIFICATE-----
`
)
