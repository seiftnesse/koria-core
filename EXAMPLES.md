# Examples

This document provides practical examples of using koria-core.

## Basic Usage Examples

### Example 1: Simple Local Proxy

Start a proxy that listens on port 8080 and forwards to port 8081:

```bash
./koria-core -listen 127.0.0.1:8080 -upstream 127.0.0.1:8081
```

### Example 2: Using Configuration File

Create a configuration file:

```bash
./koria-core -generate-config my-config.json
```

Edit `my-config.json`:

```json
{
  "timeout": "60s",
  "listen_address": "0.0.0.0:8080",
  "upstream_address": "example.com:443",
  "minecraft_server": "mc.example.com",
  "minecraft_port": 25565,
  "log_level": "info"
}
```

Start the proxy:

```bash
./koria-core -config my-config.json
```

### Example 3: Testing with netcat

Terminal 1 - Start upstream echo server:
```bash
nc -l 8081
```

Terminal 2 - Start koria-core proxy:
```bash
./koria-core -listen 127.0.0.1:8080 -upstream 127.0.0.1:8081
```

Terminal 3 - Connect client:
```bash
nc localhost 8080
```

Type messages in Terminal 3, they will appear in Terminal 1 (wrapped in Minecraft packets).

## Advanced Examples

### Example 4: Public-Facing Proxy

```json
{
  "listen_address": "0.0.0.0:25565",
  "upstream_address": "internal-service:8080",
  "timeout": "2m",
  "minecraft_server": "play.example.com",
  "minecraft_port": 25565,
  "log_level": "info"
}
```

This configuration:
- Listens on the standard Minecraft port (25565)
- Forwards to an internal service
- Uses longer timeout for slow connections
- Appears as a Minecraft server to external observers

### Example 5: Chain Multiple Proxies

Setup:
1. Proxy A: Listen 8080 → Forward to Proxy B (8081)
2. Proxy B: Listen 8081 → Forward to Final Server (8082)

Proxy A config:
```json
{
  "listen_address": "127.0.0.1:8080",
  "upstream_address": "127.0.0.1:8081",
  "timeout": "30s"
}
```

Proxy B config:
```json
{
  "listen_address": "127.0.0.1:8081",
  "upstream_address": "127.0.0.1:8082",
  "timeout": "30s"
}
```

This creates double-wrapped Minecraft packets for enhanced obfuscation.

### Example 6: Using with SSH

Forward SSH connections through the Minecraft-disguised proxy:

Server side:
```bash
./koria-core -listen 0.0.0.0:8080 -upstream localhost:22
```

Client side (using SSH ProxyCommand):
```bash
ssh -o "ProxyCommand=nc proxy-server.com 8080" user@proxy-server.com
```

Or create an SSH config (`~/.ssh/config`):
```
Host minecraft-proxy
    HostName proxy-server.com
    User myuser
    ProxyCommand nc proxy-server.com 8080
```

Connect:
```bash
ssh minecraft-proxy
```

### Example 7: Docker Deployment

Dockerfile:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o koria-core ./cmd/koria-core

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/koria-core .
COPY config.json .
EXPOSE 8080
CMD ["./koria-core", "-config", "config.json"]
```

Build and run:
```bash
docker build -t koria-core .
docker run -p 8080:8080 -v $(pwd)/config.json:/root/config.json koria-core
```

### Example 8: SystemD Service

Create `/etc/systemd/system/koria-core.service`:

```ini
[Unit]
Description=Koria-Core Minecraft Proxy
After=network.target

[Service]
Type=simple
User=koria
WorkingDirectory=/opt/koria-core
ExecStart=/opt/koria-core/koria-core -config /opt/koria-core/config.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable koria-core
sudo systemctl start koria-core
sudo systemctl status koria-core
```

## Testing and Debugging

### Test Packet Encoding

You can test the Minecraft packet encoding by examining the traffic:

```bash
# Start proxy
./koria-core -listen 127.0.0.1:8080 -upstream 127.0.0.1:8081

# In another terminal, capture traffic
tcpdump -i lo -X port 8080

# Send test data
echo "Hello" | nc localhost 8080
```

The captured traffic will show Minecraft protocol patterns (VarInt length prefixes, etc.).

### Verify Protocol Appearance

Check that the proxy generates legitimate-looking Minecraft traffic:

```bash
# Analyze packet structure
tshark -i lo -f "tcp port 8080" -Y "tcp" -T fields -e tcp.payload
```

You should see:
1. VarInt length prefixes
2. Packet IDs consistent with Minecraft protocol
3. Proper packet framing

## Performance Tuning

### Adjust Buffer Sizes

Modify `pkg/proxy/handler.go` to change buffer size:

```go
buf := make([]byte, 64*1024) // 64KB instead of 32KB
```

### Optimize for Latency vs Throughput

For low-latency applications:
```json
{
  "timeout": "5s"
}
```

For high-throughput applications:
```json
{
  "timeout": "2m"
}
```

## Security Considerations

### Use with TLS

Combine koria-core with stunnel or nginx for encryption:

```bash
# Upstream TLS termination
./koria-core -listen 127.0.0.1:8080 -upstream 127.0.0.1:8443
```

Where port 8443 is handled by:
```bash
stunnel stunnel.conf
```

stunnel.conf:
```ini
[tls-tunnel]
client = yes
accept = 127.0.0.1:8443
connect = target-server.com:443
```

### Firewall Configuration

Allow only Minecraft port:
```bash
sudo ufw allow 25565/tcp comment "Minecraft/Koria-Core"
sudo ufw enable
```

This makes the service appear as a legitimate Minecraft server.

## Troubleshooting

### Connection Timeouts

If you see timeout errors, increase the timeout in config:
```json
{
  "timeout": "120s"
}
```

### Port Already in Use

Check what's using the port:
```bash
sudo lsof -i :8080
```

Change the listen port or stop the conflicting service.

### Testing Connectivity

Test basic connectivity:
```bash
# Check if proxy is listening
nc -zv localhost 8080

# Send test data
echo "test" | nc localhost 8080
```

## Common Issues

### Issue: "Failed to connect to upstream"

**Solution**: Ensure the upstream server is running and accessible.

```bash
# Test upstream directly
nc -zv upstream-host upstream-port
```

### Issue: "Address already in use"

**Solution**: Change the listen address or kill the process using the port.

```bash
# Find process
sudo lsof -i :8080
# Kill it
kill -9 <PID>
```

### Issue: Connection drops immediately

**Solution**: Check that both ends are using compatible protocols. The upstream server should be able to handle raw TCP data.

## Monitoring

### Log Analysis

Monitor proxy activity:
```bash
./koria-core -config config.json 2>&1 | tee koria-core.log
```

### Connection Statistics

Use netstat to monitor connections:
```bash
watch -n 1 'netstat -an | grep 8080'
```

### Performance Monitoring

Monitor with htop:
```bash
htop -p $(pidof koria-core)
```
