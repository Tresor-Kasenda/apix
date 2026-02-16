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

# Use verbose mode to see headers
apix get /users -v

# Add custom headers and query params
apix get /search -q "term=golang" -H "X-Custom:value"

# Use variables
apix get /users/${USER_ID} -V "USER_ID=42"
```

## Project Configuration

Run `apix init` to create an `apix.yaml` in your project:

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
  header_format: "Bearer ${TOKEN}"
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
apix env show dev
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

# Replay it later
apix run login
apix run login -v  # with verbose headers
```

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

## Command Reference

| Command                  | Description                        |
|--------------------------|------------------------------------|
| `apix init`              | Initialize a new project           |
| `apix get <path>`        | Send GET request                   |
| `apix post <path>`       | Send POST request                  |
| `apix put <path>`        | Send PUT request                   |
| `apix patch <path>`      | Send PATCH request                 |
| `apix delete <path>`     | Send DELETE request                |
| `apix env use <name>`    | Switch environment                 |
| `apix env list`          | List environments                  |
| `apix env show <name>`   | Show environment config            |
| `apix env create <name>` | Create new environment             |
| `apix save <name>`       | Save last request                  |
| `apix run <name>`        | Run saved request                  |

### Common Flags

| Flag              | Short | Description                     |
|-------------------|-------|---------------------------------|
| `--header`        | `-H`  | Add header (key:value)          |
| `--query`         | `-q`  | Add query param (key=value)     |
| `--var`           | `-V`  | Set variable (key=value)        |
| `--data`          | `-d`  | Request body (JSON string)      |
| `--file`          | `-f`  | Request body from file          |
| `--verbose`       | `-v`  | Show response headers           |
| `--raw`           |       | Print raw response body         |

## Cross-Platform Build

```bash
make build-all
```

Produces binaries for Linux, macOS, and Windows (amd64 + arm64).

## License

MIT
