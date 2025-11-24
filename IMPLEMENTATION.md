# koria-core Implementation Summary

## Overview
This repository implements a custom xray-core proxy that disguises network traffic as Minecraft game packets for censorship circumvention purposes.

## Architecture

### Core Components

1. **Minecraft Protocol Layer** (`pkg/minecraft`)
   - VarInt encoding/decoding (Minecraft's variable-length integer format)
   - Packet structure: Length prefix + Packet ID + Payload
   - Support for common Minecraft packet types
   - String encoding/decoding utilities

2. **Proxy Layer** (`pkg/proxy`)
   - TCP proxy server with bidirectional traffic handling
   - Automatic packet wrapping (client → upstream)
   - Automatic packet unwrapping (upstream → client)
   - Connection timeout management
   - Graceful error handling

3. **Configuration System** (`pkg/config`)
   - JSON-based configuration
   - CLI flag override support
   - Validation and default values
   - Duration parsing support

4. **CLI Application** (`cmd/koria-core`)
   - Command-line interface
   - Version information
   - Config file generation
   - Signal handling for graceful shutdown

## How It Works

### Traffic Flow

```
Client ──► [Raw Data] ──► Koria-Core ──► [Minecraft Packets] ──► Upstream
       ◄── [Raw Data] ◄──            ◄── [Minecraft Packets] ◄──
```

### Packet Wrapping

1. Client sends raw TCP data
2. Proxy reads data into buffer (32KB chunks)
3. Each chunk is wrapped as a Minecraft Custom Payload packet:
   - VarInt length prefix
   - Packet ID (0x17 for Custom Payload)
   - Raw data as payload
4. Wrapped packet is sent to upstream

### Packet Unwrapping

1. Upstream sends Minecraft-wrapped data
2. Proxy reads VarInt length
3. Proxy reads Packet ID
4. Proxy reads payload data
5. Raw payload is forwarded to client

## Implementation Details

### Minecraft Protocol Compliance

The implementation follows Minecraft's protocol specification:

- **VarInt Format**: 7 bits data per byte, MSB as continuation bit
- **Packet Structure**: Standard Minecraft packet format
- **Protocol Version**: Compatible with version 760 (Minecraft 1.19.2)

### Security Considerations

- Traffic appears as Minecraft game data
- Compatible with standard Minecraft ports (25565)
- Can use standard Minecraft server addresses
- Protocol-level obfuscation only (recommend combining with TLS)

## Testing

### Test Coverage

- Minecraft protocol: 74.6% coverage
  - VarInt encoding/decoding
  - Packet encoding/decoding
  - String encoding/decoding
  - Handshake and keep-alive packet creation

- Configuration: 85.7% coverage
  - Default configuration
  - Validation
  - File I/O
  - JSON marshaling/unmarshaling
  - Duration parsing

### Test Categories

1. **Unit Tests**: Core functionality of each package
2. **Round-trip Tests**: Encode/decode cycles
3. **Edge Cases**: Empty data, large packets, invalid inputs
4. **Benchmarks**: Performance testing for encode/decode

## Performance Characteristics

- Buffer size: 32KB (configurable)
- Packet overhead: ~2-5 bytes per packet (VarInt length + packet ID)
- Memory usage: Minimal (streaming, no buffering of complete streams)
- Latency: Near-native TCP (single encode/decode per buffer)

## Limitations

1. **Protocol Fingerprinting**: Deep packet inspection may detect deviations from real Minecraft traffic
2. **Timing Analysis**: Packet timing patterns may differ from real gameplay
3. **Single Protocol State**: Implementation uses play state packets only
4. **No Compression**: Real Minecraft uses compression for packets > 256 bytes

## Future Enhancements

Potential improvements (not implemented):

1. Packet compression (Minecraft uses zlib)
2. Multiple protocol state simulation
3. Fake keep-alive packets to maintain connection appearance
4. Random padding for packet size variation
5. Timing jitter to mimic real gameplay
6. Server list ping response simulation

## Usage

### Basic Example

```bash
# Start proxy
./koria-core -listen 127.0.0.1:8080 -upstream example.com:443

# Client connects to localhost:8080
# Traffic is wrapped in Minecraft packets
# Upstream sees Minecraft-looking traffic
```

### With Configuration

```bash
# Generate config
./koria-core -generate-config config.json

# Edit config.json as needed

# Start proxy
./koria-core -config config.json
```

## Files

- `pkg/minecraft/packet.go`: Minecraft protocol implementation
- `pkg/minecraft/packet_test.go`: Protocol tests
- `pkg/proxy/handler.go`: Proxy server implementation
- `pkg/config/config.go`: Configuration management
- `pkg/config/config_test.go`: Configuration tests
- `cmd/koria-core/main.go`: CLI entry point
- `README.md`: User documentation
- `EXAMPLES.md`: Usage examples
- `IMPLEMENTATION.md`: This file

## Technical Specifications

- Language: Go 1.21+
- Protocol: Minecraft Java Edition protocol (version 760)
- Transport: TCP
- Encoding: Big-endian (network byte order)
- String Format: UTF-8

## Dependencies

Standard library only:
- `bytes`: Buffer manipulation
- `context`: Cancellation and timeouts
- `encoding/binary`: Binary encoding
- `encoding/json`: Configuration parsing
- `flag`: CLI arguments
- `fmt`: Formatting
- `io`: I/O operations
- `log`: Logging
- `net`: Network operations
- `os`: File system
- `sync`: Synchronization
- `time`: Timeouts and durations

Zero external dependencies!

## Code Quality

- ✅ All tests passing
- ✅ No `go vet` warnings
- ✅ Code formatted with `gofmt`
- ✅ No CodeQL security alerts
- ✅ Code review feedback addressed
- ✅ Comprehensive documentation

## License

MIT License - see LICENSE file for details.
