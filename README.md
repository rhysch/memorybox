# MemoryBox

A local student note organizer written in Go, with PostgreSQL persistence.

## Hackathon demo

Watch the [three-minute MemoryBox demo](demo/MemoryBox-Hackathon-Demo.mp4). The
[narration transcript](demo/Narration.txt) and [captions](demo/Captions.srt) are
included as well.

MemoryBox helps students keep class notes together, including typed notes and
text captured from photos or PDFs. It supports search, export, and optional AI
summaries that explain key ideas in simpler language.

## Stage 1: database and notes (no AI required)

Requirements: Go 1.22+, Docker with Compose.

```bash
docker compose up -d --wait db
go mod download
export DATABASE_URL='postgres://memorybox:memorybox-local@localhost:55432/memorybox?sslmode=disable'
go run .
```

In another terminal:

```bash
curl http://localhost:8080/healthz
curl -X POST http://localhost:8080/notes \
  -H 'Content-Type: application/json' \
  -d '{"title":"Cell biology","subject":"Biology","content":"Mitochondria release energy through cellular respiration."}'
curl 'http://localhost:8080/notes?q=mitochondria'
curl 'http://localhost:8080/notes?subject=Biology'
curl 'http://localhost:8080/export?format=markdown' -o notes.md
curl 'http://localhost:8080/export?format=json' -o notes.json
```

Search is literal, case-insensitive substring matching. Listings return the newest 200 matching notes; exports include every matching note. Notes persist in the `memorybox-db` Docker volume. Schema creation happens at startup. Generated summaries are stored separately from the source notes.

The web form can also capture/select a note photo or upload a PDF. PDF.js extracts selectable PDF text and renders scanned pages for Tesseract.js OCR in the browser; photos use Tesseract.js OCR. The extracted text and original attachment are saved together in PostgreSQL, then MemoryBox generates a summary. The PDF.js/Tesseract libraries and English OCR data load from public CDNs, so this needs an internet connection. Uploads are limited to 8 MB and PDFs to 20 pages. Use **Print / Save PDF** and choose **Save as PDF** in the browser print dialog to create a PDF of the visible notebook.

You can remove a mistaken note from the web interface with its **Delete** button; MemoryBox asks for confirmation before permanently deleting it. The API also supports `DELETE /notes/{id}` and returns `404` when the note does not exist.

Run the real database integration check:

```bash
TEST_DATABASE_URL='postgres://memorybox:memorybox-local@localhost:55432/memorybox?sslmode=disable' go test -v ./...
go vet ./...
```

The integration test creates and removes its own note. It checks persistence, search, subject filtering, exports, health, and validation.

## Build order

2. Connect Switch and Mattermost using the official Switch deployment and agent protocol (adapter implemented; live setup described below).
3. Enable the Go organizing agent’s optional summarizer after database verification.

The database stage must pass before enabling an AI provider. Keep original note text when generating summaries. Label summaries as generated and let the student review them.

## Local scope

This version is a single-user localhost prototype with no authentication. Do not expose it publicly or store shared student records until user identity and per-user access are implemented. Default database credentials are local development credentials. The `.env.example` file documents settings; the Go application reads environment variables, not `.env` files automatically.

## Stage 2: Switch and Mattermost

`integrations/switch.compose.yaml` is the upstream standalone Compose file from
https://github.com/sandbox-quantum/switch. Its `collab` profile includes Mattermost
and the setup service; `gateway` adds the administration UI. The local generated
`integrations/.env` contains credentials and is ignored by Git. Keep it private.

```bash
# On a fresh checkout only (refuses to overwrite existing credentials):
go run ./cmd/chat-env
docker compose --env-file integrations/.env -p memorybox-chat \
  -f integrations/switch.compose.yaml --profile collab --profile gateway up -d
```

Local ports: Mattermost `http://localhost:58065`, Switch bridge `http://localhost:58000`,
Gateway `http://localhost:53000`. Credentials are in `integrations/.env`.
The initial development stack explicitly uses the upstream `latest` image tag;
pin images to release tags or digests before sharing a reproducible submission.

Create/register a custom always-on agent named `memorybox`, retain its agent ID
and API key, and add it to a Mattermost study channel through Switch. Set these
variables when running the Go app:

```bash
export SWITCH_URL=http://localhost:58000
export SWITCH_AGENT_ID='<registered-agent-id>'
export SWITCH_API_KEY='<agent-api-key>'
export SWITCH_ROOM_ID='<study-room-id>'
go run .
```

Address the agent in Mattermost and include one of:

```text
!note save Biology | Cell respiration | Mitochondria release energy.
!note search mitochondria
!note summarize 1
!note export markdown
!note export json
!note help
```

This prototype uses one **shared notebook for one trusted study room**. All room
members can search and export that notebook. It is not a private per-student vault.
The adapter opens an authenticated SSE connection scoped to the configured room,
maintains the heartbeat, and posts replies through Switch's message endpoint.
It stops on a history gap so an operator can inspect missed context. Cursors are
in memory; delivery is not exactly-once across crashes, and a retried save may
create another note. Large exports are downloaded through the local HTTP API.

Protocol reference: https://docs.switchagents.ai/flintai/switch/internals/agent-protocol
Mattermost setup: https://docs.switchagents.ai/flintai/switch/deploy/messaging-apps/mattermost

## Stage 3: optional summarization

Run an Ollama server and install a model that fits your machine. Then set:

```bash
export OLLAMA_URL=http://localhost:11434
export OLLAMA_MODEL='<installed-model-name>'
go run .
```

Summarize an existing note:

```bash
curl -X POST http://localhost:8080/notes/1/summarize
```

The generated summary is stored separately; the original content is preserved.
Saving, searching and exporting do not call an AI model. No model or API key is
bundled. Provider tests use a local stub and do not establish model quality.

Verification completed: real PostgreSQL save/search/filter/export checks, summary
persistence with original text preserved, simulated Switch SSE/message exchange,
provider request/response checks, race detection, and `go vet`. A real model and
the live Switch/Mattermost workflow require the services and credentials above.
Ollama API: https://docs.ollama.com/api/chat

Custom registration request (API key is returned only once):

```bash
# Get a registration token for your Gateway user using Switch's agent onboarding.
export SWITCH_REGISTRATION_TOKEN='<your-registration-token>'
curl -fsS http://localhost:58000/agents \
  -H "Authorization: Bearer $SWITCH_REGISTRATION_TOKEN" \
  -H 'Content-Type: application/json' \
  --data-binary @integrations/register-agent.json
```

Keep the returned `id` and `api_key` private. Re-registering an existing name
returns a conflict; the example deliberately does not overwrite an existing agent.
