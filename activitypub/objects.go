package activitypub

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
	Actor   string        `json:"actor"`
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
