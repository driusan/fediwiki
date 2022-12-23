package inbox

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"fediwiki/activitypub"
	"fediwiki/outbox"
	"fediwiki/pages"
)

func newAcceptId(pageowner string) string {
	var idrand [36]byte
	if _, err := rand.Read(idrand[:]); err != nil {
		panic(err)
	}

	return fmt.Sprintf("https://%s/pages/%s/#accept-%s", os.Getenv("fediwikidomain"), pageowner, base64.URLEncoding.EncodeToString(idrand[:]))
}
func HandleFollow(pagesdb pages.PagesDatabase, actordb activitypub.ActorDatabase, db activitypub.ActivityDatabase, request activitypub.Follow) error {
	pagename, err := pages.GetPageNameFromActorId(request.Object)
	if err != nil {
		return err
	}
	if err := db.AddFollower(pagename, request); err != nil {
		return err
	}
	ctx := request.Context
	fmt.Printf("%v", ctx)
	request.Context = nil
	id := newAcceptId(pagename)
	accept := activitypub.Accept{
		BaseProperties: activitypub.BaseProperties{
			Context: activitypub.JSONLDContext{"https://www.w3.org/ns/activitystreams"},
			Type:    "Accept",
			Actor:   request.Object,
			Id:      id,
		},
		Object: request,
	}

	fmt.Println(accept)
	objbytes, err := json.Marshal(accept)
	if err != nil {
		return err
	}
	actor, err := outbox.GetActor(actordb, request.Actor)
	if err != nil {
		return err
	}
	log.Println("Actor", actor)

	return outbox.Send(pagesdb, pagename, *actor, activitypub.Object{
		Id:       id,
		Type:     "Accept",
		RawBytes: objbytes,
	})
}

func HandleUndo(db activitypub.ActivityDatabase, request activitypub.Undo) error {
	pagename, err := pages.GetPageNameFromActorId(request.Object.Object)
	if err != nil {
		return err
	}
	if err := db.UndoFollow(pagename, request); err != nil {
		return err
	}
	return nil
}

func HandleCreateNote(db activitypub.ActivityDatabase, incoming activitypub.CreateNote) error {
	if incoming.Type != "Create" || incoming.Object.Type != "Note" {
		return fmt.Errorf("Bad create note")
	}

	for _, to := range incoming.Object.To {
		pname, err := pages.GetPageNameFromActorId(to)
		if err == nil {
			if err := db.AddPageNote(pname, incoming.Object); err != nil {
				return err
			}
		}
	}
	for _, to := range incoming.Object.Cc {
		pname, err := pages.GetPageNameFromActorId(to)
		if err == nil {
			if err := db.AddPageNote(pname, incoming.Object); err != nil {
				return err
			}
		}
	}
	return nil
}

func Process(objectDB activitypub.ObjectDatabase, pagesdb pages.PagesDatabase, actorDb activitypub.ActorDatabase, activityDb activitypub.ActivityDatabase, incoming activitypub.Object) error {
	if objectDB != nil {
		if err := objectDB.SaveObject(incoming); err != nil {
			return err
		}
	}
	switch incoming.Type {
	case "Follow":
		var f activitypub.Follow
		if err := json.Unmarshal(incoming.RawBytes, &f); err != nil {
			return err
		}
		if err := HandleFollow(pagesdb, actorDb, activityDb, f); err != nil {
			return err
		}
	case "Undo":
		var u activitypub.Undo
		if err := json.Unmarshal(incoming.RawBytes, &u); err != nil {
			return err
		}
		if err := HandleUndo(activityDb, u); err != nil {
			return err
		}
	case "Create":
		var c activitypub.CreateNote
		if err := json.Unmarshal(incoming.RawBytes, &c); err != nil {
			return err
		}
		if c.Object.Type != "Note" {
			return fmt.Errorf("Unhandled create type %v", c.Object.Type)
		}
		if err := HandleCreateNote(activityDb, c); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unhandled object type %v", incoming.Type)
	}
	return nil
}
