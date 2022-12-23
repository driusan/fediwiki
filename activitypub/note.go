package activitypub

import (
	"time"
)

type Note struct {
	BaseProperties
	Summary      *string    `json:"summary"`
	InReplyTo    *string    `json:"inReplyTo"`
	To           []string   `json:"to"`
	Cc           []string   `json:"cc"`
	Published    *time.Time `json:"published,omitempty"`
	Url          string     `json:"url,omitempty"`
	AttributedTo string     `json:"attributedTo,omitempty"`
	MediaType    string     `json:"mediaType,omitempty"`
	Content      string     `json:"content,omitempty"`
}
