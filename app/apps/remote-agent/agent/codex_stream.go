package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type codexJSONStreamWriter struct {
	mu      sync.Mutex
	pending []byte
	result  *boundedBuffer
	emit    ChunkEmitter
	emitted int
}

type codexJSONEvent struct {
	Type    string          `json:"type"`
	Message string          `json:"message"`
	Error   json.RawMessage `json:"error"`
	Item    struct {
		Type             string `json:"type"`
		Text             string `json:"text"`
		Command          string `json:"command"`
		AggregatedOutput string `json:"aggregated_output"`
		ExitCode         *int   `json:"exit_code"`
	} `json:"item"`
}

func newCodexJSONStreamWriter(emit ChunkEmitter) *codexJSONStreamWriter {
	return &codexJSONStreamWriter{result: newBoundedBuffer(maxCommandOutputBytes), emit: emit}
}

func (w *codexJSONStreamWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending = append(w.pending, data...)
	for {
		index := bytes.IndexByte(w.pending, '\n')
		if index < 0 {
			break
		}
		w.consumeLine(w.pending[:index])
		w.pending = w.pending[index+1:]
	}
	return len(data), nil
}

func (w *codexJSONStreamWriter) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(bytes.TrimSpace(w.pending)) > 0 {
		w.consumeLine(w.pending)
	}
	w.pending = nil
}

func (w *codexJSONStreamWriter) Result() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.TrimSpace(w.result.String())
}

func (w *codexJSONStreamWriter) consumeLine(line []byte) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	var event codexJSONEvent
	if err := json.Unmarshal(line, &event); err != nil {
		w.emitText(string(line) + "\n")
		return
	}
	switch event.Type {
	case "item.started":
		if event.Item.Type == "command_execution" && strings.TrimSpace(event.Item.Command) != "" {
			w.emitText("$ " + strings.TrimSpace(event.Item.Command) + "\n")
		}
	case "item.completed":
		w.consumeCompletedItem(event)
	case "error", "turn.failed":
		message := strings.TrimSpace(event.Message)
		if message == "" {
			message = codexErrorMessage(event.Error)
		}
		if message != "" {
			w.emitText("Codex: " + message + "\n")
		}
	}
}

func (w *codexJSONStreamWriter) consumeCompletedItem(event codexJSONEvent) {
	switch event.Item.Type {
	case "agent_message":
		text := strings.TrimSpace(event.Item.Text)
		if text == "" {
			return
		}
		if w.result.buffer.Len() > 0 {
			_, _ = w.result.Write([]byte("\n\n"))
		}
		_, _ = w.result.Write([]byte(text))
		w.emitText(text + "\n")
	case "reasoning":
		if text := strings.TrimSpace(event.Item.Text); text != "" {
			w.emitText(text + "\n")
		}
	case "command_execution":
		if output := strings.TrimSpace(event.Item.AggregatedOutput); output != "" {
			w.emitText(output + "\n")
		}
		if event.Item.ExitCode != nil && *event.Item.ExitCode != 0 {
			w.emitText(fmt.Sprintf("Command exited with code %d\n", *event.Item.ExitCode))
		}
	}
}

func (w *codexJSONStreamWriter) emitText(content string) {
	if w.emit == nil || content == "" || w.emitted >= maxCommandOutputBytes {
		return
	}
	data := []byte(content)
	remaining := maxCommandOutputBytes - w.emitted
	if len(data) > remaining {
		data = data[:remaining]
	}
	w.emitted += len(data)
	w.emit(data)
}

func codexErrorMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var message string
	if json.Unmarshal(raw, &message) == nil {
		return strings.TrimSpace(message)
	}
	var detail struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &detail) == nil {
		return strings.TrimSpace(detail.Message)
	}
	return ""
}
