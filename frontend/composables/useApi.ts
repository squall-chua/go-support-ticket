import { useQuery, useMutation, type UseQueryOptions, type UseMutationOptions } from '@tanstack/vue-query'

const API_BASE_URL = 'http://localhost:8080'

export const useApi = () => {
  const fetchWithAuth = async (endpoint: string, options: RequestInit = {}) => {
    // Basic wrapper over fetch
    const headers = {
      ...options.headers,
      'Content-Type': 'application/json',
      // 'Authorization': 'Bearer ' + token, // TODO: Auth if implemented
    }

    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers,
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.message || `API error: ${response.status}`)
    }

    return response.json()
  }

  // Generic query composable
  const useApiQuery = <TData = unknown, TError = Error>(
    queryKey: any[],
    endpoint: string,
    options?: Omit<UseQueryOptions<TData, TError>, 'queryKey' | 'queryFn'>
  ) => {
    return useQuery<TData, TError>({
      queryKey,
      queryFn: () => fetchWithAuth(endpoint),
      ...options,
    })
  }

  // Generic mutation composable
  const useApiMutation = <TData = unknown, TError = Error, TVariables = unknown>(
    endpoint: string,
    method: 'POST' | 'PUT' | 'PATCH' | 'DELETE' = 'POST',
    options?: UseMutationOptions<TData, TError, TVariables>
  ) => {
    return useMutation<TData, TError, TVariables>({
      mutationFn: (variables: TVariables) =>
        fetchWithAuth(endpoint, {
          method,
          body: JSON.stringify(variables),
        }),
      ...options,
    })
  }

  return {
    fetchWithAuth,
    useApiQuery,
    useApiMutation,
  }
}
