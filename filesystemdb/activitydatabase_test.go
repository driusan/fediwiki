package filesystemdb

import (
	"fediwiki/activitypub"
)

var _ activitypub.ActivityDatabase = &FileSystemDB{}
