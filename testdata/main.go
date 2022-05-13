package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if err := makeCerts(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func makeCerts() error {
	ca, err := makeCert(nil, "ca")
	if err != nil {
		return err
	}

	_, err = makeCert(ca, "grafana")
	if err != nil {
		return err
	}

	_, err = makeCert(ca, "client")
	if err != nil {
		return err
	}

	return nil
}

func makeCert(ca *x509.Certificate, name string) (*x509.Certificate, error) {
	var crt *x509.Certificate

	if ca == nil {
		now := time.Now()

		ca = &x509.Certificate{
			BasicConstraintsValid: true,
			Subject: pkix.Name{
				Organization: []string{"Raintank, Inc."},
			},
			SerialNumber: big.NewInt(1024),
			ExtKeyUsage: []x509.ExtKeyUsage{
				x509.ExtKeyUsageClientAuth,
				x509.ExtKeyUsageServerAuth,
			},
			KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			NotBefore: now,
			NotAfter:  now.Add(1 * time.Hour),
			IsCA:      true,
		}

		crt = ca
	} else {
		// copy CA data
		crt = &x509.Certificate{}
		*crt = *ca

		// overwrite CA data that's not needed for certificates
		crt.IsCA = false
		crt.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	}

	pk, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("cannot generate RSA key for certificate %s: %w", name, err)
	}

	crtBytes, err := x509.CreateCertificate(rand.Reader, crt, ca, &pk.PublicKey, pk)
	if err != nil {
		return nil, fmt.Errorf("cannot create certificate %s: %w", name, err)
	}

	return crt, writeFiles(crtBytes, pk, name)
}

func writeFiles(crtBytes []byte, pk *rsa.PrivateKey, name string) error {
	var err error

	name, err = filepath.Abs(name)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)

	err = pem.Encode(buf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crtBytes,
	})
	if err != nil {
		return fmt.Errorf("cannot PEM encode %s: %w", name, err)
	}

	err = os.WriteFile(name+".crt", buf.Bytes(), 0600)
	if err != nil {
		return fmt.Errorf("cannot write certificate %s: %w", name, err)
	}

	buf.Reset()

	err = pem.Encode(buf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(pk),
	})
	if err != nil {
		return fmt.Errorf("cannot PEM encode RSA key %s: %w", name, err)
	}

	err = os.WriteFile(name+".key", buf.Bytes(), 0600)
	if err != nil {
		return fmt.Errorf("cannot write certificate RSA key %s: %w", name, err)
	}

	fmt.Printf("created %s.key and %[1]s.crt\n", name)

	return nil
}
