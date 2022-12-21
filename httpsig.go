package main

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-fed/httpsig"
)

// An ActivityPub actor object. Only the parts
// we care about.
type actorKeyObject struct {
	PublicKey struct {
		Id           string `json:"id"`
		Owner        string `json:"owner"`
		PublicKeyPem string `json:"publicKeyPem"`
	} `json:"publicKey"`
}

func validateHttpSig(r *http.Request, keycache KeyStore) error {
	verifier, err := httpsig.NewVerifier(r)
	if err != nil {
		return err
	}
	algorithm, err := getAlgorithm(r.Header.Get("Signature"))
	if err != nil {
		return err
	}

	pubkey, err := getKey(verifier.KeyId(), keycache)
	if err != nil {
		return err
	}
	if err := verifier.Verify(pubkey, algorithm); err != nil {
		log.Println(err)
		return err
	}
	log.Println("Validated sig")
	return nil
}

func getKey(keyId string, keycache KeyStore) (crypto.PublicKey, error) {
	if key, err := keycache.GetKey(keyId); err == nil {
		return key, nil
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", keyId, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var actor actorKeyObject
	if err := decoder.Decode(&actor); err != nil {
		return nil, err
	}
	if actor.PublicKey.Id != keyId {
		return nil, fmt.Errorf("Could not retrieve %v, got %v", keyId, actor.PublicKey.Id)
	}
	return parsePemKey(keyId, actor.PublicKey.Owner, []byte(actor.PublicKey.PublicKeyPem), keycache)
}

// httpsig doesn't like the algorithm header, but we do.
func getAlgorithm(header string) (httpsig.Algorithm, error) {
	pieces := strings.Split(header, ",")
	for _, s := range pieces {
		if !strings.HasPrefix(s, "algorithm=") {
			continue
		}
		val := strings.Replace(s, "algorithm=", "", 1)
		val = val[1 : len(val)-1]
		if val == "hs2019" {
			fmt.Printf("Warning: hs2019\n")
			return "rsa-sha256", nil
		}
		return httpsig.Algorithm(val), nil
	}
	return "", fmt.Errorf("Could not determine algorithm")
}

func parsePemKey(keyId, owner string, pemKey []byte, keycache KeyStore) (crypto.PublicKey, error) {
	pemBlock, _ := pem.Decode(pemKey)
	if pemBlock == nil {
		return nil, fmt.Errorf("No PEM block in key")
	}
	switch pemBlock.Type {
	case "PUBLIC KEY":

		key, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
		if err != nil {
			return nil, err
		}
		if keycache != nil {
			if err := keycache.SaveKey(keyId, owner, pemKey); err != nil {
				return nil, err
			}
		}
		return key, nil
	case "RSA PUBLIC KEY":
		key, err := x509.ParsePKCS1PublicKey(pemBlock.Bytes)
		if err != nil {
			return nil, err
		}
		if keycache != nil {
			if err := keycache.SaveKey(keyId, owner, pemKey); err != nil {
				return nil, err
			}
		}
		return key, nil
	default:
		return nil, fmt.Errorf("Could not key encoding from PEM")
	}
}
