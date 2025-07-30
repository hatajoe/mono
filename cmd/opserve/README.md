# opserve

A HTTP server that provides API access to 1Password CLI operations. Designed to allow Linux VMs to access 1Password secrets from a macOS host without requiring authentication on each request.

## Features

- HTTP API wrapper around 1Password CLI
- Automatic authentication handling with `op signin`
- Session caching (30 minutes) to avoid repeated authentication prompts
- Support for getting items, reading secret references, and listing items

## Prerequisites

- 1Password CLI (`op`) installed on the host system
- 1Password account configured

## Installation

```bash
go run main.go
```

## Configuration

Set the `PORT` environment variable to change the default port (8080):

```bash
PORT=8081 go run main.go
```

## API Endpoints

### GET /item
Get a specific 1Password item.

**Query Parameters:**
- `name` (required): Name or ID of the item
- `vault` (optional): Vault name
- `field` (optional): Specific field to retrieve

**Examples:**
```bash
curl "http://localhost:8080/item?name=github.com"
curl "http://localhost:8080/item?name=github.com&field=token"
curl "http://localhost:8080/item?name=MyItem&vault=Private&field=password"
```

### GET /read
Read a secret reference.

**Query Parameters:**
- `ref` (required): Secret reference (e.g., `op://Private/Item/field`)

**Example:**
```bash
curl "http://localhost:8080/read?ref=op://Private/github.com/token"
```

### GET /list
List 1Password items.

**Query Parameters:**
- `vault` (optional): Vault name to filter by
- `category` (optional): Category to filter by

**Examples:**
```bash
curl "http://localhost:8080/list"
curl "http://localhost:8080/list?vault=Private"
curl "http://localhost:8080/list?category=login"
```

### GET /health
Check server health and authentication status.

**Example:**
```bash
curl "http://localhost:8080/health"
```

### GET /whoami
Get current authenticated user.

**Example:**
```bash
curl "http://localhost:8080/whoami"
```

## Response Format

**Success Response:**
```json
{
  "data": "response_value"
}
```

**Error Response:**
```json
{
  "error": "error_message"
}
```

## Usage from Linux VM

If running the server on macOS and accessing from a Linux VM:

```bash
# Using Docker's host.docker.internal
curl "http://host.docker.internal:8081/item?name=github.com&field=token"

# Using direct IP
curl "http://192.168.1.100:8081/item?name=github.com&field=token"
```

## Environment Variables

Replace direct `op` CLI calls in your environment:

**Before:**
```bash
export TOKEN=$(op item get github.com --fields label=token --format json | jq '.value' | sed "s/\"//g")
```

**After:**
```bash
export TOKEN=$(curl -s "http://localhost:8081/item?name=github.com&field=token" | jq -r '.data')
```

## Authentication

The server automatically handles 1Password authentication:

1. Checks if current session is valid (cached for 30 minutes)
2. If not authenticated, runs `op signin` automatically
3. Verifies authentication was successful

Make sure you're initially signed in to 1Password on the host system.

## Security Notes

- This server provides access to your 1Password vault
- Only run on trusted networks
- Consider implementing authentication/authorization for production use
- The server reveals secrets using the `--reveal` flag automatically