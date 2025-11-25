# Примеры конфигураций Koria-Core

Эта директория содержит примеры конфигурационных файлов для Koria-Core.

## Файлы

### server.json
Конфигурация для серверной части:
- **Inbound**: Koria протокол на порту 25565
- **Outbound**: Freedom (прямое соединение)
- Поддерживает нескольких клиентов с UUID аутентификацией

### client.json
Конфигурация для клиентской части:
- **Inbounds**:
  - HTTP proxy на 127.0.0.1:8080
  - SOCKS5 proxy на 127.0.0.1:1080
- **Outbounds**:
  - Koria протокол (подключение к серверу)
  - Freedom (для локального трафика)

## Использование

### Запуск сервера
```bash
./koria -config configs/server.json
```

### Запуск клиента
```bash
# Отредактируйте configs/client.json:
# 1. Замените "your-server.com" на адрес вашего сервера
# 2. Убедитесь что userId совпадает с одним из клиентов на сервере

./koria -config configs/client.json
```

### Использование прокси

После запуска клиента:

**HTTP Proxy:**
```bash
# Используйте 127.0.0.1:8080 как HTTP прокси
export http_proxy=http://127.0.0.1:8080
export https_proxy=http://127.0.0.1:8080

curl https://api.ipify.org
```

**SOCKS5 Proxy:**
```bash
# Используйте 127.0.0.1:1080 как SOCKS5 прокси
curl --socks5 127.0.0.1:1080 https://api.ipify.org
```

## Структура конфигурации

```json
{
  "log": {
    "level": "info"  // debug, info, warning, error
  },
  "inbounds": [
    {
      "tag": "unique-tag",
      "protocol": "http|socks|koria",
      "listen": "host:port",
      "settings": { /* protocol-specific */ }
    }
  ],
  "outbounds": [
    {
      "tag": "unique-tag",
      "protocol": "freedom|koria",
      "settings": { /* protocol-specific */ }
    }
  ],
  "routing": {
    "domainStrategy": "AsIs|IPIfNonMatch|IPOnDemand",
    "rules": [
      {
        "type": "field",
        "domain": ["domain-pattern"],
        "ip": ["cidr"],
        "outboundTag": "target-tag"
      }
    ]
  }
}
```

## Генерация UUID

Для создания нового UUID для клиента:

```bash
# Linux/Mac
uuidgen

# или используйте онлайн генератор
# https://www.uuidgenerator.net/
```

## Протоколы

### Inbound протоколы
- **http**: HTTP/HTTPS прокси с поддержкой CONNECT
- **socks**: SOCKS5 прокси
- **koria**: Принимает соединения по Koria протоколу

### Outbound протоколы
- **freedom**: Прямое соединение (direct)
- **koria**: Туннелирование через Koria протокол

## Routing

Routing позволяет маршрутизировать трафик через разные outbound'ы на основе правил:

```json
{
  "type": "field",
  "domain": ["google.com", "*.google.com"],  // Domain matching
  "ip": ["8.8.8.8/32", "8.8.4.4/32"],       // IP CIDR matching
  "port": "80,443,8080-8090",                // Port matching
  "network": "tcp",                          // tcp|udp
  "outboundTag": "koria-out"                 // Target outbound
}
```

Правила применяются сверху вниз. Первое совпадение определяет outbound.

## Примеры использования

### 1. Простой HTTP прокси через Koria
```json
// client.json - минимальная конфигурация
{
  "inbounds": [{"tag": "http", "protocol": "http", "listen": "127.0.0.1:8080"}],
  "outbounds": [{"tag": "koria", "protocol": "koria", "settings": {...}}]
}
```

### 2. Split tunneling (локальный трафик напрямую, остальное через Koria)
```json
{
  "routing": {
    "rules": [
      {"domain": ["*.ru", "*.local"], "outboundTag": "direct"},
      {"ip": ["192.168.0.0/16", "10.0.0.0/8"], "outboundTag": "direct"},
      {"outboundTag": "koria-out"}  // default
    ]
  }
}
```

### 3. Несколько серверов Koria
```json
{
  "outbounds": [
    {"tag": "koria-us", "protocol": "koria", "settings": {"address": "us.server.com", ...}},
    {"tag": "koria-eu", "protocol": "koria", "settings": {"address": "eu.server.com", ...}}
  ],
  "routing": {
    "rules": [
      {"domain": ["*.com"], "outboundTag": "koria-us"},
      {"domain": ["*.eu"], "outboundTag": "koria-eu"}
    ]
  }
}
```
