/**
 * Date formatting utilities
 * Handles date parsing and formatting with Safari compatibility
 */

/**
 * Safely parse date string (handles Safari issues with ISO dates)
 */
export function parseDate(dateString: string | number): Date {
  if (!dateString) {
    return new Date();
  }

  // If it's a number (Unix timestamp in seconds or milliseconds)
  if (typeof dateString === 'number') {
    const timestamp = dateString < 10000000000 ? dateString * 1000 : dateString;
    return new Date(timestamp);
  }

  // Safari compatibility: normalize RFC3339 format
  // Convert: 2024-01-01T12:00:00+00:00 -> 2024-01-01T12:00:00Z
  // Convert: 2024-01-01T12:00:00-00:00 -> 2024-01-01T12:00:00Z
  let normalized = dateString.trim();

  // Replace UTC offset with Z
  normalized = normalized.replace(/([+-])00:00$/, 'Z');

  // If no timezone indicator at all, add Z
  const hasTimezone = normalized.endsWith('Z') ||
                      /[+-]\d{2}:\d{2}$/.test(normalized) ||
                      /[+-]\d{4}$/.test(normalized);

  if (!hasTimezone && normalized.includes('T')) {
    normalized = normalized + 'Z';
  }

  const date = new Date(normalized);

  // Fallback to current date if parsing failed
  if (isNaN(date.getTime())) {
    console.error('[dateUtils] Failed to parse date:', dateString);
    return new Date();
  }

  return date;
}

/**
 * Format time (HH:MM)
 */
export function formatTime(dateString: string | number): string {
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
export function formatDateSeparator(dateString: string | number): string {
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
    // Fallback to manual formatting
    const months = ['January', 'February', 'March', 'April', 'May', 'June',
                    'July', 'August', 'September', 'October', 'November', 'December'];
    const month = months[date.getMonth()];
    const day = date.getDate();
    const year = date.getFullYear();
    const currentYear = new Date().getFullYear();

    return year !== currentYear ? `${month} ${day}, ${year}` : `${month} ${day}`;
  }
}

/**
 * Get date key for grouping messages by day
 */
export function getDateKey(dateString: string): string {
  const date = parseDate(dateString);
  return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
}
