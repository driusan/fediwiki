package activitypub

import (
	"encoding/json"
	"testing"
	"time"
)

func areBasePropertiesEqual(a, b BaseProperties) bool {
	if a.Id != b.Id {
		return false
	}
	if a.Type != b.Type {
		return false
	}
	if a.Actor != b.Actor {
		return false
	}
	return true
}

func isNoteEqual(a, b Note) bool {
	if v := areBasePropertiesEqual(a.BaseProperties, b.BaseProperties); v == false {
		return false
	}
	if a.Summary != b.Summary {
		return false
	}
	if a.InReplyTo != b.InReplyTo {
		return false
	}
	if a.Published == nil && b.Published != nil {
		return false
	} else if a.Published != nil && b.Published == nil {
		return false
	} else if a.Published == nil && b.Published == nil {
		// it's fine
	} else if a.Published.Equal(*b.Published) == false {
		return false
	}
	if a.AttributedTo != b.AttributedTo {
		return false
	}
	if len(a.To) != len(b.To) {
		return false
	}

	for i := range a.To {
		if a.To[i] != b.To[i] {
			return false
		}
	}

	if len(a.Cc) != len(b.Cc) {
		return false
	}

	for i := range a.Cc {
		if a.Cc[i] != b.Cc[i] {
			return false
		}
	}

	if a.MediaType != b.MediaType {
		return false
	}
	if a.Content != b.Content {
		return false
	}
	return true
}

func parseTime(val string) *time.Time {
	t, err := time.Parse("January 2, 2006 3:04:05PM MST", val)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestNoteUnmarshalJSON(t *testing.T) {
	tests := []struct {
		Input   []byte
		Want    Note
		WantErr bool
	}{
		{[]byte(`"foo"`), Note{}, true},
		{
			[]byte(
				`{
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
				"tag": []}`),
			Note{
				BaseProperties: BaseProperties{
					Id:   "https://driusan.net/users/driusan/posts/2022-12-22/18ec67b2-b6f4-5445-68a2-10563d11ede0",
					Type: "Note",
				},
				AttributedTo: "https://driusan.net/users/driusan",
				Published:    parseTime("December 22, 2022 9:17:25PM GMT"),
				To:           []string{"abc"},
				MediaType:    "text/markdown",
				Content:      "asdfsadf\n",
			},
			false,
		},
	}

	for _, tc := range tests {
		var result Note
		err := json.Unmarshal(tc.Input, &result)
		if tc.WantErr {
			if err == nil {
				t.Error("Unexpected error. Want error got nil")
				continue
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error. Want nil got %v", err)
				continue
			}
		}
		if isNoteEqual(tc.Want, result) == false {
			t.Errorf("Want %v got %v", tc.Want, result)
		}
	}
}
func TestNoteMarshalJSON(t *testing.T) {
	tests := []struct {
		Input   Note
		Want    []byte
		WantErr bool
	}{
		{Note{
			BaseProperties: BaseProperties{
				Id:   "id",
				Type: "Note",
			},
			To: []string{},
			Cc: []string{},
		}, []byte(`{"id":"id","type":"Note","summary":null,"inReplyTo":null,"to":[],"cc":[]}`), false},
	}

	for _, tc := range tests {
		bytes, err := json.Marshal(tc.Input)
		if tc.WantErr {
			if err == nil {
				t.Error("Unexpected error. Want error got nil")
				continue
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error. Want nil got %v", err)
				continue
			}
		}
		if string(tc.Want) != string(bytes) {
			t.Errorf("Unexpected value: want %v got %v", string(tc.Want), string(bytes))
		}
	}
}
