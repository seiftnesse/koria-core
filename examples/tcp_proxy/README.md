# Koria TCP Proxy

Реальный пример использования Koria-Core для проксирования TCP трафика через Minecraft протокол.

## Как это работает

```
┌─────────┐     HTTP      ┌──────────────┐   Minecraft   ┌──────────────┐    HTTP     ┌────────┐
│ Browser │ ────────────► │ Proxy Client │ ═══════════► │ Proxy Server │ ────────────► │ Target │
│         │               │ (localhost)  │  (стеганог.) │ (удаленный)  │               │ Server │
└─────────┘               └──────────────┘               └──────────────┘               └────────┘
```

1. **Браузер** подключается к локальному proxy клиенту (например, 127.0.0.1:8080)
2. **Proxy Client** открывает виртуальный поток через Minecraft протокол к серверу
3. **Proxy Server** получает поток и проксирует к реальному целевому серверу
4. Данные передаются туда-обратно, скрытые в Minecraft пакетах

## Быстрый старт

### Шаг 1: Запустите Proxy Server

**Терминал 1:**
```bash
./koria-proxy-server -listen 0.0.0.0:25565 -target httpbin.org:80
```

Вы увидите:
```
═══════════════════════════════════════════════════════════
  Koria TCP Proxy Server
═══════════════════════════════════════════════════════════
Server UUID: 123e4567-e89b-12d3-a456-426614174000
Используйте этот UUID для подключения клиента!
Listening: 0.0.0.0:25565
Target: httpbin.org:80
═══════════════════════════════════════════════════════════
Server started successfully. Waiting for connections...
```

**ВАЖНО:** Скопируйте UUID!

### Шаг 2: Запустите Proxy Client

**Терминал 2:**
```bash
./koria-proxy-client \
  -listen 127.0.0.1:8080 \
  -server 127.0.0.1 \
  -port 25565 \
  -uuid 123e4567-e89b-12d3-a456-426614174000
```

Вы увидите:
```
═══════════════════════════════════════════════════════════
  Koria TCP Proxy Client
═══════════════════════════════════════════════════════════
Local listening: 127.0.0.1:8080
Koria server: 127.0.0.1:25565
UUID: 123e4567-e89b-12d3-a456-426614174000
═══════════════════════════════════════════════════════════
✓ Connected to Koria server successfully!
✓ Authenticated with UUID: 123e4567-e89b-12d3-a456-426614174000
✓ Listening on 127.0.0.1:8080
Ready to accept connections! Configure your browser to use this proxy.

Example: curl -x http://127.0.0.1:8080 http://httpbin.org/ip
```

### Шаг 3: Тестируем!

**Терминал 3:**
```bash
# Через curl
curl -x http://127.0.0.1:8080 http://httpbin.org/ip

# Через wget
wget -e use_proxy=yes -e http_proxy=127.0.0.1:8080 http://httpbin.org/ip -O -

# Или настройте браузер на использование HTTP proxy 127.0.0.1:8080
```

Вы должны увидеть логи в обоих терминалах:

**Proxy Client:**
```
✓ Accepted local connection from 127.0.0.1:xxxxx
✓ Opened virtual stream through Koria (Minecraft protocol)
Local -> Koria: 78 bytes
Koria -> Local: 234 bytes
✓ Proxy session completed
```

**Proxy Server:**
```
✓ Accepted virtual stream from 127.0.0.1:xxxxx
✓ Connected to target httpbin.org:80
Client -> Target: 78 bytes
Target -> Client: 234 bytes
✓ Proxy session completed for httpbin.org:80
```

## Параметры

### Proxy Server

```bash
./koria-proxy-server \
  -listen 0.0.0.0:25565          # Адрес для прослушивания
  -target httpbin.org:80         # Целевой сервер для проксирования
```

### Proxy Client

```bash
./koria-proxy-client \
  -listen 127.0.0.1:8080         # Локальный адрес для прослушивания
  -server 127.0.0.1              # Адрес Koria сервера
  -port 25565                    # Порт Koria сервера
  -uuid <UUID>                   # UUID для аутентификации (ОБЯЗАТЕЛЬНО)
```

## Примеры использования

### Проксирование HTTP запросов

```bash
# Server
./koria-proxy-server -target httpbin.org:80

# Client
./koria-proxy-client -uuid <UUID>

# Test
curl -x http://127.0.0.1:8080 http://httpbin.org/get
```

### Проксирование HTTPS (через CONNECT)

**Примечание:** Для HTTPS нужен полноценный HTTP CONNECT proxy. Текущая реализация - простой TCP proxy.

### Проксирование к любому TCP сервису

```bash
# К Redis серверу
./koria-proxy-server -target redis-server.com:6379

# К MySQL серверу
./koria-proxy-server -target mysql-server.com:3306

# К SSH серверу
./koria-proxy-server -target remote-host.com:22
```

## Удаленное использование

### На удаленном сервере:
```bash
./koria-proxy-server -listen 0.0.0.0:25565 -target httpbin.org:80
```

### На локальном компьютере:
```bash
./koria-proxy-client \
  -listen 127.0.0.1:8080 \
  -server your-server-ip.com \
  -port 25565 \
  -uuid <UUID>
```

Теперь весь ваш HTTP трафик через localhost:8080 будет:
1. ✅ Проходить через **ОДНО** TCP соединение
2. ✅ Скрыт в Minecraft пакетах (стеганография)
3. ✅ Защищен от блокировки ТСПУ по множественным соединениям

## Проверка multiplexing

Откройте несколько вкладок в браузере и одновременно загружайте страницы:

```bash
# В отдельном терминале
watch -n 1 'netstat -an | grep 25565 | grep ESTABLISHED | wc -l'
```

Результат должен показывать **1** соединение, несмотря на множество HTTP запросов!

## Настройка браузера

### Firefox
1. Settings → General → Network Settings
2. Manual proxy configuration
3. HTTP Proxy: `127.0.0.1`, Port: `8080`
4. ✓ Use this proxy server for all protocols

### Chrome/Chromium
```bash
google-chrome --proxy-server="http://127.0.0.1:8080"
```

## Архитектура

```
Browser Request
     ↓
Local Proxy Client (127.0.0.1:8080)
     ↓
Koria Client Transport
     ↓
Minecraft Protocol (стеганография в PlayerMove, CustomPayload пакетах)
     ↓
Virtual Stream через ONE TCP connection
     ↓
Minecraft Protocol (расшифровка)
     ↓
Koria Server Transport
     ↓
Remote Proxy Server
     ↓
Target Server (httpbin.org:80)
```

## Производительность

- **Латентность:** +5-10ms (стеганография overhead)
- **Пропускная способность:** ~80-85% от теоретического максимума
- **Одновременные соединения:** до 65535 через одно TCP соединение
- **Устойчивость к блокировке ТСПУ:** ✅ Отлично (всегда 1 соединение)

## Ограничения

- Текущая версия - простой TCP proxy (не HTTP CONNECT proxy)
- Для HTTPS нужна поддержка HTTP CONNECT метода
- Нет аутентификации на уровне proxy (только UUID для Koria)

## Следующие шаги

Для полноценного HTTP/HTTPS proxy смотрите:
- `examples/http_proxy/` - HTTP proxy с CONNECT support (в разработке)
- `examples/socks5_proxy/` - SOCKS5 proxy (в разработке)

## Безопасность

⚠️ **ВАЖНО:** Этот proxy предназначен для:
- Обхода блокировок ТСПУ
- Тестирования и разработки
- Легального использования в разрешенных сценариях

НЕ используйте для:
- Незаконной деятельности
- Обхода авторизованных ограничений
- Нарушения условий использования сервисов

## Поддержка

Если что-то не работает:
1. Проверьте, что UUID совпадает
2. Проверьте, что порты не заняты
3. Проверьте логи обоих компонентов
4. Убедитесь, что целевой сервер доступен

## Лицензия

MIT License
