# Flip the Switch Hackathon submission draft

**Status: draft — not submitted**

## Project

**MemoryBox — a study-room memory agent**

**Track:** Best Personal or Creative Workflow

**Tagline:** Turn your study chat into searchable, reusable class notes.

## Problem

Students share explanations, reminders, and useful links in group chats, then lose them in the message stream. MemoryBox gives a study group one place to save, search, summarize, and export its shared notes.

## What it does

MemoryBox is a Go and PostgreSQL note service with an agent adapter for Switch. From a connected Mattermost study room, a student can ask the MemoryBox agent to save a note, search the group's notebook, summarize a saved note, or export the notebook. Switch carries the requests and replies between people and the agent; the web app also provides a visual notebook.

## What makes the workflow useful

The key workflow is shared recall: a student asks in the room for a concept or saved explanation, and the agent retrieves it from the group's notes so the conversation can continue with the same context. Original note text is retained when summaries are generated.

## Technologies

Go, Switch agent protocol, Mattermost, PostgreSQL, and OpenRouter or Ollama for optional summaries.

## Links to add before submitting

- Public source repository: **not available yet**
- Demo video: **not recorded yet**
- Live demo: https://memorybox.rhysch.fun

## Submission readiness checklist

- [ ] Publish the source to a public GitHub repository and add its URL above.
- [ ] Configure a real Switch agent, API key, and Mattermost room; verify save, search, and summary replies in that room.
- [ ] Record a short demo showing a person and the agent completing the workflow through Switch.
- [ ] Update the README with exact Switch setup and demo steps, and ensure it describes the live-tested behavior accurately.
- [ ] Submit the repository, demo video, and README at https://www.wemakedevs.org/hackathons/switch/submit before Nov 7, 2026, 11:59 PM IST.

## Important project limitation

The hosted web app currently uses a shared notebook and has no user authentication. Do not put private or sensitive student information in it. The Switch adapter is present in the code, but it is not configured against a live Switch/Mattermost room yet; the project must not be described as having a live Switch demo until that is tested.
