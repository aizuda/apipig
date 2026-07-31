package toolkit

import (
	"bytes"
	"encoding/gob"
)

// Serialize 将结构体序列化为 []byte
func Serialize(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize 将 []byte 反序列化为结构体
func Deserialize(data []byte, v interface{}) error {
	buf := bytes.NewReader(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(v)
}
