# apix

A modern, framework-agnostic CLI API tester for terminal-first developers.
Replaces Postman, curl, and httpie with a simple, powerful workflow.

## Installation

### From source

```bash
go install github.com/Tresor-Kasend/apix/cmd/apix@latest
```

### Build locally

```bash
git clone https://github.com/Tresor-Kasend/apix.git
cd apix
make build
./bin/apix --help
```

## Quick Start

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
The command prompts for:
- project name (default: current directory name)
- base URL (default: framework-based suggestion)
- auth type (`none`, `bearer`, `basic`, `api_key`, `custom`)

`apix init` also ensures `.apix/` is present in `.gitignore`.

Example generated config:

```yaml
project: my-api
base_url: http://localhost:8000/api
timeout: 30
current_env: dev
headers:
  Content-Type: application/json
  Accept: application/json
auth:
  type: bearer
  token_path: data.token
  header_name: Authorization
  header_format: "Bearer ${TOKEN}"
  login_request: login
```

## Authentication

Supported auth types:
- `none`
- `bearer`
- `basic`
- `api_key`
- `custom`

Example auth config:

```yaml
auth:
  type: bearer
  token_path: data.token
  header_name: Authorization
  header_format: "Bearer ${TOKEN}"
  login_request: login

  # basic
  # username: my-user
  # password: my-pass

  # api_key
  # api_key: my-api-key
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

Environment files live in `env/<name>.yaml`:

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
responses. After a login request, the token is saved and used in subsequent
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

Use `${VAR}` syntax in URLs, headers, and request bodies:

| Variable      | Description                     |
|---------------|---------------------------------|
| `${VAR}`      | From environment or `--var` flag |
| `${TOKEN}`    | Auto-captured auth token        |
| `${TIMESTAMP}`| Current Unix timestamp          |
| `${UUID}`     | Generated UUID v4               |
| `${RANDOM}`   | Random 8-char hex string        |

```bash
apix post /events -d '{"id": "${UUID}", "ts": "${TIMESTAMP}"}'
apix get /users/${USER_ID} -V "USER_ID=42"
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
| `apix init`              | Initialize a new project           |
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
