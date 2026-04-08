# WebSocket Authentication Fix

## Проблема

WebSocket соединение получало **401 Unauthorized** ошибку при попытке подключения:

```
[WebSocket] Error: Event
[WebSocket] Closed: 1006 
[WebSocket] Reconnecting in 1000ms (attempt 1/5)
```

### Причина

**WebSocket API в браузере не передает куки автоматически** при создании соединения через `new WebSocket(url)`. Это особенность работы WebSocket протокола - в отличие от обычных HTTP запросов, куки не включаются в handshake автоматически во всех браузерах и сценариях (особенно при cross-origin запросах).

Наш код пытался получить токен только из куки:

```typescript
// frontend/src/lib/api/websocket.ts (старый код)
constructor(baseUrl?: string) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host;
  this.url = `${protocol}//${host}/ws`;  // ❌ Токен не передается!
}
```

```go
// api-gateway/internal/cores/middleware/middleware.go (старый код)
func JWTMiddleware(secret string, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string
		cookie, err := c.Cookie("access_token")  // ❌ Кука не приходит!
		if err == nil {
			token = cookie
		}
		// ...
	}
}
```

## Решение

### 1. Frontend: Передача токена через URL параметр

Добавили метод для чтения токена из куки и передачи его в URL:

```typescript
// frontend/src/lib/api/websocket.ts
constructor(baseUrl?: string) {
  if (!baseUrl) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    baseUrl = `${protocol}//${host}`;
  }
  
  // ✅ Читаем токен из куки и передаем в URL
  const token = this.getAccessToken();
  this.url = token ? `${baseUrl}/ws?token=${encodeURIComponent(token)}` : `${baseUrl}/ws`;
}

/**
 * Get access token from cookie
 */
private getAccessToken(): string | null {
  const cookies = document.cookie.split(';');
  for (const cookie of cookies) {
    const [name, value] = cookie.trim().split('=');
    if (name === 'access_token') {
      return decodeURIComponent(value);
    }
  }
  return null;
}
```

### 2. Backend: Чтение токена из query параметра

Добавили fallback для чтения токена из URL параметра:

```go
// api-gateway/internal/cores/middleware/middleware.go
func JWTMiddleware(secret string, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// Try cookie first
		cookie, err := c.Cookie("access_token")
		if err == nil {
			token = cookie
		} else {
			// Try Authorization header
			token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}

		// ✅ For WebSocket: try query parameter (since cookies may not be sent)
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return 
		}
		// ... rest of validation
	}
}
```

## Как это работает

1. **При логине** токен сохраняется в куку `access_token`
2. **При создании WebSocket** клиент читает токен из куки
3. **Токен передается в URL** как query параметр: `ws://host/ws?token=xxx`
4. **Middleware на бэкенде** проверяет токен в следующем порядке:
   - Сначала в куке (для обычных HTTP запросов)
   - Потом в Authorization header (для API клиентов)
   - Наконец в query параметре (для WebSocket)

## Альтернативные решения

### 1. Использование Sec-WebSocket-Protocol

Можно передавать токен через WebSocket subprotocol:

```typescript
// Frontend
const ws = new WebSocket('ws://host/ws', ['access_token', token]);

// Backend (Gin)
token := c.GetHeader("Sec-WebSocket-Protocol")
```

**Минусы:** Не все прокси и балансировщики корректно обрабатывают custom protocols.

### 2. Первое сообщение с токеном

Отправлять токен в первом сообщении после подключения:

```typescript
ws.onopen = () => {
  ws.send(JSON.stringify({ type: 'auth', token: token }));
};
```

**Минусы:** Сложнее реализация, нужна буферизация сообщений до авторизации.

### 3. Использование credentials: 'include'

WebSocket API не поддерживает опцию `credentials`, в отличие от `fetch()`.

## Безопасность

### Потенциальные риски передачи токена в URL:

1. **Логирование URL** - токены могут попасть в логи сервера
2. **Browser history** - токены могут сохраниться в истории браузера
3. **Referrer leakage** - токены могут утечь через Referer header

### Меры защиты:

1. ✅ **Короткий TTL токенов** - access token живет 15 минут
2. ✅ **HTTPS/WSS** - шифрование соединения
3. ✅ **Token rotation** - refresh token для обновления
4. ✅ **Blacklist в Redis** - отозванные токены не работают
5. ⚠️ **Не логировать query параметры** - настроить логирование на сервере

### Рекомендации для production:

```go
// Не логировать query параметры с токенами
r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
    // Скрыть токен в логах
    path := param.Path
    if strings.Contains(path, "token=") {
        path = strings.Split(path, "?")[0] + "?token=***"
    }
    return fmt.Sprintf("[%s] %s %s %d\n",
        param.TimeStamp.Format("2006/01/02 - 15:04:05"),
        param.Method,
        path,
        param.StatusCode,
    )
}))
```

## Тестирование

После применения фикса:

1. Залогиньтесь в приложение
2. Откройте DevTools → Network → WS
3. Проверьте что WebSocket подключается успешно
4. Статус должен быть `101 Switching Protocols`
5. В консоли должно быть: `[WebSocket] Connected`

## Дополнительные улучшения

### 1. Обновление токена при reconnect

Если токен истек во время reconnect, нужно получить новый:

```typescript
private async refreshTokenAndReconnect() {
  try {
    await fetch('/api/auth/refresh', { 
      method: 'POST',
      credentials: 'include' 
    });
    
    // Обновляем URL с новым токеном
    const token = this.getAccessToken();
    this.url = token ? `${this.baseUrl}/ws?token=${encodeURIComponent(token)}` : `${this.baseUrl}/ws`;
    
    this.connect();
  } catch (error) {
    console.error('Failed to refresh token:', error);
  }
}
```

### 2. Caddy конфигурация

Убедитесь что Caddy правильно проксирует WebSocket:

```caddyfile
@websocket {
    path /ws
}
handle @websocket {
    reverse_proxy api-gateway:8083 {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
        # WebSocket headers
        header_up Connection {>Connection}
        header_up Upgrade {>Upgrade}
    }
}
```

## Заключение

Проблема решена передачей JWT токена через URL query параметр. Это стандартный подход для WebSocket аутентификации, используемый многими приложениями (включая Socket.io, SignalR и др.).

**Статус:** ✅ Исправлено  
**Файлы изменены:**
- [`frontend/src/lib/api/websocket.ts`](frontend/src/lib/api/websocket.ts:39)
- [`api-gateway/internal/cores/middleware/middleware.go`](api-gateway/internal/cores/middleware/middleware.go:16)
