import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function createPlan(title, content, status) {
  return http.post('/api/plans', { title, content, status })
}

export function getPlans() {
  return http.get('/api/plans')
}

export function getPlan(id) {
  return http.get(`/api/plans/${id}`)
}

export function updatePlan(id, title, content, status) {
  return http.put(`/api/plans/${id}`, { title, content, status })
}

export function deletePlan(id) {
  return http.delete(`/api/plans/${id}`)
}
