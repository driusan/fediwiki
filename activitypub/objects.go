package activitypub

import (
	"time"
)

type ObjectPersister interface {
	SaveObject(Object) error
}

type Object struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	RawBytes []byte `json:"-"`
}

type BaseProperties struct {
	Context JSONLDContext `json:"@context,omitempty"`
	Id      string        `json:"id"`
	Type    string        `json:"type"`
	Actor   string        `json:"actor,omitempty"`
}

type Follow struct {
	BaseProperties
	Object string `json:"object"`
}

// FIXME: Might not necessarily be an Undo follow. Other things can be undone.
type Undo struct {
	BaseProperties
	Object Follow `json:"object"`
}

type Accept struct {
	BaseProperties
	Object Follow `json:"object"`
}

/*
{
	"@context": "https://www.w3.org/ns/activitystreams",
	"id": "https://driusan.net/users/driusan/activities/53b1fec8-2caf-f142-9590-642990ae85a4",
	"type": "Create",
	"actor": "https://driusan.net/users/driusan",
	"to": [
		"abc"
	],
	"cc": [],
	"published": "2022-12-22T21:17:25Z",
	"object": {
		"id": "https://driusan.net/users/driusan/posts/2022-12-22/18ec67b2-b6f4-5445-68a2-10563d11ede0",
		"type": "Note",
		"summary": null,
		"inReplyTo": null,
		"published": "2022-12-22T21:17:25Z",
		"attributedTo": "https://driusan.net/users/driusan",
		"to": [
			"abc"
		],
		"cc": [],
		"mediaType": "text/markdown",
		"content": "asdfsadf\n",
		"attachment": [],
		"tag": []
	}
}
*/
// FIXME: Might not be a Note created
type CreateNote struct {
	BaseProperties
	To        []string   `json:"to"`
	Cc        []string   `json:"cc"`
	Published *time.Time `json:"published,omitempty"`
	Object    Note       `json:"object"`
}
