import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function sendMessage(sessionId, content, webSearch = false, knowledgeBases = []) {
  return http.post('/api/chat/send', { session_id: sessionId, content, web_search: webSearch, knowledge_bases: knowledgeBases })
}

export function stopMessage(sessionId) {
  return http.post('/api/chat/stop', { session_id: sessionId })
}

export function getSessions() {
  return http.get('/api/chat/sessions')
}

export function getSessionMessages(sessionId, before, limit) {
  const params = {}
  if (before) params.before = before
  if (limit) params.limit = limit
  return http.get(`/api/chat/sessions/${sessionId}`, { params })
}

export function createSession() {
  return http.post('/api/chat/sessions')
}

export function deleteSession(sessionId) {
  return http.delete(`/api/chat/sessions/${sessionId}`)
}

export function startTopic(sessionId) {
  return http.put(`/api/chat/sessions/${sessionId}/topic`)
}

export function uploadChatImage(file) {
  const formData = new FormData()
  formData.append('file', file)
  return http.post('/api/chat/upload', formData)
}
