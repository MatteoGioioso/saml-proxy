package sharedKernel

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"time"
)

func GenerateCertificates(domain string) error {
	certPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "dashboard.mycoolsaml.com",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &certPrivateKey.PublicKey, certPrivateKey)
	if err != nil {
		return err
	}

	certPEM := new(bytes.Buffer)
	if err := pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return err
	}

	certPrivateKeyPEM := new(bytes.Buffer)
	if err := pem.Encode(certPrivateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivateKey),
	}); err != nil {
		return err
	}

	if err := ioutil.WriteFile(GetCertPath(domain), certPEM.Bytes(), fs.ModePerm); err != nil {
		return err
	}

	if err := ioutil.WriteFile(GetCertKeyPath(domain), certPrivateKeyPEM.Bytes(), fs.ModePerm); err != nil {
		return err
	}

	return nil
}

func GetCertPath(domain string) string {
	if os.Getenv("SSL_CERTIFICATE_AUTOGENERATE") == "true" {
		return getPath(domain, "crt")
	}

	return os.Getenv("SSL_CERTIFICATE_PATH")
}

func GetCertKeyPath(domain string) string {
	if os.Getenv("SSL_CERTIFICATE_AUTOGENERATE") == "true" {
		return getPath(domain, "key")
	}

	return os.Getenv("SSL_CERTIFICATE_KEY_PATH")
}

func getPath(domain, extension string) string {
	return fmt.Sprintf("assets/%s.%s", domain, extension)
}
