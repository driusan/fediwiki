package main

import (
	"crypto"
	"time"
)

type OAuthClient struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Website      string `json:"website"`
	RedirectURI  string `json:"redirect_uri"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type OAuthClientStore interface {
	GetClient(hostname string) (OAuthClient, error)
	StoreClient(hostname string, c OAuthClient) error
}

type KeyStore interface {
	GetKey(keyid string) (crypto.PublicKey, error)
	SaveKey(keyid, owner string, pemBytes []byte) error
}
type ObjectPersister interface {
	SaveObject(id, body string) error
}

type ActorPersister interface {
	GetPageActor(page string) (*Profile, error)
	NewPageActor(page Page, domain string, private crypto.PrivateKey, public crypto.PublicKey) (*Profile, error)
}

type Page struct {
	PageName string
	Title    string
	Summary  string
	Content  string
}

type Revision struct {
	PageName   string
	RevisionID string
	Editor     string
	EditTime   *time.Time
}

type PagePersister interface {
	GetPage(pagename string) (*Page, error)
	SavePage(page Page, pageactor Profile, editor string) error
	GetPageRevisions(pagename string) ([]Revision, error)
	GetPageRevision(pagename, revisionid string) (*Page, error)
}
