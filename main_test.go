package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDatabaseWorkflow(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration test")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(ctx, schema); err != nil {
		t.Fatal(err)
	}
	app := (&App{DB: db}).routes()
	title := "MemoryBox integration " + t.Name()
	req := httptest.NewRequest("POST", "/notes", strings.NewReader(`{"title":"`+title+`","subject":"Biology","content":"Mitochondria release energy."}`))
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	var note Note
	if err = json.Unmarshal(w.Body.Bytes(), &note); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(ctx, "DELETE FROM notes WHERE id=$1", note.ID)
	for _, path := range []string{"/healthz", "/notes?q=Mitochondria&subject=Biology", "/export?format=json&q=Mitochondria", "/export?format=markdown&q=Mitochondria"} {
		w = httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if path != "/healthz" && !strings.Contains(w.Body.String(), "Mitochondria") {
			t.Fatalf("missing persisted note in %s", path)
		}
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"message":{"content":"Energy release summary"}}`)
	}))
	defer provider.Close()
	t.Setenv("OLLAMA_URL", provider.URL)
	t.Setenv("OLLAMA_MODEL", "stub")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest("POST", "/notes/"+strconv.FormatInt(note.ID, 10)+"/summarize", nil))
	if w.Code != 200 {
		t.Fatalf("summary: %d %s", w.Code, w.Body.String())
	}
	var original, summary string
	if err := db.QueryRow(ctx, "SELECT content,summary FROM notes WHERE id=$1", note.ID).Scan(&original, &summary); err != nil {
		t.Fatal(err)
	}
	if original != note.Content || summary != "Energy release summary" {
		t.Fatal("summary did not preserve original content")
	}
	if reply := (&App{DB: db}).command(ctx, "!note search Mitochondria"); !strings.Contains(reply, title) {
		t.Fatalf("chat search: %s", reply)
	}
	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest("POST", "/notes", strings.NewReader(`{"title":"empty"}`)))
	if w.Code != 400 {
		t.Fatalf("invalid note accepted: %d", w.Code)
	}
}
