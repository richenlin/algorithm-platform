import axios from 'axios'
import type { AxiosInstance } from 'axios'

export const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

api.interceptors.response.use(
  response => response,
  error => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

export default api
