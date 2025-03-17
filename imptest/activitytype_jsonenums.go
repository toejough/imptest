// Code generated by jsonenums -type ActivityType; DO NOT EDIT.

package imptest

import (
	"encoding/json"
	"fmt"
)

var (
	_ActivityTypeNameToValue = map[string]ActivityType{
		"ActivityTypeUnset":  ActivityTypeUnset,
		"ActivityTypeReturn": ActivityTypeReturn,
		"ActivityTypePanic":  ActivityTypePanic,
		"ActivityTypeCall":   ActivityTypeCall,
	}

	_ActivityTypeValueToName = map[ActivityType]string{
		ActivityTypeUnset:  "ActivityTypeUnset",
		ActivityTypeReturn: "ActivityTypeReturn",
		ActivityTypePanic:  "ActivityTypePanic",
		ActivityTypeCall:   "ActivityTypeCall",
	}
)

func init() {
	var v ActivityType
	if _, ok := interface{}(v).(fmt.Stringer); ok {
		_ActivityTypeNameToValue = map[string]ActivityType{
			interface{}(ActivityTypeUnset).(fmt.Stringer).String():  ActivityTypeUnset,
			interface{}(ActivityTypeReturn).(fmt.Stringer).String(): ActivityTypeReturn,
			interface{}(ActivityTypePanic).(fmt.Stringer).String():  ActivityTypePanic,
			interface{}(ActivityTypeCall).(fmt.Stringer).String():   ActivityTypeCall,
		}
	}
}

// MarshalJSON is generated so ActivityType satisfies json.Marshaler.
func (r ActivityType) MarshalJSON() ([]byte, error) {
	if s, ok := interface{}(r).(fmt.Stringer); ok {
		return json.Marshal(s.String())
	}
	s, ok := _ActivityTypeValueToName[r]
	if !ok {
		return nil, fmt.Errorf("invalid ActivityType: %d", r)
	}
	return json.Marshal(s)
}

// UnmarshalJSON is generated so ActivityType satisfies json.Unmarshaler.
func (r *ActivityType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("ActivityType should be a string, got %s", data)
	}
	v, ok := _ActivityTypeNameToValue[s]
	if !ok {
		return fmt.Errorf("invalid ActivityType %q", s)
	}
	*r = v
	return nil
}
