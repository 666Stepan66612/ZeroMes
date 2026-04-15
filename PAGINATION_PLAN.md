# План реализации пагинации сообщений

## Текущее состояние
- Загружаются последние 50 сообщений при открытии чата
- Бэкенд возвращает в порядке DESC (новые → старые)
- Фронтенд разворачивает через `.reverse()` (старые → новые)

## Что нужно добавить

### 1. Новые state переменные
```typescript
const [hasMore, setHasMore] = useState(true);
const [loadingMore, setLoadingMore] = useState(false);
const messagesContainerRef = useRef<HTMLDivElement>(null);
```

### 2. Функция loadMoreMessages
```typescript
const loadMoreMessages = async () => {
  if (!chat.id || !chatKey || !hasMore || loadingMore) return;
  
  // Берем ID самого СТАРОГО сообщения (первое в массиве после reverse)
  const oldestMessage = messages[0];
  if (!oldestMessage) return;
  
  try {
    setLoadingMore(true);
    
    // Сохраняем текущую высоту скролла
    const container = messagesContainerRef.current;
    const oldScrollHeight = container?.scrollHeight || 0;
    
    const response = await getMessages({
      chat_id: chat.id,
      limit: 50,
      last_message_id: oldestMessage.id  // Бэкенд вернет 50 сообщений СТАРШЕ этого
    });
    
    if (response.messages.length === 0) {
      setHasMore(false);
      return;
    }
    
    // Расшифровываем
    const decryptedMessages = await Promise.all(
      response.messages.map(async (msg) => {
        // ... та же логика расшифровки
      })
    );
    
    // ВАЖНО: reverse() чтобы старые были в начале
    const reversedMessages = decryptedMessages.reverse();
    
    // Добавляем В НАЧАЛО массива
    setMessages(prev => [...reversedMessages, ...prev]);
    
    // Восстанавливаем позицию скролла
    setTimeout(() => {
      if (container) {
        const newScrollHeight = container.scrollHeight;
        container.scrollTop = newScrollHeight - oldScrollHeight;
      }
    }, 0);
    
    // Проверяем есть ли еще сообщения
    setHasMore(response.has_more);
    
  } catch (error) {
    console.error('Failed to load more messages:', error);
  } finally {
    setLoadingMore(false);
  }
};
```

### 3. Обработчик скролла
```typescript
const handleScroll = () => {
  const container = messagesContainerRef.current;
  if (!container) return;
  
  // Если скролл в самом верху (или близко к верху)
  if (container.scrollTop < 100) {
    loadMoreMessages();
  }
};

useEffect(() => {
  const container = messagesContainerRef.current;
  if (!container) return;
  
  container.addEventListener('scroll', handleScroll);
  return () => container.removeEventListener('scroll', handleScroll);
}, [messages, hasMore, loadingMore]);
```

### 4. Изменения в JSX
```tsx
<div 
  ref={messagesContainerRef}
  className="messages-container"
  style={{ overflowY: 'auto', height: '100%' }}
>
  {loadingMore && (
    <div className="loading-more">Загрузка...</div>
  )}
  
  {messages.map((msg) => (
    // ... рендер сообщений
  ))}
  
  <div ref={messagesEndRef} />
</div>
```

### 5. Сброс hasMore при смене чата
```typescript
useEffect(() => {
  setHasMore(true);
  setMessages([]);
}, [chat.id]);
```

## Важные моменты

1. **ID для пагинации**: Берем `messages[0].id` (самое старое после reverse)
2. **Реверс**: Новые сообщения от бэкенда тоже нужно `.reverse()` перед добавлением
3. **Порядок добавления**: `[...newMessages, ...oldMessages]` - новые В НАЧАЛО
4. **Позиция скролла**: Сохраняем и восстанавливаем, чтобы не прыгало
5. **has_more**: Бэкенд должен возвращать этот флаг в ответе

## Проверка бэкенда

Нужно убедиться что бэкенд поддерживает `last_message_id` параметр и возвращает `has_more`.

Смотри [`postgres.go`](message-service/internal/messaging/repository/postgres.go:70-78) - там уже есть логика с `lastMessageID`.
