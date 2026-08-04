package agent

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoundedEmitterWriterKeepsStreamAndResultConsistent(t *testing.T) {
	buffer := newBoundedBuffer(maxCommandOutputBytes)
	var streamed bytes.Buffer
	writer := &boundedEmitterWriter{buffer: buffer, emit: func(content []byte) {
		_, _ = streamed.Write(content)
	}}
	input := bytes.Repeat([]byte("x"), maxCommandOutputBytes+1024)

	written, err := writer.Write(input)

	assert.NoError(t, err)
	assert.Equal(t, len(input), written)
	assert.Len(t, buffer.String(), maxCommandOutputBytes)
	assert.Equal(t, buffer.String(), streamed.String())
}
