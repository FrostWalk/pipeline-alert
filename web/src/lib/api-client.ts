import { client } from '@/client/client.gen'
import { useAuthStore } from './auth-store'

client.interceptors.request.use((request) => {
  const token = useAuthStore.getState().token
  if (token) {
    const headers = new Headers(request.headers)
    headers.set('Authorization', `Bearer ${token}`)
    return new Request(request, { headers })
  }
  return request
})

export { client }
