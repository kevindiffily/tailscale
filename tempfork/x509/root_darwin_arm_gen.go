// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// Generates root_darwin_arm64.go.
//
// As of iOS 8, there is no API for querying the system trusted X.509 root
// certificates. We could use SecTrustEvaluate to verify that a trust chain
// exists for a certificate, but the x509 API requires returning the entire
// chain.
//
// Apple publishes the list of trusted root certificates for iOS on
// support.apple.com. So we parse the list and extract the certificates from
// an OS X machine and embed them into the x509 package.
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
)

var output = flag.String("output", "root_darwin_arm64.go", "file name to write")

func main() {
	certs, err := selectCerts()
	if err != nil {
		log.Fatal(err)
	}

	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, "// Code generated by root_darwin_arm_gen --output %s; DO NOT EDIT.\n", *output)
	fmt.Fprintf(buf, "%s", header)
	for _, cert := range certs {
		gzbuf := new(bytes.Buffer)
		zw, err := gzip.NewWriterLevel(gzbuf, gzip.BestCompression)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := zw.Write(cert.Raw); err != nil {
			log.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(buf, "p.addCertFuncNotDup(%q, %q, certUncompressor(%q))\n",
			cert.RawSubject,
			cert.SubjectKeyId,
			gzbuf.Bytes())
	}
	fmt.Fprintf(buf, "%s", footer)

	source, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal("source format error:", err)
	}
	if err := ioutil.WriteFile(*output, source, 0644); err != nil {
		log.Fatal(err)
	}
}

func selectCerts() (certs []*x509.Certificate, err error) {
	pemCerts, err := ioutil.ReadFile("certs.pem")
	if err != nil {
		return nil, err
	}
	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

const header = `
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !x509omitbundledroots

package x509

func loadSystemRoots() (*CertPool, error) {
	p := NewCertPool()
`

const footer = `
	return p, nil
}
`