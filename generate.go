package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"os"
	"strconv"
	"time"
)


func generate(state *State, args []string) error {
	if len(args) < 1 {
		return errors.New("missing class\n\n" + getHelpGenerate())
	}

	var class string = args[0]
	var keySize int
	var keyType string
	var keyName string  // this will make it possible to have multiple keys with different names

	var err error

	commands := flag.NewFlagSet("generate", flag.ExitOnError)

	commands.StringVar(&keyType, "type", "ecdsa", "")
	commands.IntVar(&keySize, "size", 256, "")
	commands.StringVar(&keyName, "name", "", "")

	commands.Parse(args[1:])

	// Generate keys
	var privateHeader string
	var publicHeader string

	var privKeyMarshalled, pubKeyMarshalled []byte

	switch keyType {
	case "rsa":
		privKeyMarshalled, pubKeyMarshalled, privateHeader, publicHeader, err = generateKey(keyType, keySize)
		if err != nil {
			return err
		}
	case "ecdsa":
		privKeyMarshalled, pubKeyMarshalled, privateHeader, publicHeader, err = generateKey(keyType, keySize)
		if err != nil {
			return err
		}
	default:
		return errors.New("key type " + keyType + " is not available")
	}

	// Save them
	var path string

	switch class {
	case "root":
		path = rootPath
		keyName = "root"
	case "intermediate":
		path = intermediatesPath
		if keyName == "" {
			keyName = "intermediate"
		}
	case "client":
		path = clientsPath
		if keyName == "" {
			keyName = "client"
		}
	default:
		return errors.New("can't generate a " + class)
	}

	// Generate the folder if needed
	if _, err = os.Stat(path); os.IsNotExist(err) {
		err = os.Mkdir(path, 0700)
		if err != nil {
			return err
		}
	}

	// Prepare public and private key files
	privKeyFile, err := os.OpenFile(getPathPriv(class, keyName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer privKeyFile.Close()

	pubKeyFile, err := os.OpenFile(getPathPub(class, keyName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer pubKeyFile.Close()


	// Write keys
	pem.Encode(privKeyFile, &pem.Block{Type: privateHeader, Bytes: privKeyMarshalled})
	pem.Encode(pubKeyFile, &pem.Block{Type: publicHeader, Bytes: pubKeyMarshalled})

	// Update State
	(*state).set(class, keyName, Element{
		getPath(class, keyName),
		keyType,
		keySize,
		time.Now(),
		time.Now(),
	})

	return nil
}


func generateKey(keyType string, keySize int) (privKeyMarshalled, pubKeyMarshalled []byte, privHeader, pubHeader string, err error) {
	var keySizeStr string = strconv.Itoa(keySize)

	switch keyType {
	case "rsa":
		if keySize != 1024 && keySize != 2048 && keySize != 4096 {
			return []byte{}, []byte{}, "", "", errors.New(keySizeStr + " bits is not a valid size for a RSA key")
		}

		// Generate private and public keys
		privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
		if err != nil {
			return []byte{}, []byte{}, "", "", err
		}
		publicKey := &privateKey.PublicKey

		privKeyMarshalled = x509.MarshalPKCS1PrivateKey(privateKey)
		pubKeyMarshalled, err = x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			return []byte{}, []byte{}, "", "", err
		}

		return privKeyMarshalled, pubKeyMarshalled, "RSA PRIVATE KEY", "RSA PUBLIC KEY", nil
	case "ecdsa":
		var curve elliptic.Curve

		switch keySize{
		case 224:
			curve = elliptic.P224()
		case 256:
			curve = elliptic.P256()
		case 384:
			curve = elliptic.P384()
		case 521:
			curve = elliptic.P521()
		default:
			return []byte{}, []byte{}, "", "", errors.New(keySizeStr + " bits keys size are not available")
		}

		privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return []byte{}, []byte{}, "", "", err
		}
		publicKey := &privateKey.PublicKey

		privKeyMarshalled, err = x509.MarshalECPrivateKey(privateKey)
		if err != nil {
			return []byte{}, []byte{}, "", "", err
		}
		pubKeyMarshalled, err = x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			return []byte{}, []byte{}, "", "", err
		}

		return privKeyMarshalled, pubKeyMarshalled, "PRIVATE EC KEY", "PUBLIC EC KEY", nil
	}

	return []byte{}, []byte{}, "", "", nil
}


func getHelpGenerate() string {
	return `Usage: simpleca generate [class] --type=<type> --size=<size> --name=<name>

Available classes:
	root           generate a root CA key pair
	intermediate   generate an intermediate CA key pair
	client         generate a client key pair

--type string
	(optional) The key type. Possible values: "ecdsa", "rsa". Defaults to "ecdsa".

--size string
	(optional) The key size (depends on the key type). Possible values: "224", "256", "384", "521" for EC key types;
	"1024", "2048", "4096" for RSA. Defaults to "256".

--name string
	(optional) The key name. This allows you to have multiple key of a kind (this is particularly useful to have
	multiple client keys).
`
}
