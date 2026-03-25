# Пошаговое тестирование WebSocket мессенджера

## Подготовка

```bash
docker compose up --build -d
docker compose logs -f  # В отдельном терминале для логов
```

---

## Шаг 1: [ТЕРМИНАЛ 1] Регистрация alice1

```bash
curl -s -X POST http://localhost:8083/auth/register \
  -H "Content-Type: application/json" \
  -d '{"login":"alice1","auth_hash":"hash123","public_key":"pubkey123"}' \
  | tee /tmp/alice1-register.json | jq . && \
  jq -r '.user.id' /tmp/alice1-register.json > /tmp/alice1-id.txt && \
  echo "✓ alice1 ID сохранен: $(cat /tmp/alice1-id.txt)"
```

---

## Шаг 2: [ТЕРМИНАЛ 2] Регистрация bob1

```bash
curl -s -X POST http://localhost:8083/auth/register \
  -H "Content-Type: application/json" \
  -d '{"login":"bob1","auth_hash":"hash456","public_key":"pubkey456"}' \
  | tee /tmp/bob1-register.json | jq . && \
  jq -r '.user.id' /tmp/bob1-register.json > /tmp/bob1-id.txt && \
  echo "✓ bob1 ID сохранен: $(cat /tmp/bob1-id.txt)"
```

---

## Шаг 3: [ТЕРМИНАЛ 1] Логин alice1

```bash
curl -s -X POST http://localhost:8083/auth/login \
  -H "Content-Type: application/json" \
  -c /tmp/alice1-cookies.txt \
  -d '{"login":"alice1","auth_hash":"hash123"}' | jq .
echo "✓ alice1 залогинен"
```

---

## Шаг 4: [ТЕРМИНАЛ 2] Логин bob1

```bash
curl -s -X POST http://localhost:8083/auth/login \
  -H "Content-Type: application/json" \
  -c /tmp/bob1-cookies.txt \
  -d '{"login":"bob1","auth_hash":"hash456"}' | jq .
echo "✓ bob1 залогинен"
```

---

## Шаг 5: [ТЕРМИНАЛ 1] alice1 ищет bob1

```bash
TOKEN=$(cat /tmp/alice1-cookies.txt | grep access_token | awk '{print $7}')
curl -s -X GET "http://localhost:8083/auth/search?login=bob1" \
  -H "Cookie: access_token=$TOKEN" \
  | tee /tmp/bob1-search.json | jq . && \
  jq -r '.users[0].id' /tmp/bob1-search.json > /tmp/bob1-id.txt && \
  echo "✓ bob1 найден: $(cat /tmp/bob1-id.txt)"
```

---

## Шаг 6: [ТЕРМИНАЛ 1] alice1 подключается к WebSocket

```bash
TOKEN=$(cat /tmp/alice1-cookies.txt | grep access_token | awk '{print $7}')
echo "=== alice1 подключается к WebSocket ==="
echo "alice1 ID: $(cat /tmp/alice1-id.txt)"
echo "bob1 ID: $(cat /tmp/bob1-id.txt)"
echo ""
websocat -H="Cookie: access_token=$TOKEN" ws://localhost:8083/ws
```

**Ожидаемый ответ:**
```json
{"type":"status","payload":{"connected":true,"user_id":"..."}}
```

---

## Шаг 7: [ТЕРМИНАЛ 2] bob1 подключается к WebSocket

```bash
TOKEN=$(cat /tmp/bob1-cookies.txt | grep access_token | awk '{print $7}')
websocat -H="Cookie: access_token=$TOKEN" ws://localhost:8083/ws
```

**Ожидаемый ответ:**
```json
{"type":"status","payload":{"connected":true,"user_id":"..."}}
```

---

## Шаг 8: [ТЕРМИНАЛ 1] alice1 отправляет сообщение bob1

Скопируй эту команду целиком и вставь в WebSocket терминал alice1:

```bash
{"type":"send_message","recipient_id":"962ba316-4354-496a-8801-c6e270aa633a","content":"Hello bob1!","message_type":"text"}
```

**Ожидаемый ответ в ТЕРМИНАЛЕ 1:**
```json
{"type":"message_sent","payload":{"id":"MSG_ID_1","chat_id":"CHAT_ID",...}}
```

**Скопируй MSG_ID_1 и CHAT_ID из ответа!**

**Ожидаемый ответ в ТЕРМИНАЛЕ 2 (автоматически):**
```json
{"type":"new_message","payload":{"message_id":"MSG_ID_1","sender_id":"ALICE1_ID","content":"Hello bob1!",...}}
```

---

## Шаг 9: [ТЕРМИНАЛ 2] bob1 отмечает сообщение прочитанным

Вставь в WebSocket терминал bob1 (замени CHAT_ID и MSG_ID_1 на реальные значения из шага 8):

```bash
{"type":"mark_as_read","chat_id":"CHAT_ID","last_message_id":"MSG_ID_1"}
```

**Ожидаемый ответ в ТЕРМИНАЛЕ 2:**
```json
{"type":"marked_as_read"}
```

**Ожидаемый ответ в ТЕРМИНАЛЕ 1 (автоматически):**
```json
{"type":"message_read","payload":{"chat_id":"CHAT_ID","last_message_id":"MSG_ID_1",...}}
```

---

## Шаг 10: [ТЕРМИНАЛ 2] bob1 отправляет ответ alice1

Скопируй эту команду целиком и вставь в WebSocket терминал bob1:

```bash
{"type":"send_message","recipient_id":"$(cat /tmp/alice1-id.txt)","content":"Hi alice1! Nice to meet you","message_type":"text"}
```

**Ожидаемый ответ в ТЕРМИНАЛЕ 2:**
```json
{"type":"message_sent","payload":{"id":"MSG_ID_2",...}}
```

**Скопируй MSG_ID_2 из ответа!**

**Ожидаемый ответ в ТЕРМИНАЛЕ 1 (автоматически):**
```json
{"type":"new_message","payload":{"message_id":"MSG_ID_2","sender_id":"BOB1_ID","content":"Hi alice1! Nice to meet you",...}}
```

---

## Шаг 11: [ТЕРМИНАЛ 1] alice1 отмечает сообщение прочитанным

Вставь в WebSocket терминал alice1 (замени CHAT_ID и MSG_ID_2 на реальные значения из шага 10):

```bash
{"type":"mark_as_read","chat_id":"CHAT_ID","last_message_id":"MSG_ID_2"}
```

**Оба пользователя получат уведомление о прочтении.**

---

## Шаг 12: [ТЕРМИНАЛ 1] alice1 изменяет свое сообщение

Вставь в WebSocket терминал alice1 (замени MSG_ID_1 на реальное значение из шага 8):

```bash
{"type":"alter_message","message_id":"MSG_ID_1","new_content":"Hello bob1! (edited)"}
```

**Ожидаемый ответ в ТЕРМИНАЛЕ 1:**
```json
{"type":"message_altered"}
```

**Ожидаемый ответ в ТЕРМИНАЛЕ 2 (автоматически):**
```json
{"type":"message_altered","payload":{"message_id":"MSG_ID_1","new_content":"Hello bob1! (edited)",...}}
```

---

## Шаг 13: [ТЕРМИНАЛ 2] bob1 удаляет свое сообщение

Вставь в WebSocket терминал bob1 (замени MSG_ID_2 на реальное значение из шага 10):

```bash
{"type":"delete_message","message_id":"MSG_ID_2"}
```

**Ожидаемый ответ в ТЕРМИНАЛЕ 2:**
```json
{"type":"message_deleted"}
```

**Ожидаемый ответ в ТЕРМИНАЛЕ 1 (автоматически):**
```json
{"type":"message_deleted","payload":{"message_id":"MSG_ID_2",...}}
```

---

## Шаг 14: [ТЕРМИНАЛ 1] alice1 отправляет еще одно сообщение

Скопируй эту команду целиком и вставь в WebSocket терминал alice1:

```bash
{"type":"send_message","recipient_id":"$(cat /tmp/bob1-id.txt)","content":"Another message","message_type":"text"}
```

**bob1 получит его автоматически в ТЕРМИНАЛЕ 2.**

---

## Шаг 15: [ТЕРМИНАЛ 2] bob1 делает disconnect

Нажми `Ctrl+C` в терминале bob1 для отключения.

---

## Шаг 16: [ТЕРМИНАЛ 2] bob1 делает reconnect

```bash
TOKEN=$(cat /tmp/bob1-cookies.txt | grep access_token | awk '{print $7}')
websocat -H="Cookie: access_token=$TOKEN" ws://localhost:8083/ws
```

---

## Шаг 17: [ТЕРМИНАЛ 2] bob1 получает историю сообщений

Вставь в WebSocket терминал bob1 (замени CHAT_ID на реальное значение):

```bash
{"type":"get_messages","chat_id":"CHAT_ID","limit":50}
```

**Ожидаемый ответ:**
```json
{"type":"messages","payload":{"messages":[...],"has_more":false}}
```

Увидишь все сообщения включая последнее от alice1.

---

## Дополнительные команды

### Получить список чатов (любой терминал)
```json
{"type":"get_chats"}
```

### Проверить сохраненные ID
```bash
echo "alice1 ID: $(cat /tmp/alice1-id.txt)"
echo "bob1 ID: $(cat /tmp/bob1-id.txt)"
```

### Посмотреть логи
```bash
docker compose logs -f realtime-service
docker compose logs -f message-service
```

### Очистить данные
```bash
rm -f /tmp/alice1-* /tmp/bob1-*
docker compose down -v
```

---

## Формат CHAT_ID

Chat ID формируется автоматически как два UUID отсортированных по алфавиту через двоеточие:

```
<меньший_UUID>:<больший_UUID>
```

Пример:
```
70a43e76-e48a-49e8-bfbf-c489073154bf:a00fb982-d0aa-4d0f-aad1-2eefeeffbfcc
```

---

## Типичные ошибки

❌ **"failed to send message"**
- Проверь что recipient_id это валидный UUID
- Убедись что получатель зарегистрирован

❌ **"invalid request"**
- Проверь JSON синтаксис (кавычки, запятые)
- Все поля должны быть в двойных кавычках

❌ **"token revoked"**
- Перелогинься (шаги 3-4)

❌ **Connection refused**
- Запусти Docker: `docker compose up -d`

❌ **"users": []** при поиске
- Пользователь еще не зарегистрирован
- Проверь правильность логина

---

## Структура всех WebSocket событий

### Исходящие (от клиента к серверу):

| Команда | Описание | Обязательные поля |
|---------|----------|-------------------|
| `send_message` | Отправить сообщение | `recipient_id`, `content`, `message_type` |
| `get_messages` | Получить историю | `chat_id`, `limit` |
| `mark_as_read` | Отметить прочитанным | `chat_id`, `last_message_id` |
| `alter_message` | Изменить сообщение | `message_id`, `new_content` |
| `delete_message` | Удалить сообщение | `message_id` |
| `get_chats` | Список чатов | - |

### Входящие (от сервера к клиенту):

| Событие | Когда приходит | Автоматически |
|---------|----------------|---------------|
| `status` | При подключении | Да |
| `message_sent` | После отправки сообщения | Нет |
| `new_message` | Когда кто-то отправил тебе сообщение | Да |
| `message_read` | Когда твое сообщение прочитали | Да |
| `message_altered` | Когда сообщение изменено | Да |
| `message_deleted` | Когда сообщение удалено | Да |
| `messages` | Ответ на get_messages | Нет |
| `chats` | Ответ на get_chats | Нет |
| `marked_as_read` | Подтверждение mark_as_read | Нет |
| `error` | При ошибке | Да |
