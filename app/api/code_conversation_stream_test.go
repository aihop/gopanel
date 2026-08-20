package api

import (
	"testing"
	"time"
)

func TestConversationOutputWriterStreamsGrokACPChunks(t *testing.T) {
	hub := newConversationStreamHub()
	codeConversationStreams = hub
	_, events, cancel := hub.Subscribe(9)
	defer cancel()
	writer := &conversationOutputWriter{inner: &boundedCodeOutput{}, executorID: "grok", sessionID: 9}
	line := []byte(`{"method":"session/update","params":{"update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"hello"}}}}` + "\n")
	if _, err := writer.Write(line); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Type != "delta" || event.Content != "hello" {
			t.Fatalf("unexpected stream event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected grok assistant delta")
	}
}
