package serialize

import (
	"github.com/lixianmin/got/convert"
)

// Serializer implements the serialize.Serializer interface
type JsonSerializer struct{}

// NewSerializer returns a new Serializer.
func NewJsonSerializer() *JsonSerializer {
	return &JsonSerializer{}
}

// Marshal returns the JSON encoding of v.
func (s *JsonSerializer) Marshal(v interface{}) ([]byte, error) {
	return convert.ToJsonE(v)
}

// Unmarshal parses the JSON-encoded data and stores the result
// in the value pointed to by v.
func (s *JsonSerializer) Unmarshal(data []byte, v interface{}) error {
	return convert.FromJsonE(data, v)
}

// GetName returns the name of the serializer.
func (s *JsonSerializer) GetName() string {
	return "json"
}
