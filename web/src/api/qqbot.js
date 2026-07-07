import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function startQQBotBind(appId, appSecret) {
  return http.post('/api/qqbot/bind', { app_id: appId, app_secret: appSecret })
}

export function checkQQBotBind() {
  return http.get('/api/qqbot/bind/check')
}

export function getQQBotStatus() {
  return http.get('/api/qqbot/status')
}

export function toggleQQBotChat(enabled) {
  return http.post('/api/qqbot/chat/toggle', { enabled })
}

export function getQQBotChatStatus() {
  return http.get('/api/qqbot/chat/status')
}
