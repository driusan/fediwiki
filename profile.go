package main

type PublicKey struct {
	Id           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}
type Profile struct {
	Id                string    `json:"id"`
	Type              string    `json:"type"`
	PreferredUsername string    `json:"preferredUsername"`
	Title             string    `json:"name"`
	Summary           string    `json:"summary"`
	Inbox             string    `json:"inbox"`
	Outbox            string    `json:"outbox"`
	PublicKey         PublicKey `json:"publicKey"`
}
