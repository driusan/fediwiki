package pages

import (
	"time"

	"fediwiki/activitypub"
)

const Root = "/pages/"

type Page struct {
	PageName string
	Title    string
	Summary  string
	Content  string
}

type Revision struct {
	PageName   string
	RevisionID string
	Editor     string
	EditTime   *time.Time
}

type Persister interface {
	GetPage(pagename string) (*Page, error)
	SavePage(page Page, pageactor activitypub.Actor, editor string) error
	GetPageRevisions(pagename string) ([]Revision, error)
	GetPageRevision(pagename, revisionid string) (*Page, error)
}
