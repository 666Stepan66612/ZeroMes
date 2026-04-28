/**
 * Date formatting utilities
 * Handles date parsing and formatting with Safari compatibility
 */

/**
 * Safely parse date string (handles Safari issues with ISO dates)
 */
export function parseDate(dateString: string): Date {
  // Safari has issues with ISO dates without 'Z' or timezone
  // Ensure proper format
  if (!dateString) return new Date();

  const date = new Date(dateString);

  // Check if date is valid
  if (isNaN(date.getTime())) {
    console.warn('[dateUtils] Invalid date:', dateString);
    return new Date();
  }

  return date;
}

/**
 * Format time (HH:MM)
 */
export function formatTime(dateString: string): string {
  const date = parseDate(dateString);

  try {
    return date.toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch (error) {
    console.error('[dateUtils] Error formatting time:', error);
    // Fallback to manual formatting
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
  }
}

/**
 * Check if two dates are on the same day
 */
export function isSameDay(date1: Date, date2: Date): boolean {
  return (
    date1.getFullYear() === date2.getFullYear() &&
    date1.getMonth() === date2.getMonth() &&
    date1.getDate() === date2.getDate()
  );
}

/**
 * Check if date is today
 */
export function isToday(date: Date): boolean {
  return isSameDay(date, new Date());
}

/**
 * Check if date is yesterday
 */
export function isYesterday(date: Date): boolean {
  const yesterday = new Date();
  yesterday.setDate(yesterday.getDate() - 1);
  return isSameDay(date, yesterday);
}

/**
 * Format date separator (Today, Yesterday, or date)
 */
export function formatDateSeparator(dateString: string): string {
  const date = parseDate(dateString);

  if (isToday(date)) {
    return 'Today';
  }

  if (isYesterday(date)) {
    return 'Yesterday';
  }

  // For older dates, show full date
  try {
    return date.toLocaleDateString([], {
      month: 'long',
      day: 'numeric',
      year: date.getFullYear() !== new Date().getFullYear() ? 'numeric' : undefined,
    });
  } catch (error) {
    console.error('[dateUtils] Error formatting date separator:', error);
    // Fallback to ISO date
    return date.toISOString().split('T')[0];
  }
}

/**
 * Get date key for grouping messages by day
 */
export function getDateKey(dateString: string): string {
  const date = parseDate(dateString);
  return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
}
