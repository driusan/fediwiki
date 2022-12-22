package outbox

import (
	"testing"

	"crypto/rand"
	"crypto/rsa"

	"fediwiki/activitypub"
)

func TestRequestSigning(t *testing.T) {
	req, err := makeRequest(activitypub.Actor{
		Inbox: "https://example.com/foo",
	}, []byte{'a', 'b'})
	if err != nil {
		t.Fatal(err)
	}
	if req.URL.Path != "/foo" {
		t.Errorf("Unexpect path: want /foo got %v", req.URL.Path)
	}
	if domain := req.URL.Hostname(); domain != "example.com" {
		t.Errorf("Unexpect domain: want example.com got %v", domain)
	}
	if contenttype := req.Header.Get("Content-Type"); contenttype != `application/ld+json; profile="https://www.w3.org/ns/activitystreams"` {
		t.Errorf("Unexpect Content-Type: got %v", contenttype)
	}

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := signRequest(key, "foo", req, []byte{'a', 'b'}); err != nil {
		t.Error(err)
	}
}

/*
FIXME: Move this to a test marshal function in activitypub package
func TestSend(t *testing.T) {
	accept := activitypub.Accept{
		BaseProperties: activitypub.BaseProperties{
			Context: activitypub.JSONLDContext{"https://www.w3.org/ns/activitystreams"},
			Type:    "Accept",
			Actor:   "test",
			Id:      "test#accept",
		},
		Object: activitypub.Follow{
			BaseProperties: activitypub.BaseProperties{
				Type:  "Follow",
				Actor: "bob",
				Id:    "testid",
			},
			Object: "test",
		},
	}
	objbytes, err := json.Marshal(accept)
	if err != nil {
		t.Fatal(err)
	}
	log.Println(accept)
	log.Println(string(objbytes))


	t.Fail()
}
*/
