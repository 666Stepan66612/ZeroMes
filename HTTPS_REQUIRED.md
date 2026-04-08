# HTTP vs HTTPS - Web Crypto API Error Fix

## Проблема

При попытке создать новый чат возникает ошибка:

```
Failed to create chat: TypeError: Cannot read properties of undefined (reading 'importKey')
```

## Причина

**Web Crypto API (`crypto.subtle`) доступен только в безопасном контексте:**
- ✅ HTTPS
- ✅ localhost (HTTP)
- ❌ HTTP на production IP/домене

Ваш сайт работает по HTTP на IP адресе `95.165.87.48`, поэтому `crypto.subtle` = `undefined`.

## Решение

### Вариант 1: Настроить HTTPS (Рекомендуется)

#### A. С доменом (бесплатный SSL от Let's Encrypt)

1. **Получите домен** (например, на Cloudflare, Namecheap)

2. **Настройте DNS** - укажите A-запись на ваш IP:
```
A    @    95.165.87.48
A    www  95.165.87.48
```

3. **Обновите Caddyfile:**
```caddyfile
your-domain.com {
    # Caddy автоматически получит SSL сертификат от Let's Encrypt
    
    log {
        output stdout
        format console
        level INFO
    }
    
    # WebSocket
    @websocket {
        path /ws
    }
    handle @websocket {
        reverse_proxy api-gateway:8083 {
            header_up Host {host}
            header_up X-Real-IP {remote_host}
            header_up X-Forwarded-For {remote_host}
            header_up X-Forwarded-Proto {scheme}
        }
    }
    
    # API
    handle /api/* {
        uri strip_prefix /api
        reverse_proxy api-gateway:8083 {
            header_up Host {host}
            header_up X-Real-IP {remote_host}
            header_up X-Forwarded-For {remote_host}
            header_up X-Forwarded-Proto {scheme}
        }
    }
    
    # Frontend
    handle {
        root * /var/www/frontend
        try_files {path} /index.html
        file_server
    }
}
```

4. **Перезапустите Caddy:**
```bash
docker-compose restart caddy
```

Caddy автоматически получит SSL сертификат!

#### B. Без домена (самоподписанный сертификат)

Если нет домена, можно использовать самоподписанный сертификат (браузер покажет предупреждение, но crypto.subtle будет работать):

```bash
# Создать самоподписанный сертификат
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/CN=95.165.87.48"

# Положить в директорию проекта
mkdir -p certs
mv key.pem cert.pem certs/
```

Обновите `docker-compose.yaml`:
```yaml
caddy:
  image: caddy:2-alpine
  ports:
    - "80:80"
    - "443:443"
  volumes:
    - ./Caddyfile:/etc/caddy/Caddyfile:ro
    - ./frontend/dist:/var/www/frontend:ro
    - ./certs:/certs:ro  # Добавить
    - caddy_data:/data
    - caddy_config:/config
```

Обновите `Caddyfile`:
```caddyfile
https://95.165.87.48 {
    tls /certs/cert.pem /certs/key.pem
    
    # ... rest of config
}
```

### Вариант 2: Временный workaround (НЕ для production!)

Если нужно быстро протестировать, можно использовать ngrok для туннеля с HTTPS:

```bash
# Установить ngrok
wget https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-amd64.tgz
tar xvzf ngrok-v3-stable-linux-amd64.tgz

# Запустить туннель
./ngrok http 80
```

Ngrok даст вам HTTPS URL типа `https://abc123.ngrok.io`

### Вариант 3: Cloudflare Tunnel (бесплатно, с HTTPS)

1. **Установите cloudflared:**
```bash
wget https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
chmod +x cloudflared-linux-amd64
sudo mv cloudflared-linux-amd64 /usr/local/bin/cloudflared
```

2. **Авторизуйтесь:**
```bash
cloudflared tunnel login
```

3. **Создайте туннель:**
```bash
cloudflared tunnel create messenger
cloudflared tunnel route dns messenger your-domain.com
```

4. **Запустите:**
```bash
cloudflared tunnel run --url http://localhost:80 messenger
```

## Что уже исправлено в коде

Добавлена проверка доступности Web Crypto API с понятным сообщением об ошибке:

### Новый файл: [`frontend/src/lib/crypto/utils.ts`](frontend/src/lib/crypto/utils.ts:1)

```typescript
export function getCryptoSubtle(): SubtleCrypto {
  if (!isCryptoAvailable()) {
    throw new Error(
      'Web Crypto API is not available. ' +
      'This application requires HTTPS or localhost to work. ' +
      'Please access the site via HTTPS or configure your server to use SSL/TLS.'
    );
  }
  return window.crypto.subtle;
}
```

Теперь все crypto операции используют `getCryptoSubtle()` вместо прямого `crypto.subtle`, что даст понятную ошибку.

## Проверка после настройки HTTPS

1. Откройте сайт через HTTPS
2. Проверьте в DevTools → Console:
```javascript
console.log('Crypto available:', !!window.crypto.subtle);
console.log('Secure context:', window.isSecureContext);
```

Оба должны быть `true`.

## Почему это важно

Web Crypto API требует HTTPS по соображениям безопасности:
- Предотвращает MITM атаки на криптографические операции
- Гарантирует что ключи не перехватываются
- Стандарт W3C для всех криптографических операций в браузере

## Быстрый чеклист

- [ ] Получить домен (или использовать IP с самоподписанным сертификатом)
- [ ] Настроить DNS (если есть домен)
- [ ] Обновить Caddyfile для HTTPS
- [ ] Перезапустить Caddy: `docker-compose restart caddy`
- [ ] Проверить что сайт открывается по HTTPS
- [ ] Пересобрать frontend: `cd frontend && npm run build`
- [ ] Проверить что crypto.subtle доступен

## Альтернатива: Использовать Cloudflare

Если у вас уже есть домен на Cloudflare:

1. Включите "Full (strict)" SSL/TLS mode
2. Cloudflare автоматически даст HTTPS
3. Настройте Origin Certificate в Cloudflare
4. Используйте этот сертификат в Caddy

## Дополнительные ресурсы

- [MDN: Secure Contexts](https://developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts)
- [Web Crypto API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Crypto_API)
- [Caddy HTTPS](https://caddyserver.com/docs/automatic-https)
- [Let's Encrypt](https://letsencrypt.org/)
