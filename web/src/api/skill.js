import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function getSkills() {
  return http.get('/api/skills')
}

export function createSkill(data) {
  return http.post('/api/skills', data)
}

export function updateSkill(id, data) {
  return http.put(`/api/skills/${id}`, data)
}

export function toggleSkill(id, enabled) {
  return http.put(`/api/skills/${id}/toggle`, { enabled })
}

export function deleteSkill(id) {
  return http.delete(`/api/skills/${id}`)
}
