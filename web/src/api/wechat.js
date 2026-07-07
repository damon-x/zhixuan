import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function getWeChatQRCode() {
  return http.post('/api/wechat/qrcode')
}

export function checkWeChatBind() {
  return http.get('/api/wechat/bind/status')
}

export function getWeChatStatus() {
  return http.get('/api/wechat/status')
}

export function toggleWeChatChat(enabled) {
  return http.post('/api/wechat/chat/toggle', { enabled })
}

export function getWeChatChatStatus() {
  return http.get('/api/wechat/chat/status')
}
