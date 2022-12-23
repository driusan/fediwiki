package filesystemdb

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"testing"

	"fediwiki/activitypub"
	"fediwiki/pages"
)

var _ pages.PagesDatabase = &FileSystemDB{}

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

type testActorDB map[string]activitypub.Actor

func (db testActorDB) GetForeignActor(id string) (*activitypub.Actor, error) {
	v, ok := db[id]
	if !ok {
		return nil, NotFound
	}
	return &v, nil
}
func (db testActorDB) StoreActor(actor activitypub.Actor, raw []byte) error {
	db[actor.Id] = actor
	return nil
}
func TestGetFollowers(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "pagesfollowers")
	if err != nil {
		t.Fatal("Could not create temp dir for test")
	}
	defer os.RemoveAll(tmpdir)
	db := FileSystemDB{FSRoot: tmpdir}
	followers, err := db.GetPageFollowers("foo", nil)
	if err == nil {
		t.Error("Expected error for unknown page, got nil")
	}
	if len(followers) != 0 {
		t.Error("Unknown page had followers")
	}

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

	if _, err := db.NewPageActor(page, "example.com", key, &key.PublicKey); err != nil {
		t.Error(err)
	}

	followers, err = db.GetPageFollowers("Foo", nil)
	if err != nil {
		t.Error(err)
	}
	if len(followers) != 0 {
		t.Error("New page had followers")
	}
	followReq := activitypub.Follow{
		BaseProperties: activitypub.BaseProperties{
			Id:    "abc",
			Type:  "Follow",
			Actor: "Bar",
		},
		Object: "Foo",
	}
	if err := db.AddFollower("Foo", followReq); err != nil {
		t.Error(err)
	}

	testActors := testActorDB{
		"Bar": activitypub.Actor{Id: "Bar"},
	}

	followers, err = db.GetPageFollowers("Foo", testActors)
	if err != nil {
		t.Error(err)
	}
	if len(followers) != 1 {
		t.Errorf("Expected 1 follower, got %v", len(followers))
	}
	if followers[0].Id != "Bar" {
		t.Errorf("Wrong follower, want Bar; got %v", followers[0].Id)
	}

	undo := activitypub.Undo{
		BaseProperties: activitypub.BaseProperties{
			Id:    "abc2",
			Type:  "Undo",
			Actor: "Bar",
		},
		Object: followReq,
	}
	if err := db.UndoFollow("Foo", undo); err != nil {
		t.Error(err)
	}
	followers, err = db.GetPageFollowers("Foo", testActors)
	if err != nil {
		t.Error(err)
	}
	if len(followers) != 0 {
		t.Errorf("Expected 0 followers after undo, got %v", len(followers))
	}
}
