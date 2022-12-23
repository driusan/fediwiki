package activitypub

import (
	"net/url"
)

type PublicKey struct {
	Id           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}

type Actor struct {
	Context           JSONLDContext `json:"@context"`
	Id                string        `json:"id"`
	Type              string        `json:"type"`
	PreferredUsername string        `json:"preferredUsername"`
	Name              string        `json:"name"`
	Summary           string        `json:"summary"`
	Inbox             string        `json:"inbox"`
	Outbox            string        `json:"outbox"`
	Following         string        `json:"following,omitempty"`
	Followers         string        `json:"followers,omitempty"`
	ProfileIcon       string        `json:"profileicon,omitempty"`
	PublicKey         PublicKey     `json:"publicKey"`
}

func (a Actor) MentionName() string {
	if a.PreferredUsername == "" {
		return a.Id
	}
	u, err := url.Parse(a.Id)
	if err != nil {
		return a.Id
	}
	return "@" + a.PreferredUsername + "@" + u.Hostname()
}
