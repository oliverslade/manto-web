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

Manto uses a modern `.env` file approach for configuration. The application automatically loads configuration from:

1. `.env.{environment}.local` (e.g., `.env.development.local`)
2. `.env.{environment}` (e.g., `.env.development`)
3. `.env.local`
4. `.env`

Environment is determined by `GO_ENV` or `ENVIRONMENT` variables (defaults to `production`).

#### Environment Variables

**Server Configuration:**

- `PORT` - Server port (default: 8080)
- `HOST` - Server host (default: 0.0.0.0)
- `READ_TIMEOUT` - Server read timeout (default: 30s)
- `WRITE_TIMEOUT` - Server write timeout (default: 30s)

**Logging:**

- `LOG_LEVEL` - Log level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: json, text (default: json)
- `LOG_INCLUDE_TIMESTAMP` - Include timestamps (default: true)
- `LOG_INCLUDE_SOURCE` - Include source code location (default: false)

**Anthropic API:**

- `ANTHROPIC_API_KEY` - Your Anthropic API key (optional, can be provided via UI)
- `ANTHROPIC_BASE_URL` - API base URL (default: https://api.anthropic.com)
- `ANTHROPIC_API_VERSION` - API version (default: 2023-06-01)
- `ANTHROPIC_TIMEOUT` - Request timeout (default: 60s)
- `ANTHROPIC_MAX_TOKENS` - Max tokens per request (default: 1024)
- `ANTHROPIC_TEMPERATURE` - Temperature setting (default: 0.7)

**Security:**

- `ENABLE_HSTS` - Enable HTTPS Strict Transport Security (default: true)
- `ALLOWED_API_ENDPOINTS` - Comma-separated list of allowed API endpoints
- `API_KEY_MIN_LENGTH` - Minimum API key length (default: 10)

**Validation:**

- `MAX_MESSAGE_LENGTH` - Maximum message length (default: 4000)
- `MAX_FILE_SIZE` - Maximum file upload size in bytes (default: 10485760)

#### Example Configuration

Copy `env.example` to `.env` and modify as needed:

```bash
cp env.example .env
# Edit .env with your preferred settings
```

For development:

```bash
# Set development environment
export GO_ENV=development
./manto-web
```

This will automatically load `.env.development` if it exists, falling back to `.env`.
