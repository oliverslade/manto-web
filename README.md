# Manto

Manto means a type of cloak or blanket in the Galician language.

A privacy-first web client for chatting with LLMs using your own API key.
No accounts. No server-side storage. No telemetry.

## Why this exists

Most AI chat apps either keep your data, require accounts, or hide how requests are handled. Manto is the opposite: a minimal relay + simple UI that lets you talk to a model without giving any of your data to a third party.

## Why API-direct matters for privacy

When you use provider apps (like Claude.ai, ChatGPT web interface, etc.), your conversations are typically:

- Stored on their servers indefinitely
- Used to improve models and services
- Potentially analyzed for user profiling
- Subject to future policy changes around ads or data use

**With Manto + direct API access:**

- Your conversations never touch Manto servers—they go directly from your browser to the AI provider
- API calls are typically not stored long-term by providers (unlike web app conversations)
- No conversation history builds up in anyone's system
- You maintain full control over your data, it lives only on your machine

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

1. You enter your API key in the browser (never stored on the server).

2. The browser fetches available models from `/api/models` using your API key.

3. When you send a message, the browser posts to `/api/messages` with your prompt and selected model.

4. The server forwards your request to the AI provider (e.g., Anthropic) and returns the response.

5. The server is stateless with no conversation logging or data retention.

## What this doesn't do

- No storage of prompts, responses, or keys on servers.

- No selling or sharing of data—there isn't any to sell.

- No third-party scripts or analytics.

## Roadmap (lightweight)

- Enable free use of Haiku 3.5

- Add more providers (toggle in UI).

- Optional stronger local key storage (encryption/passphrase).

- Self-hosted model option (private gateway) as a future possibility.

---

**Use at your own risk.** If you choose to store a key in the browser, do it only on a trusted, non-shared device.

## Technical Details

### Running the Application

```bash
# Build
go build -o manto-web .

# Run
./manto-web

# Or run directly
go run .
```

The application will be available at `http://localhost:8080/`

### Building from Source

Requirements:

- Go 1.21 or later

```bash
git clone <repository>
cd manto-web
go mod download
go build -o manto-web ./cmd/manto-web
```

### Endpoints

- `GET /` - Homepage
- `GET /config.js` - Client configuration
- `GET /api/models` - Get available models (requires API key)
- `POST /api/messages` - Send message to AI (requires API key)
- `GET /healthz` - Health check (returns 204)

### Configuration

Manto works out-of-the-box with sensible defaults. For custom configuration, copy `env.example` to `.env` and modify as needed:

```bash
cp env.example .env
# Edit .env with your preferred settings
```

See `env.example` for all available configuration options including server settings, logging, API configuration, and security settings.
