import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function getMemories() {
  return http.get('/api/memories')
}

export function deleteMemory(id) {
  return http.delete(`/api/memories/${id}`)
}
