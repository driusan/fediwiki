package activitypub

import (
	"encoding/json"
)

type JSONLDContext []string

func (c JSONLDContext) MarshalJSON() ([]byte, error) {
	switch len(c) {
	case 0:
		return nil, nil
	case 1:
		return json.Marshal(c[0])
	default:
		return json.Marshal([]string(c))
	}
}

// "@context":"https://www.w3.org/ns/activitystreams","id":"https://mas.to/f8d8e220-008c-4826-8bd4-e958fe92ae9e","type":"Follow","actor":"https://mas.to/users/driusan","object":"https://wiki.driusan.net/pages/FrontPage/actor"}
type FollowRequest struct {
	Context JSONLDContext `json:"@context"`
	Id      string        `json:"id"`
	Type    string        `json:"follow"`
	Actor   string        `json:"actor"`
	Object  string        `json:"object"`
}
