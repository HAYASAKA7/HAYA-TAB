/**
 * Escapes HTML special characters to prevent XSS attacks
 * @param text - The text to escape
 * @returns The escaped text safe for HTML rendering
 */
export function escapeHtml(text: string): string {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

/**
 * Escapes HTML in template strings while preserving specific safe HTML tags
 * Use this when you need to include line breaks or basic formatting in user-facing messages
 * @param strings - Template string parts
 * @param values - Values to be escaped and interpolated
 * @returns Safe HTML string
 */
export function safeHtml(strings: TemplateStringsArray, ...values: any[]): string {
  let result = strings[0]
  for (let i = 0; i < values.length; i++) {
    // Escape the value to prevent XSS
    const escaped = typeof values[i] === 'string' ? escapeHtml(values[i]) : String(values[i])
    result += escaped + strings[i + 1]
  }
  return result
}
