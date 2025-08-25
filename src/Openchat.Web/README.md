# Openchat

A privacy-first web client for chatting with LLMs using your own API key.
No accounts. No server-side storage. No telemetry.

## Why this exists

Most AI chat apps either keep your data, require accounts, or hide how requests are handled. Openchat is the opposite: a minimal relay + simple UI that lets you talk to a model without giving any of your data to a third party.

## Why API-direct matters for privacy

When you use provider apps (like Claude.ai, ChatGPT web interface, etc.), your conversations are typically:

- Stored on their servers indefinitely
- Used to improve models and services
- Potentially analyzed for user profiling
- Subject to future policy changes around ads or data use

**With Openchat + direct API access:**

- Your conversations never touch Openchat servers—they go directly from your browser to the AI provider
- API calls are typically not stored long-term by providers (unlike web app conversations)
- No conversation history builds up in anyone's system
- You maintain full control over your data—it lives only on your machine

## Core principles

**Bring-your-own key**
You paste an API key from a provider (e.g., Anthropic). It's never stored on the server.

**No accounts, no tracking**
No sign-in, databases, analytics, or cookies. Requests aren't logged.

**Local by default**
If you don't opt in to "store locally," the key lives only in the browser's memory and disappears when the tab closes.
If you do opt in, it's saved plain in your browser storage (don't use this on shared machines).

**Minimal surface area**
One stateless relay endpoint, static pages, and security headers. Fewer moving parts; fewer failure/leak paths.

**User control over model choice**
Start with Anthropic (default model: claude-3-5-haiku-latest). You can change the model, and more providers will be added incrementally.

## How it works (at a glance)

1. The browser sends your prompt + provider choice to /api/relay with your API key in a header.

2. The relay forwards the request to the provider and streams the model's response back to the page.

3. The relay is stateless and configured with no caching and no body logging.

## What this doesn't do

- No storage of prompts, responses, or keys on servers.

- No selling or sharing of data—there isn't any to sell.

- No third-party scripts or analytics.

## Roadmap (lightweight)

- Add more providers (toggle in UI).

- Optional stronger local key storage (encryption/passphrase).

- Self-hosted model option (private gateway) as a future possibility.

---

**Use at your own risk.** If you choose to store a key in the browser, do it only on a trusted, non-shared device.

## Technical Details

### Running the Application

```bash
dotnet run
```

The application will be available at `http://localhost:8080/`

### Endpoints

- `GET /` - Homepage
- `GET /healthz` - Health check (returns 204)

### Environment Configuration

The application binds to port 8080 by default. You can override this with:

```bash
export ASPNETCORE_URLS=http://+:8080
dotnet run
```
