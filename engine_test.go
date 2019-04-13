package engine

import (
	"testing"
)

func TestNewCertificate(t *testing.T) {
	// priv, cert, err := NewCertificate("name", []string{"test.com"}, "ecdsa", 384)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	//
	// keyBytes, err2 := x509.MarshalECPrivateKey(priv.(*ecdsa.PrivateKey))
	// if err2 != nil {
	// 	t.Fatal(err2)
	// }
	//
	// // pem.Encode(out io.Writer, b *pem.Block)
	// privateBlock := &pem.Block{
	// 	Type: "EC PRIVATE KEY",
	// 	Bytes: keyBytes,
	// }
	//
	// privatePem := pem.EncodeToMemory(privateBlock)
	// fmt.Println(string(privatePem))
	//
	// // pem.Encode(out io.Writer, b *pem.Block)
	// block := &pem.Block{
	// 	Type: "CERTIFICATE",
	// 	Bytes: cert,
	// }
	//
	// certPen := pem.EncodeToMemory(block)
	// fmt.Println(string(certPen))
	//
	// // block := &pem.Block{
	// 	Type: "PRIV",
	// 	Bytes: priv,
	// }
	//
	// if err := pem.Encode(os.Stdout, block); err != nil {
	// 	t.Fatal(err)
	// }

}
