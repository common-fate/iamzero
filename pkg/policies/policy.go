package policies

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type AWSIAMPolicy struct {
	Version   string
	Id        *string `json:",omitempty"`
	Statement IAMStatements
}

// Value implements the driver.Valuer interface required to serialize the object to Postgres
func (p AWSIAMPolicy) Value() (driver.Value, error) { return json.Marshal(&p) }

// Scan implements the sql.Scanner interface required to deserialize the object from Postgres
func (p *AWSIAMPolicy) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		return json.Unmarshal(v, &p)
	case string:
		return json.Unmarshal([]byte(v), &p)
	default:
		return fmt.Errorf("Unsupported type: %T", v)
	}
}

type AWSIAMStatement struct {
	Sid       string
	Effect    string
	Action    StringOrStringArray
	Principal *AWSIAMPrincipal `json:",omitempty"`
	Resource  StringOrStringArray
}

// IAMStatements implements a custom UnmarshalJSON to handle
// cases where AWS returns a single statement with no enclosing
// array
type IAMStatements []AWSIAMStatement

func (s *IAMStatements) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("no bytes to unmarshal")
	}
	switch data[0] {
	case '{':
		return s.unmarshalSingle(data)
	case '[':
		return s.unmarshalMany(data)
	}
	return errors.New("unmarshalling: AWS IAM statement neither struct nor array")
}

func (s *IAMStatements) unmarshalSingle(data []byte) error {
	var res AWSIAMStatement
	err := json.Unmarshal(data, &res)
	if err != nil {
		return err
	}
	*s = []AWSIAMStatement{res}
	return nil
}

func (s *IAMStatements) unmarshalMany(data []byte) error {
	var res []AWSIAMStatement
	err := json.Unmarshal(data, &res)
	if err != nil {
		return err
	}
	*s = res
	return nil
}

// StringOrStringArray allows AWS statements to be unmarshalled
// into a string slice.
// Sometimes AWS policy JSONs contain a string, rather than a string array.
type StringOrStringArray []string

func (c *StringOrStringArray) UnmarshalJSON(data []byte) error {
	var tmp interface{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	slice, ok := tmp.([]interface{})
	if ok {
		for _, item := range slice {
			*c = append(*c, item.(string))
		}
		return nil
	}
	theString, ok := tmp.(string)
	if ok {
		*c = append(*c, theString)
		return nil
	}
	return errors.New("Field neither slice or string")
}

type AWSIAMPrincipal struct {
	AWS string
}
