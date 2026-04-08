# Настройка HTTPS для khmelev.site

## Шаг 1: Настройте DNS

Зайдите в панель управления вашим доменом и создайте A-запись:

```
Тип: A
Имя: @
Значение: 95.165.87.48
TTL: Auto или 3600
```

Опционально добавьте www:
```
Тип: A
Имя: www
Значение: 95.165.87.48
TTL: Auto или 3600
```

Проверьте DNS (может занять до 5-10 минут):
```bash
dig khmelev.site +short
# Должно вернуть: 95.165.87.48
```

## Шаг 2: Обновите Caddyfile

На сервере замените текущий Caddyfile:

```bash
# Сделайте бэкап
cp Caddyfile Caddyfile.backup

# Скопируйте новый
cp Caddyfile.production Caddyfile

# Или вручную отредактируйте
nano Caddyfile
```

Замените содержимое на:

```caddyfile
{
    admin off
    email your-email@example.com  # Укажите ваш email!
}

khmelev.site {
    # Caddy автоматически получит SSL!
    
    log {
        output stdout
        format console
        level INFO
    }
    
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
    
    handle /api/* {
        uri strip_prefix /api
        reverse_proxy api-gateway:8083 {
            header_up Host {host}
            header_up X-Real-IP {remote_host}
            header_up X-Forwarded-For {remote_host}
            header_up X-Forwarded-Proto {scheme}
        }
    }
    
    handle {
        root * /var/www/frontend
        try_files {path} /index.html
        file_server
    }
}
```

**Важно:** Замените `your-email@example.com` на ваш реальный email!

## Шаг 3: Откройте порты

Убедитесь что порты 80 и 443 открыты:

```bash
# Проверить firewall
sudo ufw status

# Если нужно, откройте порты
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

## Шаг 4: Перезапустите Caddy

```bash
docker-compose restart caddy

# Проверьте логи
docker-compose logs -f caddy
```

В логах должно быть:
```
obtaining certificate for khmelev.site
certificate obtained successfully
```

## Шаг 5: Проверьте HTTPS

Откройте в браузере: `https://khmelev.site`

Должен быть зеленый замочек 🔒 и валидный SSL сертификат!

## Шаг 6: Обновите frontend (если нужно)

Если в коде есть хардкод URL, обновите:

```bash
cd frontend
npm run build
cd ..
docker-compose restart caddy
```

## Troubleshooting

### Ошибка: "failed to obtain certificate"

**Причина:** DNS еще не обновился или порты закрыты.

**Решение:**
```bash
# Проверьте DNS
dig khmelev.site +short

# Проверьте порты
curl -I http://khmelev.site
curl -I https://khmelev.site
```

### Ошибка: "connection refused"

**Причина:** Firewall блокирует порты.

**Решение:**
```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw reload
```

### Caddy не может получить сертификат

**Причина:** Let's Encrypt не может достучаться до сервера.

**Решение:**
1. Убедитесь что DNS указывает на правильный IP
2. Проверьте что порт 80 открыт (Let's Encrypt использует HTTP challenge)
3. Временно отключите другие веб-серверы на портах 80/443

### Проверка статуса сертификата

```bash
# Посмотреть сертификаты Caddy
docker-compose exec caddy caddy list-certificates

# Проверить сертификат через openssl
openssl s_client -connect khmelev.site:443 -servername khmelev.site < /dev/null 2>/dev/null | openssl x509 -noout -dates
```

## Автоматическое обновление сертификата

Caddy автоматически обновляет сертификаты за 30 дней до истечения. Ничего делать не нужно!

## Редирект HTTP → HTTPS

Caddy автоматически редиректит HTTP на HTTPS. Проверьте:

```bash
curl -I http://khmelev.site
# Должен вернуть: 308 Permanent Redirect
# Location: https://khmelev.site/
```

## Готово! 🎉

Теперь ваш мессенджер работает по HTTPS и Web Crypto API доступен!

Откройте `https://khmelev.site` и проверьте что:
- ✅ Зеленый замочек в адресной строке
- ✅ WebSocket подключается
- ✅ Можно создавать чаты
- ✅ Сообщения шифруются

## Дополнительно: WWW редирект

Если хотите чтобы `www.khmelev.site` редиректил на `khmelev.site`, добавьте в Caddyfile:

```caddyfile
www.khmelev.site {
    redir https://khmelev.site{uri} permanent
}
```
