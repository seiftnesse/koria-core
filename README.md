# koria-core

Custom implementation of xray-core using Minecraft packet camouflage for censorship circumvention.

## Overview

Koria-core is a network proxy tool that disguises proxy traffic as Minecraft game packets. This implementation provides:

- **Minecraft Protocol Camouflage**: All proxy traffic is wrapped in legitimate-looking Minecraft protocol packets
- **Bidirectional Proxying**: Full-duplex communication with automatic packet encoding/decoding
- **Configurable**: Easy configuration via JSON file or command-line flags
- **Lightweight**: Minimal dependencies and efficient packet handling

## Features

- ✅ Minecraft VarInt protocol encoding/decoding
- ✅ Custom payload packet wrapping for data transmission
- ✅ Fake handshake and keep-alive packets for protocol appearance
- ✅ Bidirectional traffic proxying with automatic packet translation
- ✅ Connection timeout handling
- ✅ Graceful shutdown
- ✅ JSON-based configuration

## Installation

### Build from source

```bash
git clone https://github.com/seiftnesse/koria-core.git
cd koria-core
go build -o koria-core ./cmd/koria-core
```

### Run directly

```bash
go run ./cmd/koria-core/main.go [flags]
```

## Usage

### Command-line flags

```bash
koria-core [options]

Options:
  -config string
        Path to configuration file
  -listen string
        Listen address (default "127.0.0.1:8080")
  -upstream string
        Upstream server address
  -version
        Show version information
  -generate-config string
        Generate default config file at specified path
```

### Generate default configuration

```bash
koria-core -generate-config config.json
```

This creates a `config.json` file with default settings:

```json
{
  "listen_address": "127.0.0.1:8080",
  "upstream_address": "127.0.0.1:8081",
  "timeout": "30s",
  "minecraft_server": "localhost",
  "minecraft_port": 25565,
  "log_level": "info"
}
```

### Start the proxy

```bash
# Using config file
koria-core -config config.json

# Using command-line flags
koria-core -listen 0.0.0.0:8080 -upstream example.com:443

# Using both (flags override config)
koria-core -config config.json -upstream different.com:443
```

## How It Works

### Minecraft Protocol Camouflage

Koria-core uses the Minecraft protocol to disguise proxy traffic:

1. **VarInt Encoding**: Minecraft uses variable-length integers (VarInt) for packet lengths and IDs
2. **Packet Structure**: Each packet consists of:
   - VarInt length prefix
   - VarInt packet ID
   - Packet payload

3. **Custom Payload Packets**: The proxy wraps actual data in Minecraft "Custom Payload" packets (0x17)
4. **Protocol Appearance**: Connections start with fake handshake packets to appear as legitimate Minecraft traffic

### Architecture

```
Client <-> [Koria-core Proxy] <-> Upstream Server
           |                  |
           |  Encode/Decode   |
           |  Minecraft       |
           |  Packets         |
           |__________________|
```

The proxy:
1. Accepts connections on the listen address
2. Establishes connection to upstream server
3. Wraps client→upstream traffic in Minecraft packets
4. Unwraps upstream→client traffic from Minecraft packets
5. Maintains bidirectional communication

## Configuration

### Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `listen_address` | string | Address to listen on | `127.0.0.1:8080` |
| `upstream_address` | string | Upstream server address | `127.0.0.1:8081` |
| `timeout` | duration | Connection timeout | `30s` |
| `minecraft_server` | string | Minecraft server for handshake | `localhost` |
| `minecraft_port` | uint16 | Minecraft server port | `25565` |
| `log_level` | string | Logging level | `info` |

### Duration Format

Timeout can be specified in various formats:
- `30s` - 30 seconds
- `5m` - 5 minutes
- `1h30m` - 1 hour 30 minutes

## Development

### Project Structure

```
koria-core/
├── cmd/
│   └── koria-core/      # Main application entry point
│       └── main.go
├── pkg/
│   ├── minecraft/       # Minecraft protocol implementation
│   │   └── packet.go
│   ├── proxy/           # Proxy server and handler
│   │   └── handler.go
│   └── config/          # Configuration management
│       └── config.go
├── go.mod
└── README.md
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Build
go build ./cmd/koria-core
```

## Protocol Details

### Minecraft Packet Format

```
[VarInt: Length] [VarInt: Packet ID] [Data...]
```

### Supported Packet Types

- `0x00` - Handshake (for initial connection appearance)
- `0x17` - Custom Payload (for data transmission)
- `0x21` - Keep Alive (for connection maintenance)

### VarInt Encoding

Minecraft's VarInt format:
- Each byte has 7 bits of data and 1 continuation bit
- Continues until a byte without continuation bit (0x80) is found
- Maximum 5 bytes for int32

Example:
```
Value: 300
Binary: 10010 1100
VarInt: 10101100 00000010
Bytes:  0xAC 0x02
```

## Security Considerations

This tool is designed for censorship circumvention. Consider:

- **Traffic Analysis**: While packets appear as Minecraft traffic, timing and packet size analysis may still reveal patterns
- **Protocol Fingerprinting**: Deep packet inspection may detect deviations from real Minecraft traffic
- **Use with Encryption**: Combine with TLS/SSL for upstream connections
- **Server Selection**: Choose upstream servers in jurisdictions appropriate for your use case

## License

This project is open source. See LICENSE file for details.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Acknowledgments

- Inspired by XTLS/Xray-core
- Minecraft protocol specification from wiki.vg
- Community feedback and contributions