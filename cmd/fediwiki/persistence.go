package main

import (
	"crypto"

	"fediwiki/activitypub"
	"fediwiki/pages"
)

type ObjectPersister interface {
	SaveObject(id, Type string, body []byte) error
}

type ActorPersister interface {
	GetPageActor(page string) (*activitypub.Actor, error)
	NewPageActor(page pages.Page, domain string, private crypto.PrivateKey, public crypto.PublicKey) (*activitypub.Actor, error)
}
