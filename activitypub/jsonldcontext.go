package activitypub

import (
	"bytes"
	"fmt"

	"encoding/json"
)

type JSONLDContext []interface{}

func (c JSONLDContext) MarshalJSON() ([]byte, error) {
	switch len(c) {
	case 0:
		return nil, nil
	case 1:
		return json.Marshal(c[0])
	default:
		var r []string
		for _, val := range c {
			switch s := val.(type) {
			case string:
				r = append(r, s)
			default:
				return nil, fmt.Errorf("Unhandle context type")
			}
		}
		return json.Marshal(r)
	}
}

func (c *JSONLDContext) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	if trimmed == nil || len(trimmed) == 0 {
		*c = nil
		return nil
	}
	switch trimmed[0] {
	case '"':
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		*c = []interface{}{string(s)}
		return nil
	case '[':
		var asSlice []json.RawMessage = make([]json.RawMessage, 0)
		if err := json.Unmarshal(trimmed, &asSlice); err != nil {
			return err
		}
		*c = make(JSONLDContext, 0)
		for _, raw := range asSlice {
			var piece JSONLDContext
			if err := json.Unmarshal(raw, &piece); err != nil {
				return err
			}
			*c = append(*c, piece...)
		}
		return nil
	case '{':
		m := make(map[string]interface{})
		if err := json.Unmarshal(trimmed, &m); err != nil {
			return err
		}
		*c = []interface{}{m}
		return nil
	default:
		return fmt.Errorf("Could not unmarshal %s", b)
	}
}
