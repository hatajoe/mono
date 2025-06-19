# relaymcp

A lightweight HTTP and Server-Sent Events (SSE) relay proxy written in Go.

## Features

- **HTTP Relay**: Forwards all HTTP requests to a target server while preserving original paths, headers, and query parameters
- **SSE Support**: Automatically detects and properly handles Server-Sent Events connections with streaming support
- **Simple Configuration**: Minimal command-line flags for easy setup
- **Cross-platform**: Builds for multiple architectures including ARM64 for Apple Silicon Macs

## Usage

```bash
./relaymcp [options]
```

### Options

- `-port`: Port to listen on (default: 8080)
- `-target`: Target host:port to relay requests to (default: 127.0.0.1:3845)
- `-help`: Show help message

### Examples

```bash
# Start relay server on port 8080, forwarding to localhost:3845
./relaymcp

# Custom port and target
./relaymcp -port 9000 -target 192.168.1.100:8000

# Show help
./relaymcp -help
```

## How it Works

1. **Request Detection**: Automatically detects SSE requests by checking for `Accept: text/event-stream` header
2. **Path Preservation**: All requests maintain their original paths (e.g., `/api/data` → `target:port/api/data`)
3. **Header Forwarding**: All request and response headers are preserved
4. **Streaming**: SSE connections use line-by-line streaming with immediate flushing for real-time events

## Building

```bash
# For current platform
go build

# For macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o relaymcp-darwin-arm64

# For Linux AMD64
GOOS=linux GOARCH=amd64 go build -o relaymcp-linux-amd64
```

## Use Cases

- Development proxy for MCP (Model Context Protocol) servers
- SSE event streaming relay
- Simple HTTP reverse proxy
- Local development server relay

## License

See the main repository LICENSE file.