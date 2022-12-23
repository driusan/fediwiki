package filesystemdb

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"path/filepath"

	"fediwiki/activitypub"
	"fediwiki/pages"

	"github.com/mischief/ndb"
)

func (d *FileSystemDB) SendUnprocessedObjects(objstream chan activitypub.Object, wg *sync.WaitGroup) chan string {
	wg.Add(1)
	returnstream := make(chan string)
	go func() {
		for {
			id := <-returnstream
			filename := filepath.Join(d.FSRoot, "objects", "processed.db")
			f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
			if err != nil {
				log.Println(err)
				wg.Done()
				continue
			}
			if _, err := io.WriteString(f, "\nid="+id); err != nil {
				f.Close()
				log.Println(err)
				wg.Done()
				continue
			}
			f.Close()
			wg.Done()
		}
	}()

	ndbdb, err := ndb.Open(filepath.Join(d.FSRoot, "objects", "objects.db"))
	if err != nil {
		log.Println(err)
		return returnstream
	}
	processedb, _ := ndb.Open(filepath.Join(d.FSRoot, "objects", "processed.db"))
	go func() {
		records := ndbdb.Search("type", "Follow")
		for _, r := range records {
			var obj activitypub.Object
			for _, tuple := range r {
				switch tuple.Attr {
				case "id":
					obj.Id = tuple.Val
				case "type":
					obj.Type = tuple.Val
				case "cachepath":
					bytes, err := os.ReadFile(filepath.Join(d.FSRoot, "objects", tuple.Val))

					if err != nil {
						log.Println(err)
					}
					obj.RawBytes = bytes
				}
			}
			if processedb != nil {
				processed := processedb.Search("id", obj.Id)
				if len(processed) != 0 {
					continue
				}
			}
			if obj.RawBytes != nil {
				wg.Add(1)
				objstream <- obj
			}
		}
		records = ndbdb.Search("type", "Undo")
		for _, r := range records {
			var obj activitypub.Object
			for _, tuple := range r {
				switch tuple.Attr {
				case "id":
					obj.Id = tuple.Val
				case "type":
					obj.Type = tuple.Val
				case "cachepath":
					bytes, err := os.ReadFile(filepath.Join(d.FSRoot, "objects", tuple.Val))

					if err != nil {
						log.Println(err)
					}
					obj.RawBytes = bytes
				}
			}
			if processedb != nil {
				processed := processedb.Search("id", obj.Id)
				if len(processed) != 0 {
					continue
				}
			}
			if obj.RawBytes != nil {
				wg.Add(1)
				objstream <- obj
			}
		}
		wg.Done()
	}()
	return returnstream
}
func (d *FileSystemDB) AddFollower(pagename string, request activitypub.Follow) error {
	filename := filepath.Join(d.FSRoot, pages.Root, pagename, "followers.db")
	followdb, err := ndb.Open(filename)

	if records := followdb.Search("acceptedFrom", request.Id); len(records) != 0 {
		return fmt.Errorf("Request already processed")
	}
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	record := fmt.Sprintf("\nid=%s accepted=true PageName=%s acceptedFrom=%s\n", request.Actor, pagename, request.Id)
	if _, err := f.WriteString(record); err != nil {
		return err
	}
	return nil
}
func (d *FileSystemDB) UndoFollow(pagename string, undo activitypub.Undo) error {
	filename := filepath.Join(d.FSRoot, "undo.db")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	record := fmt.Sprintf("\nid=%s type=Undo\n", undo.Object.Id)
	if _, err := f.WriteString(record); err != nil {
		return err
	}
	return nil
}
