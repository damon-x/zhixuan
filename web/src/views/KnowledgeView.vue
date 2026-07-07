<template>
  <MasterDetail :has-selection="!!currentKB" @back="currentKB = ''">
    <template #list>
      <div class="list-header">
        <span class="list-title">知识库</span>
        <el-button size="small" @click="showCreateDialog">新建</el-button>
      </div>
      <div class="list-scroll">
        <div v-if="kbList.length === 0" class="list-empty">暂无知识库</div>
        <div
          v-for="kb in kbList"
          :key="kb.name"
          class="kb-item"
          :class="{ active: currentKB === kb.name }"
          @click="enterKB(kb.name)"
        >
          <div class="kb-item-main">
            <div class="kb-item-name">{{ kb.name }}</div>
            <div v-if="kb.description" class="kb-item-desc">{{ kb.description }}</div>
            <div class="kb-item-meta">{{ kb.doc_count }} 个文档</div>
          </div>
          <div class="kb-item-actions">
            <el-button size="small" @click.stop="showEditDialog(kb)">编辑</el-button>
            <el-button size="small" type="danger" @click.stop="handleDeleteKB(kb.name)">删除</el-button>
          </div>
        </div>
      </div>
    </template>

    <template #detail>
      <div v-if="!currentKB" class="detail-empty">
        <div class="detail-empty-text">选择一个知识库查看文档</div>
      </div>
      <template v-else>
        <div class="detail-header">
          <span class="detail-title">{{ currentKB }}</span>
          <el-upload
            :show-file-list="false"
            :before-upload="handleUpload"
            action=""
          >
            <el-button type="primary" size="small">上传文档</el-button>
          </el-upload>
        </div>
        <div class="detail-body">
          <div v-if="docList.length === 0" class="section-empty">暂无文档，点击上传按钮添加</div>
          <div class="card-list">
            <div v-for="doc in docList" :key="doc.name" class="doc-card" @click="openPreview(doc)">
              <div class="doc-main">
                <div class="doc-name">{{ doc.name }}</div>
                <div class="doc-meta">
                  <span>{{ formatSize(doc.size) }}</span>
                  <span class="doc-time">{{ formatTime(doc.updated_at) }}</span>
                </div>
              </div>
              <el-button size="small" type="danger" @click.stop="handleDeleteDoc(doc.name)">删除</el-button>
            </div>
          </div>
        </div>
      </template>
    </template>
  </MasterDetail>

  <el-dialog v-model="createDialogVisible" title="新建知识库" width="400px">
    <el-input v-model="newKBName" placeholder="请输入知识库名称" @keyup.enter="handleCreateKB" />
    <el-input
      v-model="newKBDescription"
      type="textarea"
      :rows="3"
      placeholder="知识库简介（可选）"
      style="margin-top: 12px"
    />
    <template #footer>
      <el-button @click="createDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="handleCreateKB">确定</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="editDialogVisible" title="编辑知识库" width="400px">
    <el-input
      v-model="editKBDescription"
      type="textarea"
      :rows="3"
      placeholder="知识库简介"
    />
    <template #footer>
      <el-button @click="editDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="handleUpdateKB">确定</el-button>
    </template>
  </el-dialog>

  <el-dialog
    v-model="previewVisible"
    :title="previewDoc ? previewDoc.name : '预览'"
    width="900px"
    top="5vh"
    destroy-on-close
    class="preview-dialog"
  >
    <div v-if="previewDoc" class="preview-body">
      <!-- 图片 -->
      <div v-if="previewKind === 'image'" class="preview-image-wrap">
        <img :src="previewUrl" class="preview-image" :alt="previewDoc.name" />
      </div>
      <!-- PDF -->
      <iframe v-else-if="previewKind === 'pdf'" :src="previewUrl" class="preview-frame"></iframe>
      <!-- HTML（sandbox 仅开放 allow-scripts，不开 allow-same-origin：脚本可跑但 iframe 为隔离源，碰不到父页面会话） -->
      <iframe v-else-if="previewKind === 'html'" :src="previewUrl" sandbox="allow-scripts" class="preview-frame"></iframe>
      <!-- 文本 / Markdown -->
      <div v-else-if="previewKind === 'text'" class="preview-text-wrap" v-loading="textLoading">
        <div v-if="textTruncated" class="preview-truncate-tip">文件较大，仅预览前 512KB</div>
        <div v-if="previewExt === 'md'" class="preview-md" v-html="renderedMarkdown"></div>
        <pre v-else class="preview-pre">{{ textContent }}</pre>
      </div>
      <!-- 不支持 -->
      <div v-else class="preview-unsupported">
        <div>该格式暂不支持在线预览</div>
        <div class="preview-unsupported-name">{{ previewDoc.name }}</div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, computed, onActivated } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createKB, updateKB, getKBList, deleteKB, getDocs, uploadDoc, deleteDoc, previewDocURL, fetchDocText } from '../api/knowledge'
import { marked } from 'marked'
import MasterDetail from '../components/MasterDetail.vue'

const kbList = ref([])
const currentKB = ref('')
const docList = ref([])
const createDialogVisible = ref(false)
const newKBName = ref('')
const newKBDescription = ref('')
const editDialogVisible = ref(false)
const editKBName = ref('')
const editKBDescription = ref('')

// 预览
const previewVisible = ref(false)
const previewDoc = ref(null)
const previewUrl = ref('')
const previewKind = ref('unsupported')
const previewExt = ref('')
const textContent = ref('')
const textLoading = ref(false)
const textTruncated = ref(false)

const IMAGE_EXTS = new Set(['png', 'jpg', 'jpeg', 'gif', 'webp'])
const TEXT_EXTS = new Set(['txt', 'md', 'log', 'csv', 'json', 'xml', 'yml', 'yaml', 'js', 'ts', 'go', 'py', 'rs', 'java', 'c', 'cpp', 'h', 'sql', 'sh', 'conf', 'ini', 'toml', 'css', 'scss'])
const MAX_TEXT_PREVIEW = 512 * 1024

const renderedMarkdown = computed(() => {
  if (previewExt.value !== 'md') return ''
  try {
    return marked.parse(textContent.value || '')
  } catch {
    return ''
  }
})

function extOf(name) {
  const i = name.lastIndexOf('.')
  return i >= 0 ? name.slice(i + 1).toLowerCase() : ''
}

async function openPreview(doc) {
  previewDoc.value = doc
  previewVisible.value = true
  const ext = extOf(doc.name)
  previewExt.value = ext
  textContent.value = ''
  textTruncated.value = false
  if (IMAGE_EXTS.has(ext)) {
    previewKind.value = 'image'
    previewUrl.value = previewDocURL(currentKB.value, doc.name)
  } else if (ext === 'pdf') {
    previewKind.value = 'pdf'
    previewUrl.value = previewDocURL(currentKB.value, doc.name)
  } else if (ext === 'html' || ext === 'htm') {
    previewKind.value = 'html'
    previewUrl.value = previewDocURL(currentKB.value, doc.name)
  } else if (TEXT_EXTS.has(ext)) {
    previewKind.value = 'text'
    textLoading.value = true
    try {
      const res = await fetchDocText(currentKB.value, doc.name)
      let txt = typeof res.data === 'string' ? res.data : String(res.data ?? '')
      if (txt.length > MAX_TEXT_PREVIEW) {
        textTruncated.value = true
        txt = txt.slice(0, MAX_TEXT_PREVIEW)
      }
      textContent.value = txt
    } catch {
      ElMessage.error('加载文件失败')
    } finally {
      textLoading.value = false
    }
  } else {
    previewKind.value = 'unsupported'
  }
}

onActivated(() => {
  loadKBList()
})

async function loadKBList() {
  try {
    const { data } = await getKBList()
    if (data.code === 0) {
      kbList.value = data.data || []
    }
  } catch {
    ElMessage.error('加载知识库失败')
  }
}

async function loadDocs() {
  try {
    const { data } = await getDocs(currentKB.value)
    if (data.code === 0) {
      docList.value = data.data || []
    }
  } catch {
    ElMessage.error('加载文档失败')
  }
}

function showCreateDialog() {
  newKBName.value = ''
  newKBDescription.value = ''
  createDialogVisible.value = true
}

async function handleCreateKB() {
  if (!newKBName.value.trim()) {
    ElMessage.warning('请输入知识库名称')
    return
  }
  try {
    const { data } = await createKB(newKBName.value.trim(), newKBDescription.value.trim())
    if (data.code === 0) {
      ElMessage.success('创建成功')
      createDialogVisible.value = false
      loadKBList()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('创建失败')
  }
}

function showEditDialog(kb) {
  editKBName.value = kb.name
  editKBDescription.value = kb.description || ''
  editDialogVisible.value = true
}

async function handleUpdateKB() {
  try {
    const { data } = await updateKB(editKBName.value, editKBDescription.value.trim())
    if (data.code === 0) {
      ElMessage.success('更新成功')
      editDialogVisible.value = false
      loadKBList()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('更新失败')
  }
}

async function handleDeleteKB(name) {
  try {
    await ElMessageBox.confirm(`确定删除知识库「${name}」及其所有文档？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deleteKB(name)
    if (data.code === 0) {
      ElMessage.success('已删除')
      if (currentKB.value === name) {
        currentKB.value = ''
      }
      loadKBList()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
}

function enterKB(name) {
  currentKB.value = name
  loadDocs()
}

async function handleUpload(file) {
  try {
    const { data } = await uploadDoc(currentKB.value, file)
    if (data.code === 0) {
      ElMessage.success('上传成功')
      loadDocs()
      loadKBList()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('上传失败')
  }
  return false
}

async function handleDeleteDoc(name) {
  try {
    await ElMessageBox.confirm(`确定删除文档「${name}」？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deleteDoc(currentKB.value, name)
    if (data.code === 0) {
      ElMessage.success('已删除')
      loadDocs()
      loadKBList()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1024 / 1024).toFixed(1) + ' MB'
}

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>

<style scoped>
.list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}

.list-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.list-scroll {
  flex: 1;
  overflow-y: auto;
}

.list-empty {
  text-align: center;
  color: #999;
  padding: 40px 16px;
  font-size: 14px;
}

.kb-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.15s;
}

.kb-item:hover {
  background: #f5f7fa;
}

.kb-item.active {
  background: #ecf5ff;
}

.kb-item-main {
  flex: 1;
  min-width: 0;
}

.kb-item-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kb-item-desc {
  font-size: 12px;
  color: #666;
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kb-item-meta {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

.kb-item-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s;
}

.kb-item:hover .kb-item-actions,
.kb-item.active .kb-item-actions {
  opacity: 1;
}

.detail-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.detail-empty-text {
  color: #999;
  font-size: 16px;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}

.detail-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.section-empty {
  text-align: center;
  color: #999;
  padding: 24px 0;
  background: #fff;
  border-radius: 4px;
}

.card-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.doc-card {
  background: #fff;
  border-radius: 4px;
  padding: 12px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  transition: box-shadow 0.2s;
}

.doc-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
}

.doc-main {
  flex: 1;
  min-width: 0;
}

.doc-name {
  font-size: 15px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.doc-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

.preview-body {
  min-height: 72vh;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.preview-image-wrap {
  flex: 1;
  overflow: auto;
  text-align: center;
}

.preview-image {
  max-width: 100%;
}

.preview-frame {
  width: 100%;
  height: 82vh;
  border: none;
}

.preview-text-wrap {
  flex: 1;
  overflow-y: auto;
  position: relative;
}

.preview-pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #303133;
}

.preview-md {
  font-size: 14px;
  line-height: 1.7;
  color: #303133;
}

.preview-truncate-tip {
  padding: 8px 12px;
  background: #fdf6ec;
  color: #e6a23c;
  font-size: 12px;
  border-radius: 4px;
  margin-bottom: 8px;
}

.preview-unsupported {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #999;
  gap: 8px;
}

.preview-unsupported-name {
  font-size: 13px;
}
</style>
