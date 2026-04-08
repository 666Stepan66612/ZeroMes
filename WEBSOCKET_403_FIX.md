# WebSocket 403 Error Fix

## Проблема

После исправления 401 ошибки появилась новая проблема - **403 Forbidden**:

```
[GIN-debug] [WARNING] Headers were already written. Wanted to override status code 403 with 500
[GIN] 2026/04/08 - 17:35:04 | 403 | 388.845µs | 212.16.25.111 | GET "/ws"
```

## Причины

### 1. CORS в WebSocket Upgrader

В [`websocket.go:28`](api-gateway/internal/gateway/transport/websocket.go:28) был неправильный `CheckOrigin`:

```go
CheckOrigin: func(r *http.Request) bool {
    if os.Getenv("ENV") == "development" {
        origin := r.Header.Get("Origin")
        return strings.HasPrefix(origin, "http://localhost:") || 
               strings.HasPrefix(origin, "http://127.0.0.1:")
    }
    
    return true  // ✅ Это работает только если ENV != "development"
},
```

Проблема: если `ENV` не установлен или равен "development", то для production IP адресов возвращается `false` (по умолчанию).

### 2. Rate Limiting на WebSocket

WebSocket пытается переподключиться при ошибках, что может вызвать срабатывание rate limiter.

## Решение

### 1. Упростить CheckOrigin

```go
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins (WebSocket is authenticated via JWT)
		return true
	},
}
```

**Безопасно ли это?**  
✅ Да, потому что:
- WebSocket защищен JWT токеном
- Токен проверяется в middleware перед upgrade
- Без валидного токена соединение не установится
- CORS для WebSocket работает иначе чем для HTTP

### 2. Убрать Rate Limiting с WebSocket (опционально)

WebSocket - это долгоживущее соединение, rate limiting может мешать reconnect'ам.

## Деплой исправления

```bash
# На сервере
git pull
docker-compose up --build -d api-gateway
docker-compose logs -f api-gateway
```

## Проверка

После деплоя в логах должно быть:

```
[GIN] 2026/04/08 - 17:40:00 | 101 | ... | GET "/ws?token=..."
```

Статус **101 Switching Protocols** = успех!

## Альтернативное решение (более строгое)

Если хотите ограничить origins в production:

```go
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		
		// Development
		if strings.HasPrefix(origin, "http://localhost:") || 
		   strings.HasPrefix(origin, "http://127.0.0.1:") {
			return true
		}
		
		// Production - whitelist your domains/IPs
		allowedOrigins := []string{
			"http://95.165.87.48",
			"https://your-domain.com",
		}
		
		for _, allowed := range allowedOrigins {
			if origin == allowed || origin == "" {
				return true
			}
		}
		
		return false
	},
}
```

Но для начала лучше использовать простой вариант с `return true`.

## Почему 403 вместо другой ошибки?

WebSocket upgrade возвращает 403 когда `CheckOrigin` возвращает `false`. Это стандартное поведение gorilla/websocket.

## Дополнительно: Логирование для отладки

Добавьте логирование в CheckOrigin для отладки:

```go
CheckOrigin: func(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	slog.Info("WebSocket origin check", "origin", origin, "remote", r.RemoteAddr)
	return true
},
```

Это поможет увидеть откуда приходят запросы.
