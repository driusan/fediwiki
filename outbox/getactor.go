package outbox

import (
	"io"
	"log"

	"encoding/json"

	"net/http"

	"fediwiki/activitypub"
)

func GetActor(cachedb activitypub.ActorDatabase, actorid string) (*activitypub.Actor, error) {
	if actor, err := cachedb.GetForeignActor(actorid); err == nil {
		log.Println("Got actor from cache")
		return actor, nil
	}

	req, err := http.NewRequest("GET", actorid, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams", application/ld+json, application/activity+json`)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var actor activitypub.Actor
	if err := json.Unmarshal(body, &actor); err != nil {
		return nil, err
	}
	if err := cachedb.StoreActor(actor, body); err != nil {
		// we unmarshalled it so don't return the error
		// from caching it, just print it
		log.Println(err)
	}

	return &actor, nil
}
