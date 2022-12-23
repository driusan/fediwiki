package activitypub

import (
	"reflect"
	"testing"
	/*
		"bytes"
		"fmt"
	*/

	"encoding/json"
)

func isJSONLDContextEqual(a, b JSONLDContext) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if reflect.TypeOf(a) != reflect.TypeOf(b) {
			return false
		}
		switch a[i].(type) {
		case string:
			if a[i].(string) != b[i].(string) {
				return false
			}
		case map[string]interface{}:
			a1 := a[i].(map[string]interface{})
			b1 := b[i].(map[string]interface{})
			if len(a1) != len(b1) {
				return false
			}
			for key := range a1 {
				if a1[key] != b1[key] {
					return false
				}
			}
		default:
			panic("Could not handle type")
		}
	}

	return true
}
func TestJSONLDContextUnmarshalJSON(t *testing.T) {
	tests := []struct {
		Input []byte
		Want  JSONLDContext
	}{
		{[]byte(`"foo"`), []interface{}{string("foo")}},
		{[]byte(`["foo", "bar"]`), []interface{}{string("foo"), string("bar")}},
		{[]byte(`["foo", {"foo": "bar"}]`), []interface{}{"foo", map[string]interface{}{"foo": "bar"}}},
	}
	for _, tc := range tests {
		var result JSONLDContext
		if err := json.Unmarshal(tc.Input, &result); err != nil {
			t.Error(err)
			continue
		}
		if isJSONLDContextEqual(result, tc.Want) != true {
			t.Errorf("Unexpected result. Want %v got %v", tc.Want, result)
		}
	}
}

/*
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
		*c = make(JSONLDContext, 0)
		*c = append(*c, "xx")
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
*/
