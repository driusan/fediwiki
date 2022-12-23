package filesystemdb

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"path/filepath"

	"fediwiki/activitypub"
	"fediwiki/pages"

	"github.com/mischief/ndb"
)

func (db *FileSystemDB) GetPageActor(page string) (*activitypub.Actor, error) {
	filename := filepath.Join(db.FSRoot, pages.Root, page, "actor.json")
	if !strings.HasPrefix(filename, db.FSRoot+pages.Root) {
		// They were trying to use ../.. or something to escape the
		// filesystem
		return nil, fmt.Errorf("Invalid page name")
	}
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		return nil, NotFound
	}
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var p activitypub.Actor
	if err := json.Unmarshal(bytes, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *FileSystemDB) NewPageActor(p pages.Page, domain string, private crypto.PrivateKey, public crypto.PublicKey) (*activitypub.Actor, error) {
	pageurl := "https://" + domain + pages.Root + p.PageName
	id := pageurl + "/actor"
	keybytes, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return nil, err
	}
	block := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: keybytes})
	prof := &activitypub.Actor{
		Context:           activitypub.JSONLDContext{"https://www.w3.org/ns/activitystreams", "https://w3id.org/security/v1"},
		Id:                id,
		Type:              "Service",
		PreferredUsername: p.PageName,
		Name:              p.Title,
		Summary:           p.Summary,
		Inbox:             pageurl + "/inbox",
		Outbox:            pageurl + "/outbox",
		PublicKey: activitypub.PublicKey{
			Id:           id + "#main-key",
			Owner:        id,
			PublicKeyPem: string(block),
		},
	}
	filedir := filepath.Join(db.FSRoot, pages.Root, p.PageName)
	if !strings.HasPrefix(filedir, db.FSRoot+pages.Root) {
		// Make sure no ones trying to escape with a ../../ or something
		return nil, fmt.Errorf("Unknown error creating directory")
	}
	if err := os.MkdirAll(filedir, 0775); err != nil {
		return nil, err
	}

	filename := filepath.Join(filedir, "actor.json")

	bytes, err := json.Marshal(prof)
	if err != nil {
		return nil, err
	}
	privkeybytes, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(filedir, "private.pem"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privkeybytes}), 0400); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filename, bytes, 0664); err != nil {
		return nil, err
	}
	return prof, nil
}

func (d *FileSystemDB) GetPrivateKey(pagename string) (*activitypub.Actor, crypto.PrivateKey, error) {
	actor, err := d.GetPageActor(pagename)
	if err != nil {
		return nil, nil, err
	}
	pagedir := filepath.Join(d.FSRoot, pages.Root, pagename)
	privkeybytes, err := os.ReadFile(filepath.Join(pagedir, "private.pem"))
	if err != nil {
		return nil, nil, err
	}

	pemblock, _ := pem.Decode(privkeybytes)

	switch pemblock.Type {
	case "PRIVATE KEY":
		privkey, err := x509.ParsePKCS8PrivateKey(pemblock.Bytes)
		if err != nil {
			return nil, nil, err
		}
		return actor, privkey, nil
	case "RSA PRIVATE KEY":
		privkey, err := x509.ParsePKCS1PrivateKey(pemblock.Bytes)
		if err != nil {
			return nil, nil, err
		}
		return actor, privkey, nil
	default:
		return nil, nil, fmt.Errorf("Unknown key type")
	}
}

func (d *FileSystemDB) GetPageFollowers(pagename string, actors activitypub.ActorDatabase) ([]activitypub.Actor, error) {
	pagedir := filepath.Join(d.FSRoot, pages.Root, pagename)
	dbname := filepath.Join(pagedir, "followers.db")
	if _, err := os.Stat(pagedir); errors.Is(err, os.ErrNotExist) {
		return nil, NotFound
	}
	if _, err := os.Stat(dbname); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	followdb, err := ndb.Open(dbname)
	if err != nil {
		return nil, err
	}

	records := followdb.Search("accepted", "true")
	var result []activitypub.Actor = nil
	for _, record := range records {
		var actor activitypub.Actor
		var acceptId string
		for _, t := range record {
			switch t.Attr {
			case "id":
				a2, err := actors.GetForeignActor(t.Val)
				if err != nil {
					return nil, err
				}
				actor = *a2
			case "acceptedFrom":
				acceptId = t.Val
			}
		}
		if d.isUndone(acceptId) == false {
			result = append(result, actor)
		}
	}

	return result, nil
}

func (d *FileSystemDB) isUndone(id string) bool {
	filename := filepath.Join(d.FSRoot, "undo.db")
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		return false
	}

	undodb, err := ndb.Open(filename)
	if err != nil {
		panic(err)
	}
	records := undodb.Search("id", id)
	return len(records) > 0

}
