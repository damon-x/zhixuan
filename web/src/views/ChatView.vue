<template>
  <div class="chat-page">
    <div class="chat-layout">
      <!-- Sidebar -->
      <div class="sidebar" :class="{ 'sidebar-open': sidebarVisible }">
        <div class="sidebar-header">
          <div class="sidebar-header-actions">
            <el-button size="small" @click="handleNewSession">新建对话</el-button>
            <el-button size="small" @click="showSkillManager = true">Skill 管理</el-button>
            <el-button size="small" @click="showMemoryManager = true">记忆管理</el-button>
          </div>
        </div>
        <div class="session-list">
          <div
            v-for="s in sessions"
            :key="s.session_id"
            class="session-item"
            :class="{ active: currentSessionId === s.session_id }"
            @click="selectSession(s.session_id)"
          >
            <div class="session-info">
              <div class="session-summary">{{ s.is_main ? '知玄' : (s.title || '新对话') }}</div>
              <div class="session-time">{{ s.is_main ? '默认会话' : formatTime(s.updated_at) }}</div>
            </div>
            <el-button
              v-if="!s.is_main"
              class="session-delete"
              size="small"
              type="danger"
              :icon="Delete"
              circle
              @click.stop="handleDeleteSession(s.session_id)"
            />
          </div>
          <div v-if="sessions.length === 0" class="empty-tip">暂无对话</div>
        </div>
      </div>

      <!-- Overlay for mobile -->
      <div v-if="sidebarVisible" class="sidebar-overlay" @click="sidebarVisible = false"></div>

      <!-- Chat area -->
      <div class="chat-main">
        <div class="chat-mobile-bar">
          <el-button size="small" @click="sidebarVisible = !sidebarVisible">会话列表</el-button>
        </div>
        <div v-if="!currentSessionId" class="chat-empty">
          <div class="chat-empty-text">选择或新建一个对话开始聊天</div>
        </div>
        <template v-else>
          <div class="message-list" ref="messageListRef" @scroll="onMessageListScroll">
            <div v-if="loadingMore" class="load-more-tip">加载中...</div>
            <div v-else-if="!hasMore && messages.length > 0" class="load-more-tip">没有更早的消息了</div>
            <template v-for="(msg, idx) in messages" :key="msg.id">
              <div v-if="isTopicDivider(idx)" class="topic-divider">
                <span class="topic-divider-text">以上为历史话题</span>
              </div>
              <div
                class="message-row"
                :class="{ 'message-right': msg.role === 'user', 'message-left': msg.role === 'assistant' }"
              >
                <div class="message-content">
                  <div class="message-bubble" :class="{ 'bubble-user': msg.role === 'user', 'bubble-ai': msg.role === 'assistant' }">
                    <div v-if="msg.role === 'assistant'" class="message-text markdown-body" v-html="renderMarkdown(msg.content)"></div>
                    <div v-else class="message-text" v-html="renderUserMessage(msg.content)"></div>
                  </div>
                  <div v-if="msg.created_at" class="message-time">{{ formatMsgTime(msg.created_at) }}</div>
                </div>
              </div>
            </template>
            <div v-if="messages.length === 0" class="empty-tip">发送第一条消息开始对话</div>
          </div>

          <div class="input-area">
            <div class="input-toolbar">
              <el-button v-if="isCurrentMain" size="small" @click="handleStartTopic">开始新话题</el-button>
              <el-switch
                v-model="webSearchEnabled"
                active-text="联网搜索"
                inactive-text=""
                size="small"
                style="--el-switch-on-color: #409eff;"
              />
              <el-popover placement="top" :width="280" trigger="click">
                <template #reference>
                  <el-button size="small">知识库{{ selectedKBs.length ? ` (${selectedKBs.length})` : '' }}</el-button>
                </template>
                <div class="kb-select-list">
                  <el-checkbox-group v-model="selectedKBs">
                    <div v-for="kb in kbList" :key="kb.name" class="kb-select-item">
                      <el-checkbox :label="kb.name" :value="kb.name">
                        <span class="kb-select-name">{{ kb.name }}</span>
                        <span v-if="kb.description" class="kb-select-desc">{{ kb.description }}</span>
                      </el-checkbox>
                    </div>
                  </el-checkbox-group>
                  <div v-if="kbList.length === 0" class="kb-select-empty">暂无知识库</div>
                </div>
              </el-popover>
              <el-popover placement="top" :width="120" trigger="click">
                <template #reference>
                  <el-button size="small" :icon="Plus" />
                </template>
                <div class="attach-menu">
                  <div class="attach-menu-item" @click="triggerImageUpload">图片</div>
                </div>
              </el-popover>
              <input ref="fileInputRef" type="file" accept="image/*" style="display:none" @change="onImageSelected" />
            </div>
            <div v-if="pendingImage" class="image-preview-card">
              <img :src="`/api/resource?path=${encodeURIComponent(pendingImage.imagePath)}`" class="image-preview-thumb" />
              <span class="image-preview-close" @click="removePendingImage">&times;</span>
            </div>
            <div class="input-toolbar-row">
              <div class="input-wrapper">
                <div
                  ref="inputEl"
                  class="rich-input"
                  contenteditable="true"
                  :data-placeholder="sending ? '等待回复...' : '输入消息，回车发送，Shift+回车换行'"
                  @input="onInputChange"
                  @keydown="onInputKeydown"
                  @paste="onPaste"
                  @compositionstart="isComposing = true"
                  @compositionend="isComposing = false"
                ></div>
                <!-- @ Note search popup -->
                <div v-if="noteSearchVisible" class="note-search-popup">
                  <div
                    v-for="note in noteSearchResults"
                    :key="note.id"
                    class="note-search-item"
                    @mousedown.prevent="insertNoteRef(note)"
                  >
                    <span class="note-search-title">{{ note.title }}</span>
                    <span class="note-search-time">{{ formatTime(note.updated_at) }}</span>
                  </div>
                  <div v-if="noteSearchResults.length === 0" class="note-search-empty">无匹配笔记</div>
                </div>
              </div>
              <el-button v-if="!sending" type="primary" :disabled="!inputHasContent" @click="handleSend" circle class="send-btn">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
                  <path d="M4.93934 10.2598C4.35355 9.67396 4.35355 8.72444 4.93934 8.13865L10.2304 2.84763C11.203 1.87527 12.7875 1.86579 13.7675 2.84568L19.0605 8.13865C19.646 8.72439 19.646 9.67401 19.0605 10.2597C18.4748 10.8455 17.5252 10.8454 16.9394 10.2597L13.4999 6.82029L13.4999 20.6992C13.4999 21.5275 12.8282 22.199 11.9999 22.1992C11.1715 22.1992 10.4999 21.5276 10.4999 20.6992L10.4999 6.82029L7.06044 10.2598C6.47471 10.8455 5.52514 10.8454 4.93934 10.2598Z" fill="currentColor"></path>
                </svg>
              </el-button>
              <el-button v-else @click="handleStop" circle class="send-btn">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
                  <g clip-path="url(#clip0_299_3088)">
                    <path d="M12 0.5C18.3513 0.5 23.5 5.64873 23.5 12C23.5 18.3513 18.3513 23.5 12 23.5C5.64873 23.5 0.5 18.3513 0.5 12C0.5 5.64873 5.64873 0.5 12 0.5ZM12 2.5C6.75329 2.5 2.5 6.75329 2.5 12C2.5 17.2467 6.75329 21.5 12 21.5C17.2467 21.5 21.5 17.2467 21.5 12C21.5 6.75329 17.2467 2.5 12 2.5ZM12.5 7.5C14.3856 7.5 15.3283 7.50015 15.9141 8.08594C16.4998 8.67172 16.5 9.61438 16.5 11.5V12.5C16.5 14.3856 16.4998 15.3283 15.9141 15.9141C15.3283 16.4998 14.3856 16.5 12.5 16.5H11.5C9.61438 16.5 8.67172 16.4998 8.08594 15.9141C7.50015 15.3283 7.5 14.3856 7.5 12.5V11.5C7.5 9.61438 7.50015 8.67172 8.08594 8.08594C8.67172 7.50015 9.61438 7.5 11.5 7.5H12.5Z" fill="currentColor"></path>
                  </g>
                  <defs>
                    <clipPath id="clip0_299_3088"><rect width="24" height="24" fill="currentColor"></rect></clipPath>
                  </defs>
                </svg>
              </el-button>
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- QQBot Bind Dialog -->
    <el-dialog v-model="showQQBotDialog" title="绑定QQBot" width="420px" :close-on-click-modal="false" @close="onQQBotDialogClose">
      <div v-if="qqBotBound" class="qqbot-status">
        <el-result icon="success" title="已绑定QQBot" sub-text="QQ机器人已成功绑定">
          <template #extra>
            <el-button type="primary" @click="showQQBotDialog = false">关闭</el-button>
          </template>
        </el-result>
      </div>
      <div v-else-if="qqBotBinding">
        <div class="qqbot-code-area">
          <p>请用QQ给机器人发送以下验证码：</p>
          <div class="qqbot-code">{{ qqBotCode }}</div>
          <p class="qqbot-tip">等待验证中...</p>
        </div>
      </div>
      <div v-else>
        <el-form label-width="100px">
          <el-form-item label="App ID">
            <el-input v-model="qqBotAppId" placeholder="QQ机器人 App ID" />
          </el-form-item>
          <el-form-item label="App Secret">
            <el-input v-model="qqBotAppSecret" type="password" placeholder="QQ机器人 App Secret" show-password />
          </el-form-item>
        </el-form>
        <div style="text-align: right">
          <el-button type="primary" :disabled="!qqBotAppId || !qqBotAppSecret" @click="handleStartBind">开始绑定</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- Image Lightbox -->
    <div v-if="lightboxSrc" class="lightbox" @click="lightboxSrc = ''">
      <img :src="lightboxSrc" class="lightbox-img" @click.stop />
    </div>

    <!-- Skill Manager -->
    <SkillManager v-model="showSkillManager" />

    <!-- Memory Manager -->
    <MemoryManager v-model="showMemoryManager" />
  </div>
</template>

<script setup>
import { ref, computed, onActivated, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Plus } from '@element-plus/icons-vue'
import { sendMessage, getSessions, getSessionMessages, createSession, deleteSession, startTopic, stopMessage, uploadChatImage } from '../api/chat'
import { getNotes } from '../api/note'
import { getKBList } from '../api/knowledge'
import { marked } from 'marked'
import SkillManager from '../components/SkillManager.vue'
import MemoryManager from '../components/MemoryManager.vue'

const route = useRoute()

const router = useRouter()
const sessions = ref([])
const currentSessionId = ref('')
const messages = ref([])
const sendingSessionId = ref(null)
const sidebarVisible = ref(false)
const showSkillManager = ref(false)
const showMemoryManager = ref(false)
const messageListRef = ref(null)
const inputEl = ref(null)

// @ Note search state
const noteSearchVisible = ref(false)
const noteSearchResults = ref([])
const atStartPosition = ref(null)
const webSearchEnabled = ref(true)
const selectedKBs = ref([])
const kbList = ref([])
const pendingImage = ref(null) // { name, imagePath }
const fileInputRef = ref(null)
const lightboxSrc = ref('')

const sending = computed(() => sendingSessionId.value === currentSessionId.value)
const inputHasText = ref(false)
const inputHasContent = computed(() => inputHasText.value || !!pendingImage.value)
// IME 组合态标记：中文输入法选词时按回车只确认候选词，不触发发送
const isComposing = ref(false)
// 历史消息分页加载
const hasMore = ref(false)
const loadingMore = ref(false)

const isCurrentMain = computed(() => {
  const s = sessions.value.find(s => s.session_id === currentSessionId.value)
  return s?.is_main ?? false
})

const currentTopicSince = computed(() => {
  const s = sessions.value.find(s => s.session_id === currentSessionId.value)
  return s?.topic_since ?? 0
})

function isTopicDivider(idx) {
  const topicSince = currentTopicSince.value
  if (!topicSince) return false
  const msg = messages.value[idx]
  if (!msg || msg.id <= topicSince) return false
  // 当前消息在 topic 边界之后，判断是否为边界后第一条
  if (idx === 0) return true
  return messages.value[idx - 1]?.id <= topicSince
}

function formatMsgTime(t) {
  if (!t) return ''
  const d = new Date(t)
  if (isNaN(d.getTime())) return ''
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

async function handleStartTopic() {
  if (!currentSessionId.value) return
  try {
    const { data } = await startTopic(currentSessionId.value)
    if (data.code === 0) {
      ElMessage.success('已开始新话题')
      await loadSessions()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('操作失败')
  }
}

let lightboxBound = false

onActivated(async () => {
  await loadSessions()
  if (!currentSessionId.value) {
    const routeSessionId = route.params.sessionId
    if (routeSessionId) {
      currentSessionId.value = routeSessionId
    } else {
      const mainSession = sessions.value.find(s => s.is_main)
      if (mainSession) {
        currentSessionId.value = mainSession.session_id
      }
    }
  }
  await loadMessages()
  loadKBList()

  // Image lightbox: delegate click on .chat-image (bind once)
  await nextTick()
  if (!lightboxBound && messageListRef.value) {
    messageListRef.value.addEventListener('click', (e) => {
      const img = e.target.closest('.chat-image')
      if (img) {
        lightboxSrc.value = img.src
      }
    })
    lightboxBound = true
  }
})

watch(() => route.params.sessionId, async (newId, oldId) => {
  if (!route.path.startsWith('/chat')) return
  if (newId === oldId) return
  if (newId && newId !== currentSessionId.value) {
    currentSessionId.value = newId
    await loadMessages()
  }
})

async function loadSessions() {
  try {
    const { data } = await getSessions()
    if (data.code === 0) {
      sessions.value = data.data || []
    }
  } catch {
    ElMessage.error('加载会话失败')
  }
}

async function loadKBList() {
  try {
    const { data } = await getKBList()
    if (data.code === 0) {
      kbList.value = data.data || []
    }
  } catch { /* ignore */ }
}

async function selectSession(sessionId) {
  currentSessionId.value = sessionId
  sidebarVisible.value = false
  await loadMessages()
  router.push({ name: 'ChatSession', params: { sessionId } })
}

async function loadMessages() {
  if (!currentSessionId.value) return
  try {
    const { data } = await getSessionMessages(currentSessionId.value)
    if (data.code === 0) {
      messages.value = data.data || []
      hasMore.value = !!data.has_more
      await nextTick()
      scrollToBottom()
    }
  } catch {
    ElMessage.error('加载消息失败')
  }
}

async function loadMoreMessages() {
  if (loadingMore.value || !hasMore.value || !currentSessionId.value) return
  if (messages.value.length === 0) return
  loadingMore.value = true
  const list = messageListRef.value
  const oldScrollHeight = list?.scrollHeight || 0
  const oldScrollTop = list?.scrollTop || 0
  const oldestId = messages.value[0].id
  try {
    const { data } = await getSessionMessages(currentSessionId.value, oldestId)
    if (data.code === 0) {
      const older = data.data || []
      hasMore.value = !!data.has_more
      if (older.length > 0) {
        messages.value = [...older, ...messages.value]
        await nextTick()
        // 保持视觉位置：把旧内容的顶边对齐到原来的 scrollTop
        if (list) {
          list.scrollTop = list.scrollHeight - oldScrollHeight + oldScrollTop
        }
      }
    }
  } catch { /* ignore */ }
  loadingMore.value = false
}

function onMessageListScroll() {
  const list = messageListRef.value
  if (!list) return
  if (list.scrollTop < 50) {
    loadMoreMessages()
  }
}

async function handleNewSession() {
  try {
    const { data } = await createSession()
    if (data.code === 0) {
      currentSessionId.value = data.data.session_id
      messages.value = []
      sidebarVisible.value = false
      await loadSessions()
      router.push({ name: 'ChatSession', params: { sessionId: data.data.session_id } })
    }
  } catch {
    ElMessage.error('创建会话失败')
  }
}

async function handleDeleteSession(sessionId) {
  try {
    await ElMessageBox.confirm('确定删除该对话及其所有消息？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deleteSession(sessionId)
    if (data.code === 0) {
      ElMessage.success('已删除')
      if (currentSessionId.value === sessionId) {
        const mainSession = sessions.value.find(s => s.is_main)
        if (mainSession) {
          currentSessionId.value = mainSession.session_id
        }
        await loadMessages()
      }
      loadSessions()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
}

// Extract plain text + note ref tags from contenteditable
function getInputContent() {
  const el = inputEl.value
  if (!el) return ''

  function walkNodes(parent) {
    let text = ''
    for (const node of parent.childNodes) {
      if (node.nodeType === Node.TEXT_NODE) {
        text += node.textContent
      } else if (node.nodeType === Node.ELEMENT_NODE) {
        if (node.classList.contains('note-ref')) {
          text += `[note:${node.dataset.noteName}:${node.dataset.noteId}]`
        } else if (node.tagName === 'BR') {
          text += '\n'
        } else {
          if (text.length > 0 && !text.endsWith('\n')) text += '\n'
          text += walkNodes(node)
        }
      }
    }
    return text
  }

  return walkNodes(el).trim()
}

function clearInput() {
  if (inputEl.value) {
    inputEl.value.innerHTML = ''
  }
}

async function handleSend() {
  const text = getInputContent()
  if ((!text && !pendingImage.value) || sending.value || !currentSessionId.value) return

  sendingSessionId.value = currentSessionId.value
  clearInput()
  inputHasText.value = false
  noteSearchVisible.value = false

  let content = text
  if (pendingImage.value) {
    const imgTag = `[image:${pendingImage.value.name}:${pendingImage.value.imagePath}]`
    content = content ? imgTag + content : imgTag
    pendingImage.value = null
  }

  const sendSessionId = currentSessionId.value
  const tempId = Date.now()
  messages.value.push({ id: tempId, role: 'user', content, created_at: new Date().toISOString() })
  await nextTick()
  scrollToBottom()

  try {
    const { data } = await sendMessage(sendSessionId, content, webSearchEnabled.value, selectedKBs.value)
    if (currentSessionId.value !== sendSessionId) return
    if (data.code === 0) {
      messages.value.push(data.data)
      await nextTick()
      scrollToBottom()
      loadSessions()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    if (currentSessionId.value !== sendSessionId) return
    ElMessage.error('发送失败')
  } finally {
    sendingSessionId.value = null
  }
}

async function handleStop() {
  if (!currentSessionId.value) return
  try {
    await stopMessage(currentSessionId.value)
  } catch { /* ignore */ }
}

function triggerImageUpload() {
  if (fileInputRef.value) fileInputRef.value.click()
}

async function onImageSelected(e) {
  const file = e.target.files?.[0]
  if (!file) return
  try {
    const { data } = await uploadChatImage(file)
    if (data.code === 0) {
      pendingImage.value = { name: file.name, imagePath: data.data.image_path }
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('上传失败')
  }
  // Reset input so same file can be re-selected
  e.target.value = ''
}

function removePendingImage() {
  pendingImage.value = null
}

// @ Note search
function onInputChange() {
  inputHasText.value = !!getInputContent()
  checkAtTrigger()
}

function onPaste(e) {
  e.preventDefault()
  const text = e.clipboardData.getData('text/plain')
  document.execCommand('insertText', false, text)
}

function checkAtTrigger() {
  const el = inputEl.value
  if (!el) return
  const sel = window.getSelection()
  if (!sel.rangeCount) { noteSearchVisible.value = false; return }
  const range = sel.getRangeAt(0)
  if (!range.collapsed) { noteSearchVisible.value = false; return }

  // Find the @ in current text node
  const node = range.startContainer
  if (node.nodeType !== Node.TEXT_NODE) { noteSearchVisible.value = false; return }
  const textBefore = node.textContent.substring(0, range.startOffset)
  const atIndex = textBefore.lastIndexOf('@')
  if (atIndex === -1) {
    noteSearchVisible.value = false
    return
  }

  // Check no space between @ and cursor
  const query = textBefore.substring(atIndex + 1)
  if (query.includes(' ') || query.includes('\n')) {
    noteSearchVisible.value = false
    return
  }

  atStartPosition.value = { node, offset: atIndex }
  searchNotes(query)
}

async function searchNotes(query) {
  try {
    const { data } = await getNotes(query || undefined)
    if (data.code === 0) {
      noteSearchResults.value = data.data || []
    }
  } catch {
    noteSearchResults.value = []
  }
  noteSearchVisible.value = true
}

function insertNoteRef(note) {
  const el = inputEl.value
  if (!el || !atStartPosition.value) return

  const { node, offset } = atStartPosition.value
  const sel = window.getSelection()
  const range = document.createRange()

  // Select from @ to cursor
  range.setStart(node, offset)
  range.setEnd(node, node.textContent.length > (sel.focusOffset ?? offset) ? sel.getRangeAt(0).startOffset : offset)

  // Get actual cursor position in text
  const selRange = sel.getRangeAt(0)
  range.setStart(node, offset)
  range.setEnd(node, selRange.startOffset)
  range.deleteContents()

  // Create capsule span
  const span = document.createElement('span')
  span.className = 'note-ref'
  span.contentEditable = 'false'
  span.dataset.noteId = note.id
  span.dataset.noteName = note.title
  span.textContent = `@${note.title}`

  range.insertNode(span)

  // Add space after
  const space = document.createTextNode('\u00A0')
  span.after(space)

  // Move cursor after space
  const newRange = document.createRange()
  newRange.setStartAfter(space)
  newRange.collapse(true)
  sel.removeAllRanges()
  sel.addRange(newRange)

  noteSearchVisible.value = false
  atStartPosition.value = null
}

function onInputKeydown(e) {
  // 回车发送：IME 组合态下（如中文输入法选词回车）仅确认候选词不发送；
  // Shift+回车走默认换行行为
  if (e.key === 'Enter') {
    if (isComposing.value || e.isComposing || e.keyCode === 229) return
    if (e.shiftKey) return
    e.preventDefault()
    handleSend()
    return
  }
  if (e.key === 'Escape') {
    noteSearchVisible.value = false
  }
  if (e.key === 'Backspace') {
    // Check if cursor is right after a note-ref, delete whole ref
    const sel = window.getSelection()
    if (!sel.rangeCount) return
    const range = sel.getRangeAt(0)
    if (!range.collapsed) return
    const node = range.startContainer
    // If cursor is in a text node right after a note-ref
    if (node.nodeType === Node.TEXT_NODE && range.startOffset === 0 && node.previousSibling && node.previousSibling.classList?.contains('note-ref')) {
      e.preventDefault()
      node.previousSibling.remove()
    }
    // If cursor is inside or right at boundary of note-ref
    if (node.nodeType === Node.ELEMENT_NODE && node.classList?.contains('note-ref')) {
      e.preventDefault()
      node.remove()
    }
  }
}

function scrollToBottom() {
  if (messageListRef.value) {
    messageListRef.value.scrollTop = messageListRef.value.scrollHeight
  }
}

function renderMarkdown(content) {
  let text = content || ''
  text = text.replace(/\[image:([^:\]]+):([^\]]+)\]/g, (match, name, path) => {
    return `<img src="/api/resource?path=${encodeURIComponent(path)}" alt="${name}" class="chat-image" />`
  })
  return marked.parse(text)
}

function renderUserMessage(content) {
  // Split by image tags, escape text parts, then reassemble
  const parts = content.split(/(\[image:[^:\]]+:[^\]]+\])/)
  let result = ''
  for (const part of parts) {
    const imgMatch = part.match(/^\[image:([^:\]]+):([^\]]+)\]$/)
    if (imgMatch) {
      result += `<img src="/api/resource?path=${encodeURIComponent(imgMatch[2])}" alt="${imgMatch[1]}" class="chat-image" />`
    } else {
      const escaped = part
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
      result += escaped.replace(/\[note:([^:]*):(\d+)\]/g, '<span class="inline-note-ref">@$1</span>')
    }
  }
  return result
}

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  const pad = n => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

</script>

<style scoped>
.chat-page {
  height: 100%;
  background: #fff;
  display: flex;
  flex-direction: column;
}

.sidebar-toggle {
  display: none;
}

.chat-mobile-bar {
  display: none;
}

.chat-layout {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* Sidebar */
.sidebar {
  width: 280px;
  background: #fff;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.sidebar-header-actions {
  display: flex;
  gap: 8px;
}

.sidebar-header-actions .el-button {
  flex: 1;
}

.session-list {
  flex: 1;
  overflow-y: auto;
}

.session-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.2s;
  gap: 8px;
}

.session-item:hover {
  background: #f0f2f5;
}

.session-item.active {
  background: #ecf5ff;
}

.session-info {
  flex: 1;
  min-width: 0;
}

.session-summary {
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-time {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

.session-delete {
  opacity: 0;
  transition: opacity 0.2s;
  flex-shrink: 0;
}

.session-item:hover .session-delete {
  opacity: 1;
}

.empty-tip {
  text-align: center;
  color: #999;
  margin-top: 60px;
  padding: 0 20px;
}

.load-more-tip {
  text-align: center;
  color: #909399;
  font-size: 12px;
  padding: 8px 0;
}

/* Chat main */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.chat-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.chat-empty-text {
  color: #999;
  font-size: 16px;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.message-row {
  display: flex;
}

.message-right {
  justify-content: flex-end;
}

.message-left {
  justify-content: flex-start;
}

.message-content {
  display: flex;
  flex-direction: column;
  max-width: 70%;
}

.message-right .message-content {
  align-items: flex-end;
}

.message-left .message-content {
  align-items: flex-start;
}

.message-bubble {
  padding: 10px 14px;
  border-radius: 12px;
  word-break: break-word;
  white-space: pre-wrap;
}

.bubble-user {
  background: #e8eef5;
  color: #303133;
  border-bottom-right-radius: 4px;
}

.bubble-ai {
  background: transparent;
  color: #303133;
  border-bottom-left-radius: 4px;
  white-space: normal;
}

.message-text {
  font-size: 14px;
  line-height: 1.6;
}

.send-btn {
  width: 36px;
  height: 36px;
  padding: 0;
  flex-shrink: 0;
  align-self: flex-end;
}

.message-time {
  font-size: 11px;
  color: #909399;
  background: transparent;
  margin-top: 4px;
  padding: 0 4px;
  line-height: 1.2;
}

/* Inline note ref capsule in user messages */
.message-text :deep(.inline-note-ref) {
  display: inline-block;
  background: rgba(255, 255, 255, 0.3);
  color: #fff;
  padding: 1px 8px;
  border-radius: 10px;
  font-size: 13px;
  vertical-align: baseline;
  margin: 0 2px;
}

.markdown-body :deep(p) {
  margin: 0 0 8px 0;
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 20px;
  margin: 4px 0;
}

.markdown-body :deep(li) {
  margin: 2px 0;
}

.markdown-body :deep(code) {
  background: rgba(0, 0, 0, 0.08);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
  font-family: Menlo, Monaco, Consolas, monospace;
}

.markdown-body :deep(pre) {
  background: rgba(0, 0, 0, 0.08);
  padding: 12px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 8px 0;
}

.markdown-body :deep(pre code) {
  background: none;
  padding: 0;
  font-size: 13px;
}

.markdown-body :deep(blockquote) {
  border-left: 3px solid #c0c4cc;
  padding-left: 12px;
  margin: 8px 0;
  color: #606266;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3) {
  margin: 12px 0 6px 0;
  font-weight: 600;
}

.markdown-body :deep(table) {
  border-collapse: collapse;
  margin: 8px 0;
  width: 100%;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid #dcdfe6;
  padding: 6px 10px;
  text-align: left;
}

.markdown-body :deep(th) {
  background: rgba(0, 0, 0, 0.04);
}

.message-text :deep(.chat-image) {
  max-width: 100%;
  max-height: 300px;
  border-radius: 8px;
  margin: 4px 0;
  object-fit: contain;
}

.topic-divider {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px 0;
}

.topic-divider-text {
  font-size: 12px;
  color: #909399;
  background: #f5f5f5;
  padding: 4px 16px;
  border-radius: 12px;
}

/* Input area */
.input-area {
  display: flex;
  flex-direction: column;
  padding: 12px 20px 16px 20px;
  border-top: 1px solid #e4e7ed;
  background: #fff;
  gap: 8px;
  position: relative;
}

.image-preview-card {
  position: relative;
  display: inline-block;
  width: 64px;
  height: 64px;
  border-radius: 8px;
  overflow: hidden;
}

.image-preview-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-preview-close {
  display: none;
  position: absolute;
  top: 0;
  right: 0;
  width: 20px;
  height: 20px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 14px;
  line-height: 20px;
  text-align: center;
  cursor: pointer;
  border-radius: 0 8px 0 4px;
}

.image-preview-card:hover .image-preview-close {
  display: block;
}

.attach-menu-item {
  padding: 8px 12px;
  cursor: pointer;
  font-size: 14px;
  border-radius: 4px;
  transition: background 0.15s;
}

.attach-menu-item:hover {
  background: #ecf5ff;
  color: #409eff;
}

.input-toolbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-bottom: 4px;
}

.input-toolbar-row {
  display: flex;
  align-items: flex-end;
  gap: 12px;
}

.input-wrapper {
  flex: 1;
  position: relative;
}

.rich-input {
  min-height: 60px;
  max-height: 160px;
  overflow-y: auto;
  padding: 8px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-size: 14px;
  line-height: 1.6;
  outline: none;
  white-space: pre-wrap;
  word-break: break-word;
}

.rich-input:empty::before {
  content: attr(data-placeholder);
  color: #c0c4cc;
}

.rich-input:focus {
  border-color: #409eff;
}

/* Note ref capsule in input */
.rich-input :deep(.note-ref) {
  display: inline-block;
  background: #ecf5ff;
  color: #409eff;
  padding: 0 8px;
  border-radius: 10px;
  font-size: 13px;
  line-height: 20px;
  margin: 0 2px;
  vertical-align: baseline;
  cursor: default;
  user-select: all;
}

/* Note search popup */
.note-search-popup {
  position: absolute;
  bottom: 100%;
  left: 0;
  right: 0;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  max-height: 220px;
  overflow-y: auto;
  z-index: 10;
  margin-bottom: 4px;
}

.note-search-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  cursor: pointer;
  transition: background 0.15s;
}

.note-search-item:hover {
  background: #ecf5ff;
}

.note-search-title {
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.note-search-time {
  font-size: 12px;
  color: #999;
  flex-shrink: 0;
  margin-left: 12px;
}

.note-search-empty {
  padding: 16px;
  text-align: center;
  color: #999;
  font-size: 13px;
}

/* KB select popover */
.kb-select-list {
  max-height: 300px;
  overflow-y: auto;
}

.kb-select-item {
  padding: 4px 0;
}

.kb-select-item :deep(.el-checkbox__label) {
  display: inline-flex;
  flex-direction: column;
}

.kb-select-name {
  font-size: 14px;
}

.kb-select-desc {
  font-size: 12px;
  color: #999;
  margin-top: 2px;
}

.kb-select-empty {
  padding: 16px;
  text-align: center;
  color: #999;
  font-size: 13px;
}

/* Image Lightbox */
.lightbox {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  cursor: pointer;
}

.lightbox-img {
  max-width: 90vw;
  max-height: 90vh;
  object-fit: contain;
  cursor: default;
}

/* Mobile */
@media (max-width: 768px) {
  .sidebar-toggle {
    display: inline-flex;
  }

  .chat-mobile-bar {
    display: flex;
    padding: 8px 12px;
    background: #fff;
    border-bottom: 1px solid #e4e7ed;
    flex-shrink: 0;
  }

  .sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 280px;
    z-index: 100;
    transform: translateX(-100%);
    transition: transform 0.3s;
  }

  .sidebar.sidebar-open {
    transform: translateX(0);
  }

  .sidebar-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.3);
    z-index: 99;
  }

  .message-content {
    max-width: 85%;
  }

  .input-area {
    padding: 12px;
  }
}
</style>
