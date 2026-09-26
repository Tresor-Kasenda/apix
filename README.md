# apix

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo/apix-logo-dark.svg">
    <img src="assets/logo/apix-logo.svg" alt="apix logo" width="560">
  </picture>
</p>

A modern, framework-agnostic CLI API tester for terminal-first developers.
Replaces Postman, curl, and httpie with a simple, powerful workflow.

<p align="center">
  <a href="https://github.com/Tresor-Kasenda/apix/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/Tresor-Kasenda/apix"></a>
  <a href="https://github.com/Tresor-Kasenda/apix/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/github/license/Tresor-Kasenda/apix"></a>
  <a href="https://github.com/Tresor-Kasenda/apix/stargazers"><img alt="GitHub stars" src="https://img.shields.io/github/stars/Tresor-Kasenda/apix"></a>
  <a href="https://github.com/Tresor-Kasenda/apix/blob/main/go.mod"><img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/Tresor-Kasenda/apix"></a>
</p>

`apix` is built for developers who prefer terminal + Git workflows, and plugs into
**any** HTTP API project — whatever the language, framework or repository layout:
- Bootstrap from an OpenAPI/Swagger spec or from your source code (`apix init`)
- Reuse your existing `.env` files and CI environment variables (`${VAR}`)
- Run from any sub-directory, monorepos included (like `git`)
- Save API requests as versioned files (`requests/*.yaml`)
- Run chained flows with variable capture (`apix chain`)
- Add assertions and run API checks in CI (`apix test`)
- Watch requests live while editing (`apix watch`)

| Need | curl | apix |
|------|------|------|
| Reuse requests across team | Manual scripts | Versioned request files (`requests/*.yaml`) |
| Multi-step API flows | Ad-hoc shell glue | Native chaining + capture (`apix chain`) |
| API validation in CI | Custom scripting | Built-in assertions (`apix test`) |
| Fast local iteration | Re-run commands manually | Watch mode (`apix watch`) |

If apix saves you time, give the repo a star.

## Installation

### Homebrew (recommended on macOS/Linux)

```bash
brew tap Tresor-Kasenda/homebrew-tap
brew install apix
```

### One-liner install script (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/Tresor-Kasenda/apix/main/install.sh | bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/Tresor-Kasenda/apix/main/install.sh | bash -s -- v0.1.1
```

### Snap (Linux)

```bash
sudo snap install apix
```

### Build from source

```bash
git clone https://github.com/Tresor-Kasenda/apix.git
cd apix
make build
./bin/apix --help
```

## Quick Start (60 seconds)

```bash
# 1) Initialize project structure (apix.yaml, requests/, env/)
apix init

# 2) Send your first request
apix get /users

# 3) Save and replay request workflows
apix save list-users
apix run list-users

# 4) Add assertions and run tests
apix test
```

Then explore:
- `apix import openapi <file-or-url>` to import every endpoint of any API
- `apix chain` for end-to-end API flows
- `apix import` / `apix export` for migration from curl/Postman/Insomnia
- `apix watch` for fast edit-and-rerun loops

## Community

- Report bugs: https://github.com/Tresor-Kasenda/apix/issues/new
- Request features: https://github.com/Tresor-Kasenda/apix/issues/new
- General issues list: https://github.com/Tresor-Kasenda/apix/issues

## Maintainer Release Workflow

```bash
# Cross-compile + package archives + checksums
make dist

# Generate Homebrew formula from dist/checksums.txt
make dist-brew

# Publish formula to Homebrew tap
TAP_REPO=Tresor-Kasenda/homebrew-tap GITHUB_TOKEN=<token> make dist-brew-publish

# Full pipeline (optional snap package)
WITH_BREW_PUBLISH=1 TAP_REPO=Tresor-Kasenda/homebrew-tap GITHUB_TOKEN=<token> WITH_SNAP=1 make dist-all
```

Dry-run tap publication without push:

```bash
TAP_REPO=Tresor-Kasenda/homebrew-tap DRY_RUN=1 make dist-brew-publish
```

## Works With Any Project

apix makes no assumption about your stack. Everything it needs comes from four
universal sources, in this order of preference:

1. **An OpenAPI / Swagger document** — the framework-agnostic route source.
2. **Your `.env` files and OS environment** — for base URLs, ports and secrets.
3. **Source-code detection** — for 30+ frameworks, when no spec exists.
4. **Explicit flags / `apix.yaml`** — always win.

### Project discovery (monorepos & sub-directories)

Like `git`, apix walks up from the current directory until it finds an
`apix.yaml`. `requests/`, `env/` and `.apix/` always resolve against that
project root, so every command works from `backend/src/controllers/` as well as
from the root. Set `APIX_PROJECT_DIR` to force a root.

`apix init` also looks inside conventional monorepo folders (`backend/`, `api/`,
`server/`, `apps/`, `services/`, ...) when nothing is found at the root.

### Detected frameworks

| Language | Frameworks |
|----------|------------|
| PHP | Laravel / Lumen, Symfony, Slim, CakePHP, CodeIgniter, Yii, Laminas / Mezzio |
| Python | Django (+ DRF), FastAPI, Flask |
| JavaScript / TypeScript | NestJS, AdonisJS, Hono, Koa, Express, Fastify |
| Go | Gin, Chi, Echo, Fiber, Gorilla Mux, `net/http` (Go 1.22+ patterns) |
| Java / Kotlin | Spring Boot, Quarkus, Micronaut, Ktor |
| C# | ASP.NET Core (controllers + minimal APIs) |
| Ruby | Rails, Sinatra |
| Rust | Actix, Axum, Rocket |
| Elixir | Phoenix |
| Swift | Vapor |

Not listed? Point apix at your OpenAPI document (`apix init --spec ...` or
`apix import openapi ...`) or simply pass `--base-url`: nothing else is needed.

Detected route parameters are normalized into apix variables:
`/users/{id}`, `/users/:id` and `/users/<int:id>` all become `/users/${id}`.

### Base URL detection

`apix init` proposes a base URL from, in order:

1. `APIX_BASE_URL`, `API_BASE_URL`, `API_URL` or `BASE_URL` in `.env`
2. `APP_URL` in `.env` (+ the framework API prefix, e.g. `/api` for Laravel)
3. `PORT`, `APP_PORT`, `SERVER_PORT`, `HTTP_PORT` or `API_PORT` in `.env`
4. `server.port` in Spring's `application.properties`
5. The framework's default dev port (8000 Django/Laravel, 3000 Node, 8080 Go/Java, 5000 .NET, ...)

### Non-interactive setup (CI, scripts, Docker)

Prompts are skipped with `--yes` or automatically when stdin is not a terminal:

```bash
apix init --yes
apix init --name shop --base-url http://localhost:3000 --auth none -y
apix init --spec http://localhost:8000/openapi.json -y
```

### Runtime overrides

| Environment variable | Effect |
|----------------------|--------|
| `APIX_ENV`           | Selects the environment (like `apix env use`, without editing files) |
| `APIX_BASE_URL`      | Overrides the base URL for every request |
| `APIX_PROJECT_DIR`   | Forces the project root instead of searching upward |

```yaml
# .github/workflows/api.yml (excerpt)
- run: apix test
  env:
    APIX_ENV: ci
    APIX_BASE_URL: http://localhost:8080
    API_TOKEN: ${{ secrets.API_TOKEN }}   # referenced as ${API_TOKEN} in apix.yaml
```

## Extended Quick Start

```bash
# Initialize a project
apix init

# Send requests
apix get /users
apix post /login -d '{"email": "user@example.com", "password": "secret"}'
apix put /users/1 -d '{"name": "Updated Name"}'
apix delete /users/1
apix head /users
apix options /users

# Use verbose mode to see headers
apix get /users -v

# Add custom headers and query params
apix get /search -q "term=golang" -H "X-Custom:value"

# Send form payloads
apix post /upload --form "type=avatar" --form "file=@./photo.jpg"
apix post /login --urlencoded "email=user@example.com" --urlencoded "password=secret"

# Use variables
apix get /users/${USER_ID} -V "USER_ID=42"

# Environment workflow
apix env show
apix env copy dev staging
apix env delete staging --force
```

## Project Configuration

Run `apix init` to create an `apix.yaml` in your project.
The command prompts for (or accepts as flags `--name`, `--base-url`, `--auth`):
- project name (default: current directory name)
- base URL (default: detected, see [Base URL detection](#base-url-detection))
- auth type (`none`, `bearer`, `basic`, `api_key`, `custom`)

It then imports routes from an OpenAPI document (`--spec`, or one found in the
project such as `openapi.yaml`, `docs/swagger.json`, `storage/api-docs/api-docs.json`),
falling back to source-code route detection. `.apix/` is added to `.gitignore`.

Example config:

```yaml
project: my-api
base_url: ${APP_URL:-http://localhost:8000}/api   # variables work here too
timeout: 30
current_env: dev
headers:
  Content-Type: application/json
  Accept: application/json
auth:
  type: bearer
  token_path: access_token|token|data.token   # first match wins
  header_name: Authorization
  header_format: "Bearer ${TOKEN}"
  login_request: login

# Optional: adapt apix to an existing repository layout
requests_dir: tests/http        # default: requests
env_dir: tests/http/env         # default: env
dotenv: [.env, .env.testing]    # default: [.env, .env.local]
```

Editing commands such as `apix env use` preserve your comments and key order.

## Authentication

Supported auth types:
- `none`
- `bearer`
- `basic`
- `api_key`
- `custom`

Credentials support variables, so secrets can stay in `.env` or CI secrets
instead of being committed. A credential whose variable is undefined is never
sent as a literal `${VAR}`.

Example auth config:

```yaml
auth:
  type: bearer
  token_path: data.token
  header_name: Authorization
  header_format: "Bearer ${TOKEN}"
  login_request: login

  # basic
  # username: ${API_USERNAME}
  # password: ${API_PASSWORD}

  # api_key
  # api_key: ${API_KEY}
  # header_name: X-API-Key
  # header_format: "${API_KEY}"

  # custom
  # header_name: X-Custom-Auth
  # header_format: "Token ${TOKEN}"
```

## Environments

Manage different environments (dev, staging, production):

```bash
# Create environments
apix env create staging
apix env create production

# Switch environment
apix env use staging

# List environments
apix env list

# Show environment config
apix env show
apix env show dev

# Copy and delete environments
apix env copy dev staging
apix env delete staging
apix env delete staging --force
```

Environment files live in `env/<name>.yaml` (or `env_dir`). Every key is
optional: a new environment inherits everything from `apix.yaml` until you
override it.

```yaml
base_url: https://staging.api.example.com
headers:
  X-Debug: "true"
variables:
  API_KEY: staging-key-123
```

## Saved Requests

Save and replay requests:

```bash
# Run a request
apix post /login -d '{"email": "test@test.com", "password": "pass"}'

# Save the last request
apix save login
apix save login --from-last

# Replay it later
apix run login
apix run login -v  # with verbose headers
apix run login --env staging

# Manage saved requests
apix list
apix show login
apix rename login auth-login
apix delete auth-login --saved
```

Chain multiple saved requests with captured variables:

```yaml
# requests/login.yaml
name: login
method: POST
path: /login
body: '{"email":"test@test.com","password":"pass"}'
capture:
  TOKEN: data.token
  USER_ID: data.user.id
```

```yaml
# requests/get-profile.yaml
name: get-profile
method: GET
path: /users/${USER_ID}
headers:
  Authorization: "Bearer ${TOKEN}"
```

```bash
apix chain login get-profile
apix chain login get-profile --env staging
apix chain login get-profile -V "TENANT=acme"
```

## Watch Mode + Hooks

Watch a saved request and re-run it on file changes:

```bash
# Event-based watch (fs events)
apix watch login

# Polling fallback (checks changes every 5 seconds)
apix watch login --interval 5s
```

Add declarative hooks in request YAML:

```yaml
# requests/login.yaml
name: login
method: POST
path: /login
body: '{"email":"test@test.com","password":"pass"}'
pre_request:
  - run: bootstrap
    capture:
      SESSION_ID: data.session.id
post_request:
  - run: metrics
capture:
  TOKEN: data.token
```

Hook behavior:
- `pre_request` runs before the main request.
- `post_request` runs after a successful main request.
- Hook failures stop the current iteration with an explicit error message.
- Guardrails prevent recursive hook loops.

## Auto Token Capture

When `auth.token_path` is configured, apix automatically captures tokens from
responses. Separate several candidate paths with `|` to support different API
conventions (`access_token|token|data.token`); array indexes are supported too
(`data.0.token`). After a login request, the token is saved and used in subsequent
requests:

```bash
# This captures the token automatically
apix post /login -d '{"email": "test@test.com", "password": "pass"}'
# Token captured and saved

# Subsequent requests include the Bearer token
apix get /protected/resource
```

When a response returns `401 Unauthorized` and `auth.login_request` is set,
apix automatically runs that saved login request, captures the new token, and
retries the original request once.

## Variables

Use `${VAR}` syntax in the base URL, paths, headers, query params, bodies and
auth credentials. `${VAR:-default}` provides a fallback value.

Resolution order (first match wins):

1. `--var` / `-V` flags and values captured by `chain` / hooks
2. Built-ins: `${TOKEN}`, `${TIMESTAMP}`, `${UUID}`, `${RANDOM}`
3. `variables` of the active environment (`env/<name>.yaml`)
4. `variables` of `apix.yaml`
5. OS environment variables (CI secrets, shell exports)
6. `.env` files of the project (`dotenv` setting)
7. The inline default of `${VAR:-default}`

| Variable      | Description                     |
|---------------|---------------------------------|
| `${VAR}`      | From flags, config, OS environment or `.env` |
| `${VAR:-x}`   | Same, with `x` as fallback      |
| `${TOKEN}`    | Auto-captured auth token        |
| `${TIMESTAMP}`| Current Unix timestamp          |
| `${UUID}`     | Generated UUID v4               |
| `${RANDOM}`   | Random 8-char hex string        |

```bash
apix post /events -d '{"id": "${UUID}", "ts": "${TIMESTAMP}"}'
apix get /users/${USER_ID} -V "USER_ID=42"
apix get '/search?q=${QUERY:-golang}'
```

## Testing With Assertions

Define an `expect` block in request YAML files, then run `apix test`.

```yaml
# requests/get-profile.yaml
name: get-profile
method: GET
path: /users/${USER_ID}
headers:
  Authorization: "Bearer ${TOKEN}"
expect:
  status:
    eq: 200
  body:
    data.user.id:
      is_number: true
    data.user.name:
      contains: "Alice"
  headers:
    Content-Type:
      contains: application/json
  response_time:
    lte: 500
```

Supported operators:
- `exists`
- `eq`
- `contains`
- `is_number`
- `is_string`
- `is_array`
- `is_bool`
- `is_null`
- `gt`
- `gte`
- `lt`
- `lte`
- `length`

```bash
# Run all requests with an expect block in requests/
apix test

# Run one request test
apix test get-profile

# Run tests from a custom directory
apix test --dir tests/
```

## Developer Experience

apix keeps a local request history and can show the effective merged config:

```bash
# Show the 20 most recent request executions
apix history

# Show a custom number of entries
apix history --limit 50

# Clear history
apix history --clear

# Show active merged configuration (apix.yaml + current env)
apix config show
```

History is stored in `.apix/history.jsonl`.
Standard status output now includes response duration and body size.

## Import / Export

Import from external tools/formats:

```bash
# Import every operation of an OpenAPI 3 / Swagger 2 document (JSON or YAML, file or URL)
apix import openapi openapi.yaml
apix import openapi http://localhost:8000/openapi.json      # FastAPI
apix import openapi http://localhost:3000/api-json          # NestJS
apix import openapi http://localhost:8080/v3/api-docs       # Spring
apix import openapi swagger.json --with-base-path           # prefix paths with the spec base path

# Import from Postman collection JSON
apix import postman collection.json

# Import from Insomnia export JSON
apix import insomnia insomnia-export.json

# Import from a curl command
apix import curl "curl -X POST https://api.example.com/login -H 'Content-Type: application/json' -d '{\"email\":\"test@test.com\"}'"
```

Export to external formats:

```bash
# Export one saved request as curl
apix export curl login

# Export all saved requests as Postman collection JSON
apix export postman
apix export postman --output postman-collection.json
```

## Advanced Network

Retry, proxy, TLS, and cookie controls:

```bash
# Retry flaky endpoints (network errors + 5xx)
apix get /unstable --retry 3 --retry-delay 200ms

# Route through a proxy
apix get /users --proxy http://localhost:8080

# Ignore TLS certificate validation (self-signed, local env)
apix get https://self-signed.local -k

# Use client TLS certificate and key
apix get https://mtls.example.com --cert client.crt --key client.key

# Disable persistent cookie jar for one request
apix get /session --no-cookies
```

By default, cookies are persisted between requests in `.apix/cookies.jar`.

## Command Reference

| Command                  | Description                        |
|--------------------------|------------------------------------|
| `apix init`              | Initialize a new project (`--yes`, `--name`, `--base-url`, `--auth`, `--spec`, `--no-detect`) |
| `apix get <path>`        | Send GET request                   |
| `apix post <path>`       | Send POST request                  |
| `apix put <path>`        | Send PUT request                   |
| `apix patch <path>`      | Send PATCH request                 |
| `apix delete <path>`     | Send DELETE request (`--saved` to delete a saved request) |
| `apix head <path>`       | Send HEAD request                  |
| `apix options <path>`    | Send OPTIONS request               |
| `apix env use <name>`    | Switch environment                 |
| `apix env list`          | List environments                  |
| `apix env show [name]`   | Show active or named environment   |
| `apix env create <name>` | Create new environment             |
| `apix env copy <src> <dest>` | Copy an environment            |
| `apix env delete <name>` | Delete an environment              |
| `apix save <name>`       | Save last request                  |
| `apix run <name>`        | Run saved request                  |
| `apix chain <req1> <req2> [...]` | Run saved requests sequentially with variable capture |
| `apix test [name]`       | Run request assertions (`--dir` for custom folder) |
| `apix watch <name>`      | Re-run a saved request on file changes (`--interval` for polling) |
| `apix history`           | Show request execution history (`--limit`, `--clear`) |
| `apix config show`       | Show merged active configuration |
| `apix import openapi <file-or-url>` | Import an OpenAPI 3 / Swagger 2 spec |
| `apix import postman <file>` | Import a Postman collection |
| `apix import insomnia <file>` | Import an Insomnia export |
| `apix import curl "<cmd>"` | Import one curl command |
| `apix export curl <name>` | Export one saved request as curl |
| `apix export postman`    | Export all saved requests as Postman JSON |
| `apix list`              | List saved requests                |
| `apix show <name>`       | Show a saved request YAML          |
| `apix rename <old> <new>`| Rename a saved request             |
| `apix delete <name> --saved` | Delete a saved request         |

### Common Flags

| Flag              | Short | Description                     |
|-------------------|-------|---------------------------------|
| `--header`        | `-H`  | Add header (key:value)          |
| `--query`         | `-q`  | Add query param (key=value)     |
| `--var`           | `-V`  | Set variable (key=value)        |
| `--env`           |       | Use a specific environment for `run`/`chain`/`test`/`watch` only |
| `--interval`      |       | Polling interval for `apix watch` (e.g. `5s`) |
| `--dir`           |       | Use a custom directory for `apix test` |
| `--data`          | `-d`  | Request body (JSON string)      |
| `--file`          | `-f`  | Request body from file          |
| `--form`          |       | Multipart field (key=value or key=@file) |
| `--urlencoded`    |       | URL-encoded field (key=value)   |
| `--verbose`       | `-v`  | Show response headers           |
| `--raw`           |       | Print raw response body         |
| `--headers-only`  |       | Print only status + headers     |
| `--body-only`     |       | Print only response body        |
| `--silent`        | `-s`  | Print only body (script mode)   |
| `--output`        | `-o`  | Write response body to file     |
| `--timeout`       | `-t`  | Override timeout (seconds)      |
| `--no-follow`     |       | Disable redirect following      |
| `--retry`         |       | Retry count on network errors and 5xx |
| `--retry-delay`   |       | Base retry delay (`200ms`, `1s`, ...) |
| `--proxy`         |       | Proxy URL (`http://localhost:8080`) |
| `--insecure`      | `-k`  | Skip TLS certificate validation |
| `--cert`          |       | Client TLS certificate file     |
| `--key`           |       | Client TLS key file             |
| `--no-cookies`    |       | Disable persistent cookie jar   |

## Cross-Platform Build

```bash
make build-all
```

Produces binaries for Linux, macOS, and Windows (amd64 + arm64).

## License

MIT
