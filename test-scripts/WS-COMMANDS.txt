# Генератор команд для WebSocket

Используй эти команды чтобы сгенерировать JSON с автоподстановкой UUID.

## Для alice1 (ТЕРМИНАЛ 1)

### Отправить сообщение bob1
```bash
echo "{\"type\":\"send_message\",\"recipient_id\":\"$(cat /tmp/bob1-id.txt)\",\"content\":\"Hello bob1!\",\"message_type\":\"text\"}"
```

### Отправить еще сообщение bob1
```bash
echo "{\"type\":\"send_message\",\"recipient_id\":\"$(cat /tmp/bob1-id.txt)\",\"content\":\"Another message\",\"message_type\":\"text\"}"
```

---

## Для bob1 (ТЕРМИНАЛ 2)

### Отправить сообщение alice1
```bash
echo "{\"type\":\"send_message\",\"recipient_id\":\"$(cat /tmp/alice1-id.txt)\",\"content\":\"Hi alice1! Nice to meet you\",\"message_type\":\"text\"}"
```

---

## Универсальные команды (для любого терминала)

### Отметить как прочитанное (замени CHAT_ID и MSG_ID)
```bash
echo '{"type":"mark_as_read","chat_id":"CHAT_ID","last_message_id":"MSG_ID"}'
```

### Изменить сообщение (замени MSG_ID)
```bash
echo '{"type":"alter_message","message_id":"MSG_ID","new_content":"Hello bob1! (edited)"}'
```

### Удалить сообщение (замени MSG_ID)
```bash
echo '{"type":"delete_message","message_id":"MSG_ID"}'
```

### Получить историю (замени CHAT_ID)
```bash
echo '{"type":"get_messages","chat_id":"CHAT_ID","limit":50}'
```

### Получить список чатов
```bash
echo '{"type":"get_chats"}'
```

---

## Как использовать:

1. Открой **3 терминала**:
   - Терминал 1: WebSocket alice1
   - Терминал 2: WebSocket bob1
   - Терминал 3: Генератор команд

2. В терминале 3 выполни команду генерации
3. Скопируй вывод
4. Вставь в нужный WebSocket терминал (1 или 2)

---

## Проверка сохраненных ID

```bash
echo "alice1 ID: $(cat /tmp/alice1-id.txt)"
echo "bob1 ID: $(cat /tmp/bob1-id.txt)"
```
