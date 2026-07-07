import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function createTodo(data) {
  return http.post('/api/todos', data)
}

export function getTodos(planId) {
  const params = {}
  if (planId) params.plan_id = planId
  return http.get('/api/todos', { params })
}

export function updateTodo(id, data) {
  return http.put(`/api/todos/${id}`, data)
}

export function deleteTodo(id) {
  return http.delete(`/api/todos/${id}`)
}
