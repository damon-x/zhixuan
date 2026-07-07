<template>
  <MasterDetail :has-selection="hasSelection" @back="goBack">
    <template #list>
      <div class="list-header">
        <span class="list-title">笔记</span>
        <el-button size="small" @click="router.push('/notes/new')">新建</el-button>
      </div>
      <div class="list-scroll">
        <div v-if="notes.length === 0" class="list-empty">还没有笔记</div>
        <div
          v-for="note in notes"
          :key="note.id"
          class="note-item"
          :class="{ active: String(route.params.id) === String(note.id) }"
          @click="router.push(`/notes/${note.id}`)"
        >
          <div class="note-item-title">{{ note.title || '无标题' }}</div>
          <div class="note-item-time">{{ formatTime(note.updated_at) }}</div>
        </div>
      </div>
    </template>

    <template #detail>
      <!-- Empty placeholder -->
      <div v-if="!hasSelection" class="detail-empty">
        <div class="detail-empty-text">选择或新建一篇笔记</div>
      </div>

      <!-- Editor -->
      <template v-else>
        <div class="detail-header">
          <span class="detail-title">{{ isEdit ? '编辑笔记' : '新建笔记' }}</span>
          <div class="detail-actions">
            <template v-if="isEdit">
              <el-button size="small" @click="handleSummarize" :loading="aiLoading">总结</el-button>
              <el-button size="small" @click="handleGenerateTodos" :loading="aiLoading">生成待办</el-button>
              <el-button size="small" type="danger" @click="handleDelete">删除</el-button>
            </template>
            <el-button type="primary" size="small" @click="save" :loading="saving">保存</el-button>
          </div>
        </div>
        <div class="detail-body">
          <el-input v-model="title" placeholder="请输入标题" class="title-input" size="large" />
          <el-input
            v-model="content"
            type="textarea"
            placeholder="请输入内容"
            :autosize="{ minRows: 12 }"
            class="content-input"
          />
        </div>
      </template>
    </template>
  </MasterDetail>

  <!-- Summary dialog -->
  <el-dialog v-model="summaryDialogVisible" title="笔记总结" width="500px">
    <div class="summary-content">{{ summaryText }}</div>
    <template #footer>
      <el-button @click="summaryDialogVisible = false">关闭</el-button>
      <el-button type="primary" @click="appendToNote">添加到笔记末尾</el-button>
    </template>
  </el-dialog>

  <!-- Generate todos dialog -->
  <el-dialog v-model="todosDialogVisible" title="生成待办" width="500px">
    <div v-if="generatedTodos.length > 0" class="generated-todos">
      <el-checkbox v-model="todoCheckAll" @change="handleTodoCheckAllChange" style="margin-bottom:12px;">全选</el-checkbox>
      <el-checkbox-group v-model="checkedTodoIndices">
        <div v-for="(todo, idx) in generatedTodos" :key="idx" class="generated-todo-item">
          <el-checkbox :value="idx">
            <span class="todo-item-title">{{ todo.title }}</span>
            <span v-if="todo.content" class="todo-item-content"> - {{ todo.content }}</span>
            <el-tag :type="priorityTag(todo.priority)" size="small" style="margin-left:8px;">{{ priorityLabel(todo.priority) }}</el-tag>
          </el-checkbox>
        </div>
      </el-checkbox-group>
    </div>
    <template #footer>
      <el-button @click="todosDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="addCheckedTodos" :disabled="checkedTodoIndices.length === 0">添加 {{ checkedTodoIndices.length }} 项待办</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, onActivated } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getNotes, getNote, createNote, updateNote, deleteNote } from '../api/note'
import { executeAiTask } from '../api/ai'
import { createTodo } from '../api/todo'
import MasterDetail from '../components/MasterDetail.vue'

const route = useRoute()
const router = useRouter()
const notes = ref([])

const title = ref('')
const content = ref('')
const saving = ref(false)
const aiLoading = ref(false)
const notePlanId = ref(null)

const hasSelection = computed(() => route.params.id !== undefined)
const isEdit = computed(() => {
  const id = route.params.id
  return id !== undefined && id !== 'new'
})

// Summary dialog
const summaryDialogVisible = ref(false)
const summaryText = ref('')

// Generate todos dialog
const todosDialogVisible = ref(false)
const generatedTodos = ref([])
const checkedTodoIndices = ref([])
const todoCheckAll = ref(true)

onActivated(() => {
  loadNotes()
})

watch(() => route.params.id, async (newId) => {
  if (!route.path.startsWith('/notes')) return
  await loadDetail(newId)
}, { immediate: true })

async function loadNotes() {
  try {
    const { data } = await getNotes()
    if (data.code === 0) {
      notes.value = data.data || []
    }
  } catch {
    ElMessage.error('加载笔记失败')
  }
}

async function loadDetail(id) {
  title.value = ''
  content.value = ''
  notePlanId.value = null
  if (id === undefined) return
  if (id === 'new') {
    // New note: capture plan_id from query (used when creating from a plan)
    const queryPlanId = route.query.plan_id
    notePlanId.value = queryPlanId ? Number(queryPlanId) : null
    return
  }
  try {
    const { data } = await getNote(id)
    if (data.code === 0) {
      title.value = data.data.title
      content.value = data.data.content
      notePlanId.value = data.data.plan_id || null
    } else {
      ElMessage.error(data.msg)
      router.push('/notes')
    }
  } catch {
    ElMessage.error('加载笔记失败')
    router.push('/notes')
  }
}

async function save() {
  if (!title.value.trim()) {
    ElMessage.warning('请输入标题')
    return
  }
  saving.value = true
  try {
    let data
    const id = route.params.id
    if (isEdit.value) {
      const res = await updateNote(id, title.value, content.value)
      data = res.data
    } else {
      const res = await createNote(title.value, content.value, notePlanId.value || undefined)
      data = res.data
    }
    if (data.code === 0) {
      ElMessage.success('保存成功')
      await loadNotes()
      if (isEdit.value) {
        // already editing existing note, stay
      } else if (notePlanId.value) {
        // new note created from a plan context, return to the plan
        router.push(`/plans/${notePlanId.value}`)
      } else {
        // new note created, navigate to its edit route
        router.push(`/notes/${data.data.id}`)
      }
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function handleSummarize() {
  if (!content.value.trim()) {
    ElMessage.warning('笔记内容为空')
    return
  }
  aiLoading.value = true
  try {
    const { data } = await executeAiTask(content.value + '\n\n帮我总结笔记内容', 'summarize_note')
    if (data.code === 0) {
      summaryText.value = data.data.summary
      summaryDialogVisible.value = true
    } else {
      ElMessage.error(data.msg || '总结失败')
    }
  } catch {
    ElMessage.error('AI 请求失败')
  } finally {
    aiLoading.value = false
  }
}

function appendToNote() {
  content.value = content.value + '\n\n---\n总结：\n' + summaryText.value
  summaryDialogVisible.value = false
  ElMessage.success('已添加到笔记末尾')
}

async function handleGenerateTodos() {
  if (!content.value.trim()) {
    ElMessage.warning('笔记内容为空')
    return
  }
  aiLoading.value = true
  try {
    const { data } = await executeAiTask(content.value + '\n\n帮我根据笔记内容生成待办事项', 'generate_todos')
    if (data.code === 0) {
      generatedTodos.value = data.data.todos || []
      checkedTodoIndices.value = generatedTodos.value.map((_, i) => i)
      todoCheckAll.value = true
      todosDialogVisible.value = true
    } else {
      ElMessage.error(data.msg || '生成待办失败')
    }
  } catch {
    ElMessage.error('AI 请求失败')
  } finally {
    aiLoading.value = false
  }
}

function handleTodoCheckAllChange(val) {
  if (val) {
    checkedTodoIndices.value = generatedTodos.value.map((_, i) => i)
  } else {
    checkedTodoIndices.value = []
  }
}

function priorityLabel(p) {
  return ['低', '中', '高'][p] || '低'
}

function priorityTag(p) {
  return ['info', 'warning', 'danger'][p] || 'info'
}

async function addCheckedTodos() {
  const todos = checkedTodoIndices.value.map(i => generatedTodos.value[i])
  let successCount = 0
  for (const todo of todos) {
    try {
      const { data } = await createTodo({
        title: todo.title,
        content: todo.content || '',
        priority: todo.priority || 0,
        deadline: '',
      })
      if (data.code === 0) successCount++
    } catch { /* continue */ }
  }
  ElMessage.success(`已添加 ${successCount} 项待办`)
  todosDialogVisible.value = false
}

function goBack() {
  router.push('/notes')
}

async function handleDelete() {
  const id = route.params.id
  try {
    await ElMessageBox.confirm('确定删除该笔记？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deleteNote(id)
    if (data.code === 0) {
      ElMessage.success('已删除')
      await loadNotes()
      router.push('/notes')
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
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

.note-item {
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.15s;
}

.note-item:hover {
  background: #f5f7fa;
}

.note-item.active {
  background: #ecf5ff;
}

.note-item.active::before {
  display: none;
}

.note-item-title {
  font-size: 14px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.note-item-time {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
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
}

.detail-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.detail-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.title-input {
  margin-bottom: 16px;
}

.content-input :deep(.el-textarea__inner) {
  font-size: 15px;
  line-height: 1.6;
}

.summary-content {
  white-space: pre-wrap;
  line-height: 1.8;
  font-size: 14px;
  max-height: 400px;
  overflow-y: auto;
}

.generated-todos {
  max-height: 400px;
  overflow-y: auto;
}

.generated-todo-item {
  padding: 6px 0;
  border-bottom: 1px solid #f0f2f5;
}

.generated-todo-item:last-child {
  border-bottom: none;
}

.todo-item-title {
  font-weight: 500;
}

.todo-item-content {
  color: #999;
  font-size: 13px;
}
</style>
