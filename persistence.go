package main

import "crypto"

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

type ActorPersister interface {
	GetPageActor(page string) (*Profile, error)
	SavePageActor(page, domain string, key crypto.PublicKey) error
}

type Page struct {
	PageName string
	Title    string
	Summary  string
	Content  string
}
type PagePersister interface {
	GetPage(pagename string) (*Page, error)
	SavePage(page Page) error
}
