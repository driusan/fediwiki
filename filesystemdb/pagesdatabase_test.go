package filesystemdb

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"testing"

	"fediwiki/pages"
)

// func (db *FileSystemDB) NewPageActor(p pages.Page, domain string, private crypto.PrivateKey, public crypto.PublicKey) (*activitypub.Actor, error) {

func TestNewPageActor(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "pagestest")
	if err != nil {
		t.Fatal("Could not create temp dir for test")
	}
	defer os.RemoveAll(tmpdir)
	db := FileSystemDB{FSRoot: tmpdir}
	page := pages.Page{
		PageName: "Foo",
		Title:    "Foo title",
		Summary:  "yay",
		Content:  "hoooray",
	}

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}

	actor, err := db.NewPageActor(page, "example.com", key, &key.PublicKey)
	if err != nil {
		t.Error(err)
	}
	if actor.Name != page.Title {
		t.Error("Actor name not equal to page title")
	}
}
