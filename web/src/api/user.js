import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function register(username, password) {
  return http.post('/api/register', { username, password })
}

export function login(username, password) {
  return http.post('/api/login', { username, password })
}

export function getMe() {
  return http.get('/api/me')
}

export function logout() {
  return http.post('/api/logout')
}
