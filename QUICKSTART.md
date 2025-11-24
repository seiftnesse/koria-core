# Quick Start Guide

Get started with koria-core in 5 minutes.

## Installation

### Option 1: Build from Source

```bash
git clone https://github.com/seiftnesse/koria-core.git
cd koria-core
go build -o koria-core ./cmd/koria-core
```

### Option 2: Using Go Install

```bash
go install github.com/seiftnesse/koria-core/cmd/koria-core@latest
```

## Basic Usage

### 1. Check Installation

```bash
./koria-core -version
```

Output:
```
koria-core version 0.1.0
Custom xray-core implementation with Minecraft packet camouflage
```

### 2. Generate Configuration

```bash
./koria-core -generate-config config.json
```

This creates a `config.json` file with default settings.

### 3. Edit Configuration

Edit `config.json` to set your upstream server:

```json
{
  "timeout": "30s",
  "listen_address": "127.0.0.1:8080",
  "upstream_address": "example.com:443",
  "minecraft_server": "localhost",
  "minecraft_port": 25565,
  "log_level": "info"
}
```

### 4. Start the Proxy

```bash
./koria-core -config config.json
```

You should see:
```
╦╔═╔═╗╦═╗╦╔═╗  ╔═╗╔═╗╦═╗╔═╗
╠╩╗║ ║╠╦╝║╠═╣  ║  ║ ║╠╦╝║╣ 
╩ ╩╚═╝╩╚═╩╩ ╩  ╚═╝╚═╝╩╚═╚═╝
Custom Xray-core with Minecraft Packet Camouflage
Version: 0.1.0

Configuration:
  Listen Address:   127.0.0.1:8080
  Upstream Address: example.com:443
  Timeout:          30s
  Minecraft Server: localhost:25565

Koria-core proxy server listening on 127.0.0.1:8080
```

### 5. Test the Connection

In another terminal, test the proxy:

```bash
curl -x http://localhost:8080 https://example.com
```

Or use netcat:

```bash
echo "Hello" | nc localhost 8080
```

## Common Use Cases

### Use Case 1: Local Testing

Start two terminals:

**Terminal 1** - Start an echo server:
```bash
nc -l 8081
```

**Terminal 2** - Start the proxy:
```bash
./koria-core -listen 127.0.0.1:8080 -upstream 127.0.0.1:8081
```

**Terminal 3** - Connect and test:
```bash
nc localhost 8080
# Type messages and press Enter - they'll appear in Terminal 1
```

### Use Case 2: Remote Proxy

```bash
./koria-core -listen 0.0.0.0:8080 -upstream remote-server.com:443
```

Connect from another machine:
```bash
curl -x http://your-server-ip:8080 https://example.com
```

### Use Case 3: Appearing as Minecraft Server

Use the standard Minecraft port:

```bash
./koria-core -listen 0.0.0.0:25565 -upstream backend-service:8080
```

To external observers, this looks like a Minecraft server on port 25565.

## Command-Line Options

```
Usage: koria-core [options]

Options:
  -config string
        Path to configuration file
  -listen string
        Listen address (default "127.0.0.1:8080")
  -upstream string
        Upstream server address (required if no config file)
  -version
        Show version information
  -generate-config string
        Generate default config file at specified path
```

## Quick Configuration Examples

### Minimal Config

```json
{
  "listen_address": "127.0.0.1:8080",
  "upstream_address": "example.com:443",
  "timeout": "30s"
}
```

### High-Performance Config

```json
{
  "listen_address": "0.0.0.0:8080",
  "upstream_address": "fast-server.com:443",
  "timeout": "2m",
  "log_level": "warn"
}
```

### Public Minecraft Appearance

```json
{
  "listen_address": "0.0.0.0:25565",
  "upstream_address": "hidden-service:8080",
  "timeout": "1m",
  "minecraft_server": "play.example.com",
  "minecraft_port": 25565
}
```

## Troubleshooting

### "Address already in use"

Someone else is using the port. Try a different port:

```bash
./koria-core -listen 127.0.0.1:8090 -upstream example.com:443
```

### "Failed to connect to upstream"

Check that the upstream server is reachable:

```bash
nc -zv upstream-host upstream-port
```

### "Connection timeout"

Increase the timeout in your config:

```json
{
  "timeout": "120s"
}
```

### No Output

Check if the proxy is running:

```bash
ps aux | grep koria-core
```

Check if it's listening:

```bash
netstat -an | grep 8080
```

## Next Steps

- Read [EXAMPLES.md](EXAMPLES.md) for more advanced usage
- Check [README.md](README.md) for detailed documentation
- Review [IMPLEMENTATION.md](IMPLEMENTATION.md) for technical details

## Tips

1. **Start Simple**: Test locally before deploying remotely
2. **Check Logs**: The proxy logs all connections and data transfers
3. **Use Configuration Files**: Easier to manage than command-line flags
4. **Test with netcat**: Simple way to verify the proxy works
5. **Monitor Resources**: Check CPU/memory usage under load

## Security Notes

⚠️ **Important Security Considerations:**

1. This tool provides **protocol-level obfuscation only**
2. Always use **TLS/SSL** for the upstream connection
3. Consider using with **VPN** or **Tor** for additional security
4. Be aware of **timing and traffic analysis** risks
5. Test in a **safe environment** first

## Getting Help

- Issues: https://github.com/seiftnesse/koria-core/issues
- Documentation: See README.md, EXAMPLES.md, IMPLEMENTATION.md
- Code: Browse the source code in pkg/ directory

## Monitoring the Proxy

### View Active Connections

```bash
lsof -i :8080
```

### Monitor Traffic

```bash
tcpdump -i any -n port 8080
```

### Check System Resources

```bash
top -p $(pidof koria-core)
```

## Stopping the Proxy

Press `Ctrl+C` in the terminal running koria-core. It will shut down gracefully:

```
^C
Shutting down gracefully...
```

## Running as a Service

For production use, consider running as a systemd service. See [EXAMPLES.md](EXAMPLES.md) for details.

---

**That's it! You're now running koria-core.** 🎉

For more advanced usage and examples, check out the other documentation files.
