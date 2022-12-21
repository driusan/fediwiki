package outbox

import (
	// "fmt"
	"bytes"
	"crypto"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-fed/httpsig"

	"fediwiki/activitypub"
	"fediwiki/pages"
)

func Send(pagesdb pages.PagesDatabase, frompage string, toactor activitypub.Actor, obj activitypub.Object) error {
	pageactor, privkey, err := pagesdb.GetPrivateKey(frompage)
	if err != nil {
		return err
	}
	req, err := makeRequest(toactor, obj.RawBytes)
	if err != nil {
		return err
	}

	if err := signRequest(privkey, pageactor.PublicKey.Id, req, obj.RawBytes); err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error", err)
	}
	log.Println(string(respBody))
	return nil
}

func makeRequest(toactor activitypub.Actor, body []byte) (*http.Request, error) {
	req, err := http.NewRequest("POST", toactor.Inbox, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
	req.Header.Set("Host", req.URL.Hostname())

	now := time.Now().In(time.UTC)
	req.Header.Set("Date", now.Format(http.TimeFormat))
	return req, err
}

func signRequest(privateKey crypto.PrivateKey, pubKeyId string, r *http.Request, body []byte) error {
	prefs := []httpsig.Algorithm{httpsig.RSA_SHA256}
	digestAlgorithm := httpsig.DigestSha256
	// The "Date" and "Digest" headers must already be set on r, as well as r.URL.
	headersToSign := []string{httpsig.RequestTarget, "date", "digest", "host", "content-type"}
	signer, _, err := httpsig.NewSigner(prefs, digestAlgorithm, headersToSign, httpsig.Signature, 60*60*24*30)
	if err != nil {
		return err
	}
	return signer.SignRequest(privateKey, pubKeyId, r, body)
}
