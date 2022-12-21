package activitypub

import (
	"sync"
)

type ActivityDatabase interface {
	SendUnprocessedObjects(obstream chan Object, wg *sync.WaitGroup) chan string

	AddFollower(pagename string, request Follow) error
	UndoFollow(pagename string, request Undo) error
}

type ActorDatabase interface {
	GetForeignActor(url string) (*Actor, error)
	StoreActor(actor Actor, rawobject []byte) error
}

type ObjectDatabase interface {
	SaveObject(Object) error
}
