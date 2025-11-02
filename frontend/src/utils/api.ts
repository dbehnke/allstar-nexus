/**
 * API utility for making authenticated requests to the backend
 */

interface RequestOptions {
  method?: string
  headers?: Record<string, string>
  body?: string
}

/**
 * Make an authenticated API request
 * @param url - The API endpoint URL
 * @param options - Fetch options (method, headers, body)
 * @returns Promise with the parsed JSON response
 */
export async function apiRequest<T = any>(
  url: string,
  options: RequestOptions = {}
): Promise<T> {
  const token = localStorage.getItem('token')

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...options.headers
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const fetchOptions: RequestInit = {
    method: options.method || 'GET',
    headers,
    ...(options.body && { body: options.body })
  }

  const response = await fetch(url, fetchOptions)

  if (!response.ok) {
    let errorMessage = `HTTP ${response.status}: ${response.statusText}`

    try {
      const errorData = await response.json()
      if (errorData.error) {
        errorMessage = errorData.error
      } else if (errorData.message) {
        errorMessage = errorData.message
      }
    } catch {
      // Ignore JSON parse errors, use default error message
    }

    throw new Error(errorMessage)
  }

  const data = await response.json()
  return data as T
}
