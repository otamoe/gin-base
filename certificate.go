package server

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"time"
)

type (
	Certificate struct {
		Certificate string `json:"certificate"`
		PrivateKey  string `json:"private_key"`
	}
)

func NewCertificate(name string, hosts []string, typ string, bits int) (priv crypto.PrivateKey, cert []byte, err error) {
	var pub crypto.PublicKey
	switch typ {
	case "ecdsa":
		{
			var privateKey *ecdsa.PrivateKey
			switch bits {
			case 224:
				privateKey, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
			case 256:
				privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			case 384:
				privateKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
			case 521:
				privateKey, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
			}
			if err != nil {
				return
			}
			priv = privateKey
			pub = privateKey.Public()
		}
	default:
		{
			var privateKey *rsa.PrivateKey
			if privateKey, err = rsa.GenerateKey(rand.Reader, bits); err != nil {
				return
			}
			priv = privateKey
			pub = privateKey.Public()
		}
	}

	max := new(big.Int).Lsh(big.NewInt(1), 128)
	var serialNumber *big.Int
	if serialNumber, err = rand.Int(rand.Reader, max); err != nil {
		return
	}

	subject := pkix.Name{
		Organization:       []string{"Organization"},
		OrganizationalUnit: []string{"Organizational Unit"},
		CommonName:         name,
	}

	template := &x509.Certificate{
		SerialNumber:        serialNumber,
		Subject:             subject,
		NotBefore:           time.Now().Add(-(time.Hour * 24 * 30)),
		NotAfter:            time.Now().Add(time.Hour * 24 * 365 * 20),
		KeyUsage:            x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:         []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		PermittedDNSDomains: hosts,
		PermittedURIDomains: hosts,
	}

	if cert, err = x509.CreateCertificate(rand.Reader, template, template, pub, priv); err != nil {
		return
	}

	return
}

func EncodeCertificate(priv crypto.PrivateKey, cert []byte) (certificate Certificate, err error) {

	var privBytes []byte
	var typ string
	switch val := priv.(type) {
	case *ecdsa.PrivateKey:
		typ = "EC"
		if privBytes, err = x509.MarshalECPrivateKey(val); err != nil {
			return
		}
	case *rsa.PrivateKey:
		typ = "RSA"
		if privBytes, err = x509.MarshalPKCS8PrivateKey(val); err != nil {
			return
		}
	default:
		err = errors.New("Unknown certificate type")
		return
	}

	privBlock := &pem.Block{
		Type:  typ + " PRIVATE KEY",
		Bytes: privBytes,
	}
	privPem := pem.EncodeToMemory(privBlock)

	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}

	certPem := pem.EncodeToMemory(certBlock)

	certificate = Certificate{
		Certificate: string(certPem),
		PrivateKey:  string(privPem),
	}
	return
}
