package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSummarizeProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body struct {
			Model    string              `json:"model"`
			Stream   bool                `json:"stream"`
			Messages []map[string]string `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Model != "test-model" || body.Stream || len(body.Messages) != 2 || body.Messages[1]["content"] != "Original notes" {
			t.Errorf("unexpected request: %+v", body)
		}
		fmt.Fprint(w, `{"message":{"content":"A concise summary"}}`)
	}))
	defer server.Close()
	t.Setenv("OLLAMA_URL", server.URL)
	t.Setenv("OLLAMA_MODEL", "test-model")
	result, err := summarize(context.Background(), "Original notes")
	if err != nil || result != "A concise summary" {
		t.Fatalf("%q %v", result, err)
	}
}
func TestSummarizeDisabled(t *testing.T) {
	t.Setenv("OLLAMA_URL", "")
	got, err := summarize(context.Background(), "A short biology note.")
	if err != nil || !strings.Contains(got, "Main idea: A short biology note.") || !strings.Contains(got, "Study tip:") {
		t.Fatalf("fallback summary %q, %v", got, err)
	}
}
func TestSwitchProtocol(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	posted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing API key")
		}
		switch r.URL.Path {
		case "/agents/test/events":
			if r.URL.Query().Get("rooms") != "room-1" || r.URL.Query().Get("connection_id") == "" {
				t.Error("missing stream scope")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "event: connection_state\ndata: {\"cursor\":0,\"heartbeat_interval_seconds\":2}\n\n")
			fmt.Fprint(w, "id: 1\nevent: message\ndata: {\"type\":\"message\",\"room_id\":\"room-1\",\"sequence\":1,\"payload\":{\"addressed\":true,\"body\":\"!note help\"}}\n\n")
		case "/agents/test/message":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["room_id"] != "room-1" || !strings.Contains(body["content"], "!note save") {
				t.Errorf("bad reply: %v", body)
			}
			posted = true
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	client := switchClient{base: server.URL, id: "test", key: "test-key", room: "room-1", connection: "connection-test", app: &App{}}
	client.session(ctx)
	if !posted || client.cursor.Load() != 1 {
		t.Fatal("message did not complete")
	}
}
