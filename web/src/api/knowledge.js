import axios from 'axios'

const http = axios.create({
  withCredentials: true,
})

export function createKB(name, description) {
  return http.post('/api/knowledge-bases', { name, description })
}

export function updateKB(name, description) {
  return http.put(`/api/knowledge-bases/${encodeURIComponent(name)}`, { description })
}

export function getKBList() {
  return http.get('/api/knowledge-bases')
}

export function deleteKB(name) {
  return http.delete(`/api/knowledge-bases/${encodeURIComponent(name)}`)
}

export function getDocs(kbName) {
  return http.get(`/api/knowledge-bases/${encodeURIComponent(kbName)}/docs`)
}

export function uploadDoc(kbName, file) {
  const formData = new FormData()
  formData.append('file', file)
  return http.post(`/api/knowledge-bases/${encodeURIComponent(kbName)}/docs`, formData)
}

export function deleteDoc(kbName, docName) {
  return http.delete(`/api/knowledge-bases/${encodeURIComponent(kbName)}/docs/${encodeURIComponent(docName)}`)
}

// 预览文件的同源 URL，供 <img>/<iframe> 直接引用
export function previewDocURL(kbName, docName) {
  return `/api/knowledge-bases/${encodeURIComponent(kbName)}/docs/${encodeURIComponent(docName)}`
}

// 以纯文本拉取文件内容（避免被当 JSON 解析），供文本/Markdown 预览
export function fetchDocText(kbName, docName) {
  return http.get(previewDocURL(kbName, docName), {
    responseType: 'text',
    transformResponse: [data => data],
  })
}
