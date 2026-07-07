import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function executeAiTask(prompt, toolName) {
  return http.post('/api/ai/task', { prompt, tool_name: toolName })
}
