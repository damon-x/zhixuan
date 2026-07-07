import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function createNote(title, content, planId) {
  const data = { title, content }
  if (planId) {
    data.plan_id = planId
  }
  return http.post('/api/notes', data)
}

export function getNotes(keyword, planId) {
  const params = {}
  if (keyword) params.keyword = keyword
  if (planId) params.plan_id = planId
  return http.get('/api/notes', { params })
}

export function getNote(id) {
  return http.get(`/api/notes/${id}`)
}

export function updateNote(id, title, content) {
  return http.put(`/api/notes/${id}`, { title, content })
}

export function deleteNote(id) {
  return http.delete(`/api/notes/${id}`)
}
