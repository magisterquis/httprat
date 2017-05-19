package main

/*
 * tls.go
 * Get or make a TLS pair
 * By J. Stuart McMurray
 * Created 20170518
 * Last Modified 20170518
 */

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

/* ORGANIZATION is used in the generate TLS cert */
const ORGANIZATION = "Kittens, Inc."

/* VALIDDAYS is the number of days the generated TLS certificat is valid */
const VALIDDAYS = 3652

/* RSABITS is the number of bits in the generated RSA key */
//const RSABITS = 4096
const RSABITS = 1024

/* getCertificate gets a certificate from the key and cert files, if both are
not the empty string, or generates one for the given address.  It terminates
the program on error or if either keyf or certf is empty but the other isn't */
func getCertificate(laddr, certf, keyf string) tls.Certificate {
	var cert tls.Certificate

	if "" == certf && "" == keyf {
		/* If we have no TLS cert, make one */
		return makeCertificate(laddr)
	} else if "" == certf || "" == keyf {
		/* If we only have one or the other, tell the user we need
		both */
		log.Fatalf(
			"Need both a PEM-encoded certificate (-cert) " +
				"and key (-key)\n",
		)
	} else {
		/* Load the given keypair */
		var err error
		cert, err = tls.LoadX509KeyPair(certf, keyf)
		if nil != err {
			log.Fatalf(
				"Unable to load keypair from %v and %v: %v",
				certf,
				keyf,
				err,
			)
		}
		log.Printf("Loaded TLS keypair from %v and %v", certf, keyf)
	}
	return cert
}

/* makeCertificate generates a self-signed TLS certificate for the given
address.  It terminates the program on error. */
func makeCertificate(addr string) tls.Certificate {
	/* Split port from address */
	h, _, err := net.SplitHostPort(addr)
	if nil != err {
		log.Fatalf(
			"Unable to split %v into host from port: %v",
			addr,
			err,
		)
		os.Exit(1)
	}
	log.Printf("Generating TLS keypair for %v", h)

	/* Most of the below stolen from
		https://golang.org/src/crypto/tls/generate_cert.go
		under the following license

	Copyright (c) 2009 The Go Authors. All rights reserved.

	Redistribution and use in source and binary forms, with or without
	modification, are permitted provided that the following conditions are
	met:

	   * Redistributions of source code must retain the above copyright
	notice, this list of conditions and the following disclaimer.
	   * Redistributions in binary form must reproduce the above
	copyright notice, this list of conditions and the following disclaimer
	in the documentation and/or other materials provided with the
	distribution.
	   * Neither the name of Google Inc. nor the names of its
	contributors may be used to endorse or promote products derived from
	this software without specific prior written permission.

	THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
	"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
	LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
	A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
	OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
	SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
	LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
	DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
	THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
	(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
	OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
	*/

	priv, err := rsa.GenerateKey(rand.Reader, RSABITS)
	if nil != err {
		log.Fatalf("Unable to generate RSA key: %v", err)
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	notBefore := time.Now()
	notAfter := notBefore.Add(VALIDDAYS * 24 * time.Hour)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{ORGANIZATION},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
		IsCA: true,
	}
	if ip := net.ParseIP(h); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, h)
	}
	log.Printf("%T %T", priv.PublicKey, priv) /* DEBUG */
	derBytes, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&priv.PublicKey,
		priv,
	)
	if nil != err {
		log.Fatalf("Failed to create certificate: %v", err)
	}
	cpem := pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: derBytes},
	)
	kpem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)
	cert, err := tls.X509KeyPair(cpem, kpem)
	if nil != err {
		log.Fatalf("Unable to parse generated PEM blocks: %v", err)
	}
	log.Printf("Made certificate for %v", h)
	return cert
}
