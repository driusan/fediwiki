package pages

import (
	"crypto"

	"fediwiki/activitypub"
)

type PagesDatabase interface {
	activitypub.ActorDatabase
	GetPageActor(page string) (*activitypub.Actor, error)
	NewPageActor(page Page, domain string, private crypto.PrivateKey, public crypto.PublicKey) (*activitypub.Actor, error)
	GetPrivateKey(pagename string) (*activitypub.Actor, crypto.PrivateKey, error)
}
