import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function createSchedule(data) {
  return http.post('/api/schedules', data)
}

export function getSchedules() {
  return http.get('/api/schedules')
}

export function updateSchedule(id, data) {
  return http.put(`/api/schedules/${id}`, data)
}

export function deleteSchedule(id) {
  return http.delete(`/api/schedules/${id}`)
}

export function getScheduleTypes() {
  return http.get('/api/schedules/types')
}

export function getScheduleLogs(id) {
  return http.get(`/api/schedules/${id}/logs`)
}
