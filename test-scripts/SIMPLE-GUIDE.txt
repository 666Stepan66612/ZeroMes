# Простое тестирование WebSocket мессенджера

## Шаг 1: Запуск Docker

```bash
docker compose up --build -d
docker compose logs -f  # Смотри логи в отдельном терминале
```

---

## ТЕРМИНАЛ 1 - User1 (alice1)

### 1. Регистрация alice1
```bash
curl -s -X POST http://localhost:8083/auth/register \
  -H "Content-Type: application/json" \
  -d '{"login":"alice1","auth_hash":"hash123","public_key":"pubkey123"}' \
  | tee /tmp/alice1-register.json | jq . && \
  jq -r '.user.id' /tmp/alice1-register.json > /tmp/alice1-id.txt && \
  echo "alice1 ID сохранен: $(cat /tmp/alice1-id.txt)"
```

### 2. Логин alice1
```bash
curl -s -X POST http://localhost:8083/auth/login \
  -H "Content-Type: application/json" \
  -c /tmp/alice1-cookies.txt \
  -d '{"login":"alice1","auth_hash":"hash123"}' | jq .
```

### 3. Поиск bob1 (после его регистрации в терминале 2)
```bash
TOKEN=$(cat /tmp/alice1-cookies.txt | grep access_token | awk '{print $7}')
curl -s -X GET "http://localhost:8083/auth/search?login=bob1" \
  -H "Cookie: access_token=$TOKEN" \
  | tee /tmp/bob1-search.json | jq . && \
  jq -r '.users[0].id' /tmp/bob1-search.json > /tmp/bob1-id.txt && \
  echo "bob1 ID сохранен: $(cat /tmp/bob1-id.txt)"
```

### 4. Подключение к WebSocket
```bash
TOKEN=$(cat /tmp/alice1-cookies.txt | grep access_token | awk '{print $7}')
echo "alice1 ID: $(cat /tmp/alice1-id.txt)"
echo "bob1 ID: $(cat /tmp/bob1-id.txt)"
echo "Подключаюсь к WebSocket..."
websocat -H="Cookie: access_token=$TOKEN" ws://localhost:8083/ws
```

После подключения увидишь:
```json
{"type":"status","payload":{"connected":true,"user_id":"..."}}
```

### 5. Отправить сообщение bob1
Вставь в WebSocket (автоматически подставит bob1 ID):
```bash
# Скопируй эту команду целиком и вставь в WebSocket терминал:
{"type":"send_message","recipient_id":"$(cat /tmp/bob1-id.txt)","content":"Hello bob1!","message_type":"text"}
```

Или вручную замени BOB1_ID:
```json
{"type":"send_message","recipient_id":"BOB1_ID","content":"Hello bob1!","message_type":"text"}
```

Получишь ответ с MSG_ID - скопируй его!

### 6. Изменить свое сообщение
```json
{"type":"alter_message","message_id":"MSG_ID","new_content":"Hello bob1! (edited)"}
```

### 7. Получить список чатов
```json
{"type":"get_chats"}
```

Скопируй CHAT_ID из ответа!

### 8. Получить историю сообщений
```json
{"type":"get_messages","chat_id":"CHAT_ID","limit":50}
```

### 9. Отметить как прочитанное
```json
{"type":"mark_as_read","chat_id":"CHAT_ID","last_message_id":"MSG_ID"}
```

---

## ТЕРМИНАЛ 2 - User2 (bob1)

### 1. Регистрация bob1
```bash
curl -s -X POST http://localhost:8083/auth/register \
  -H "Content-Type: application/json" \
  -d '{"login":"bob1","auth_hash":"hash456","public_key":"pubkey456"}' \
  | tee /tmp/bob1-register.json | jq . && \
  jq -r '.user.id' /tmp/bob1-register.json > /tmp/bob1-id.txt && \
  echo "bob1 ID сохранен: $(cat /tmp/bob1-id.txt)"
```

### 2. Логин bob1
```bash
curl -s -X POST http://localhost:8083/auth/login \
  -H "Content-Type: application/json" \
  -c /tmp/bob1-cookies.txt \
  -d '{"login":"bob1","auth_hash":"hash456"}' | jq .
```

### 3. Подключение к WebSocket
```bash
TOKEN=$(cat /tmp/bob1-cookies.txt | grep access_token | awk '{print $7}')
echo "bob1 ID: $(cat /tmp/bob1-id.txt)"
echo "alice1 ID: $(cat /tmp/alice1-id.txt)"
echo "Подключаюсь к WebSocket..."
websocat -H="Cookie: access_token=$TOKEN" ws://localhost:8083/ws
```

### 4. Автоматически получишь сообщение от alice1
```json
{"type":"new_message","payload":{"message_id":"MSG_ID","sender_id":"ALICE1_ID","content":"Hello bob1!",...}}
```

Скопируй MSG_ID и CHAT_ID из сообщения!

### 5. Отметить как прочитанное
```json
{"type":"mark_as_read","chat_id":"CHAT_ID","last_message_id":"MSG_ID"}
```

alice1 автоматически получит уведомление:
```json
{"type":"message_read","payload":{...}}
```

### 6. Отправить ответ alice1
```json
{"type":"send_message","recipient_id":"ALICE1_ID","content":"Hi alice1!","message_type":"text"}
```

alice1 автоматически получит это сообщение.

### 7. Удалить свое сообщение
```json
{"type":"delete_message","message_id":"MSG_ID"}
```

Оба пользователя увидят событие удаления:
```json
{"type":"message_deleted","payload":{...}}
```

### 8. Disconnect и Reconnect
- Нажми `Ctrl+C` для отключения
- Снова выполни команду подключения (шаг 3)
- Получи историю:
```json
{"type":"get_messages","chat_id":"CHAT_ID","limit":50}
```

---

## Быстрая проверка сохраненных ID

```bash
echo "alice1 ID: $(cat /tmp/alice1-id.txt)"
echo "bob1 ID: $(cat /tmp/bob1-id.txt)"
```

---

## Формат CHAT_ID

Chat ID = два UUID отсортированных по алфавиту через двоеточие.

Пример:
```
70a43e76-e48a-49e8-bfbf-c489073154bf:a00fb982-d0aa-4d0f-aad1-2eefeeffbfcc
```

Меньший UUID идет первым.

---

## Полный сценарий тестирования

1. **Терминал 1**: Регистрация alice1 → Логин
2. **Терминал 2**: Регистрация bob1 → Логин
3. **Терминал 1**: Поиск bob1 → Подключение WS
4. **Терминал 2**: Подключение WS
5. **Терминал 1**: Отправить сообщение bob1
6. **Терминал 2**: Автоматически получит → Отметить прочитанным
7. **Терминал 1**: Автоматически получит уведомление о прочтении
8. **Терминал 2**: Отправить ответ alice1
9. **Терминал 1**: Автоматически получит → Отметить прочитанным
10. **Терминал 1**: Изменить свое сообщение
11. **Терминал 2**: Автоматически получит событие изменения
12. **Терминал 2**: Удалить свое сообщение
13. **Терминал 1**: Автоматически получит событие удаления
14. **Терминал 1**: Отправить еще сообщение
15. **Терминал 2**: Disconnect → Reconnect → get_messages

---

## Полезные команды

### Проверить что Docker работает
```bash
docker compose ps
```

### Посмотреть логи
```bash
docker compose logs -f api-gateway
docker compose logs -f realtime-service
docker compose logs -f message-service
docker compose logs -f auth-service
```

### Остановить все
```bash
docker compose down -v
```

### Перезапустить
```bash
docker compose restart
```

### Очистить сохраненные данные
```bash
rm -f /tmp/alice1-* /tmp/bob1-*
```

---

## Типичные ошибки

**"failed to send message"** 
- Проверь что recipient_id это валидный UUID
- Убедись что получатель зарегистрирован

**"token revoked"** 
- Перелогинься (шаг 2)

**"invalid request"** 
- Проверь JSON синтаксис (кавычки, запятые)
- Убедись что все обязательные поля заполнены

**Connection refused** 
- Docker не запущен: `docker compose up -d`
- Порты заняты: `docker compose ps`

**"users": []** при поиске
- Пользователь еще не зарегистрирован
- Проверь правильность логина

---

## Структура событий WebSocket

### Исходящие (от клиента):
- `send_message` - отправить сообщение
- `get_messages` - получить историю
- `mark_as_read` - отметить прочитанным
- `alter_message` - изменить сообщение
- `delete_message` - удалить сообщение
- `get_chats` - список чатов

### Входящие (от сервера):
- `status` - статус подключения
- `message_sent` - подтверждение отправки
- `new_message` - новое сообщение (автоматически)
- `message_read` - сообщение прочитано (автоматически)
- `message_altered` - сообщение изменено (автоматически)
- `message_deleted` - сообщение удалено (автоматически)
- `messages` - история сообщений
- `chats` - список чатов
- `error` - ошибка
