# Инструкция по тестированию после исправлений

## 🔧 Что было исправлено

1. ✅ PBKDF2 с солью для генерации ключей
2. ✅ Убрано хранение ключа в `window`
3. ✅ Исправлен SQL баг в auth-service (параметры UPDATE)
4. ✅ Исправлена инициализация outboxRepo

## ⚠️ BREAKING CHANGES

Из-за изменения алгоритма генерации ключей (добавлена соль), **старые пользователи не смогут войти**.

### Решение для локального тестирования:

**Вариант 1: Очистить базу данных (рекомендуется)**
```bash
# Остановить все сервисы
docker-compose down

# Удалить volumes с данными
docker volume rm newmessenger_postgres_data
docker volume rm newmessenger_postgres_messages_data

# Запустить заново
docker-compose up -d
```

**Вариант 2: Пересоздать пользователя**
1. Удалить пользователя из БД вручную
2. Зарегистрироваться заново

## 🚀 Перезапуск сервисов

После исправления SQL бага нужно пересобрать auth-service:

```bash
# Пересобрать и перезапустить все сервисы
docker-compose up -d --build

# ИЛИ только auth-service
docker-compose up -d --build auth-service

# Проверить логи
docker-compose logs -f auth-service
```

## ✅ Тестирование

### 1. Регистрация нового пользователя
```
1. Открыть http://localhost/register
2. Ввести login: testuser
3. Ввести password: testpassword123
4. Зарегистрироваться
```

### 2. Вход
```
1. Открыть http://localhost/login
2. Ввести те же credentials
3. Проверить, что вход успешен
```

### 3. Отправка сообщений
```
1. Найти другого пользователя через поиск
2. Отправить сообщение
3. Проверить, что сообщение зашифровано и доставлено
```

### 4. Смена пароля (НОВОЕ)
```
1. Перейти в Settings → Change Password
2. Ввести старый пароль: testpassword123
3. Ввести новый пароль: newpassword456
4. Подтвердить новый пароль
5. Нажать "Change Password"
6. Проверить, что:
   - Пароль изменен успешно
   - Можно войти с новым паролем
   - Старые сообщения расшифровываются
```

### 5. Проверка безопасности

**Проверить, что ключ НЕ в window:**
```javascript
// Открыть DevTools Console
console.log(window.encryptionKey); // должно быть undefined
console.log(Object.keys(window).filter(k => k.includes('key'))); // не должно быть ключей
```

**Проверить, что ключ в замыкании (недоступен):**
```javascript
// Это НЕ должно работать
try {
  const module = await import('/src/lib/crypto/secureStorage.ts');
  console.log(module.cachedPrivateKey); // undefined (приватная переменная)
} catch (e) {
  console.log('Ключ недоступен извне ✅');
}
```

## 🐛 Известные проблемы

### Проблема: "Failed to re-encrypt key for chat"
**Причина**: Старые ключи чатов были зашифрованы старым алгоритмом (без соли).

**Решение**: 
- Для тестирования: Очистить БД и создать новые чаты
- Для production: Нужна миграция (пересоздание ключей чатов)

### Проблема: "mismatched param and argument count"
**Причина**: Был баг в SQL запросе (исправлен).

**Решение**: Пересобрать auth-service:
```bash
docker-compose up -d --build auth-service
```

## 📊 Проверка логов

```bash
# Auth service
docker-compose logs -f auth-service | grep -i error

# Message service
docker-compose logs -f message-service | grep -i error

# API Gateway
docker-compose logs -f api-gateway | grep -i error

# Realtime service
docker-compose logs -f realtime-service | grep -i error
```

## 🔐 Проверка безопасности

### XSS тест (симуляция)
```javascript
// В DevTools Console попробовать получить ключ
console.log('=== XSS Test ===');
console.log('window.encryptionKey:', window.encryptionKey); // undefined ✅
console.log('globalThis.encryptionKey:', globalThis.encryptionKey); // undefined ✅

// Попробовать найти ключ в window
const suspiciousKeys = Object.keys(window).filter(k => 
  k.toLowerCase().includes('key') || 
  k.toLowerCase().includes('private') ||
  k.toLowerCase().includes('secret')
);
console.log('Suspicious keys in window:', suspiciousKeys); // должно быть пусто ✅
```

### Проверка PBKDF2
```javascript
// Проверить, что разные пользователи с одинаковым паролем имеют разные ключи
// (это можно проверить только на бэкенде, сравнив auth_hash в БД)
```

## 📝 Чеклист тестирования

- [ ] Очистил БД или удалил старых пользователей
- [ ] Пересобрал auth-service (`docker-compose up -d --build auth-service`)
- [ ] Зарегистрировал нового пользователя
- [ ] Вошел в систему
- [ ] Отправил сообщение
- [ ] Проверил, что `window.encryptionKey` === undefined
- [ ] Сменил пароль
- [ ] Вышел и вошел с новым паролем
- [ ] Старые сообщения расшифровываются
- [ ] Нет ошибок в логах сервисов

## 🎯 Ожидаемый результат

После всех исправлений:
- ✅ Регистрация работает
- ✅ Вход работает
- ✅ Сообщения шифруются и расшифровываются
- ✅ Смена пароля работает
- ✅ Ключ НЕ доступен через `window`
- ✅ Разные пользователи с одинаковым паролем имеют разные ключи
- ✅ Нет SQL ошибок

---

**Дата**: 2026-04-07  
**Версия**: После критических исправлений безопасности
