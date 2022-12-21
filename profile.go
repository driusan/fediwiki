package main

import "encoding/json"

type PublicKey struct {
	Id           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}

type JsonLDContext []string

type Profile struct {
	Context           JsonLDContext `json:"@context"`
	Id                string        `json:"id"`
	Type              string        `json:"type"`
	PreferredUsername string        `json:"preferredUsername"`
	Title             string        `json:"name"`
	Summary           string        `json:"summary"`
	Inbox             string        `json:"inbox"`
	Outbox            string        `json:"outbox"`
	PublicKey         PublicKey     `json:"publicKey"`
}

func (c JsonLDContext) MarshalJSON() ([]byte, error) {
	switch len(c) {
	case 0:
		return nil, nil
	case 1:
		return json.Marshal(c[0])
	default:
		return json.Marshal([]string(c))
	}
}
