# Deployment Instructions - WebSocket Fix

## Что изменилось

1. **Frontend:** [`frontend/src/lib/api/websocket.ts`](frontend/src/lib/api/websocket.ts:39) - добавлена передача токена через URL
2. **Backend:** [`api-gateway/internal/cores/middleware/middleware.go`](api-gateway/internal/cores/middleware/middleware.go:28) - добавлено чтение токена из query параметра

## Шаги для деплоя

### Вариант 1: Полный пересборка (рекомендуется)

```bash
# На сервере, в директории проекта

# 1. Получить изменения из git
git pull

# 2. Пересобрать frontend
cd frontend
npm run build
cd ..

# 3. Пересобрать и перезапустить все сервисы
docker-compose down
docker-compose up --build -d

# 4. Проверить логи
docker-compose logs -f api-gateway
```

### Вариант 2: Быстрый деплой (только измененные сервисы)

```bash
# На сервере

# 1. Получить изменения
git pull

# 2. Пересобрать frontend
cd frontend
npm run build
cd ..

# 3. Пересобрать только api-gateway (он использует измененный middleware)
docker-compose up --build -d api-gateway

# 4. Перезапустить Caddy чтобы подхватить новый frontend
docker-compose restart caddy

# 5. Проверить
docker-compose logs -f api-gateway caddy
```

### Вариант 3: Без git (если изменения только локальные)

```bash
# На локальной машине - скопировать файлы на сервер
scp frontend/src/lib/api/websocket.ts user@server:/path/to/project/frontend/src/lib/api/
scp api-gateway/internal/cores/middleware/middleware.go user@server:/path/to/project/api-gateway/internal/cores/middleware/

# На сервере - выполнить шаги из Варианта 2
```

## Проверка после деплоя

### 1. Проверить что сервисы запущены

```bash
docker-compose ps
```

Все сервисы должны быть в статусе `Up`.

### 2. Проверить логи api-gateway

```bash
docker-compose logs -f api-gateway | grep "/ws"
```

После успешного подключения должно быть:
```
[GIN] 2026/04/08 - 17:30:00 | 101 | ... | GET "/ws?token=..."
```

Статус **101** (Switching Protocols) = успех!  
Статус **401** (Unauthorized) = токен не передается или невалиден

### 3. Проверить в браузере

1. Откройте приложение в браузере
2. Залогиньтесь
3. Откройте DevTools → Console
4. Должно быть: `[WebSocket] Connected`
5. Откройте DevTools → Network → WS
6. Должно быть соединение с статусом `101 Switching Protocols`

### 4. Проверить что токен передается

В DevTools → Network → WS → Headers должно быть:
```
Request URL: ws://your-server/ws?token=eyJhbGc...
```

## Troubleshooting

### Проблема: Все еще 401 ошибка

**Причина 1:** Frontend не пересобран
```bash
cd frontend
npm run build
docker-compose restart caddy
```

**Причина 2:** api-gateway не перезапущен
```bash
docker-compose up --build -d api-gateway
```

**Причина 3:** Токен не сохраняется в куки
- Проверьте что логин прошел успешно
- Откройте DevTools → Application → Cookies
- Должна быть кука `access_token`

### Проблема: WebSocket подключается, но сразу отключается

Проверьте логи realtime-service:
```bash
docker-compose logs -f realtime-service
```

### Проблема: CORS ошибки

Убедитесь что в `api-gateway/internal/gateway/transport/websocket.go` правильно настроен `CheckOrigin`:

```go
CheckOrigin: func(r *http.Request) bool {
    if os.Getenv("ENV") == "development" {
        origin := r.Header.Get("Origin")
        return strings.HasPrefix(origin, "http://localhost:") || 
               strings.HasPrefix(origin, "http://127.0.0.1:")
    }
    
    // Production: добавьте ваш домен
    origin := r.Header.Get("Origin")
    allowedOrigins := []string{
        "http://95.165.87.48",
        "https://your-domain.com",
    }
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
},
```

## Быстрая команда для полного деплоя

```bash
#!/bin/bash
# deploy.sh

set -e  # Exit on error

echo "🚀 Starting deployment..."

# Pull changes
echo "📥 Pulling changes from git..."
git pull

# Build frontend
echo "🔨 Building frontend..."
cd frontend
npm run build
cd ..

# Rebuild and restart services
echo "🐳 Rebuilding Docker containers..."
docker-compose down
docker-compose up --build -d

# Wait for services to start
echo "⏳ Waiting for services to start..."
sleep 10

# Check status
echo "✅ Checking service status..."
docker-compose ps

# Show logs
echo "📋 Showing recent logs..."
docker-compose logs --tail=50 api-gateway

echo "✨ Deployment complete!"
echo "🔍 Check WebSocket connection in browser DevTools"
```

Сохраните как `deploy.sh`, сделайте исполняемым и запустите:
```bash
chmod +x deploy.sh
./deploy.sh
```

## Минимальные команды (если спешите)

```bash
git pull && cd frontend && npm run build && cd .. && docker-compose up --build -d api-gateway && docker-compose restart caddy
```

Одной строкой - получить изменения, собрать фронт, пересобрать api-gateway, перезапустить caddy.
