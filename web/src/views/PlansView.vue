<template>
  <MasterDetail :has-selection="hasSelection" @back="goBack">
    <template #list>
      <div class="list-header">
        <span class="list-title">计划</span>
        <el-button size="small" @click="router.push('/plans/new')">新建</el-button>
      </div>
      <div class="list-scroll">
        <div v-if="plans.length === 0" class="list-empty">暂无计划</div>
        <div
          v-for="plan in plans"
          :key="plan.id"
          class="plan-item"
          :class="{ active: String(route.params.id) === String(plan.id) }"
          @click="router.push(`/plans/${plan.id}`)"
        >
          <div class="plan-item-main">
            <div class="plan-item-title">{{ plan.title }}</div>
            <div class="plan-item-time">{{ formatTime(plan.updated_at) }}</div>
          </div>
          <el-tag :type="statusTagType(plan.status)" size="small">{{ statusLabel(plan.status) }}</el-tag>
        </div>
      </div>
    </template>

    <template #detail>
      <!-- Empty placeholder -->
      <div v-if="!hasSelection" class="detail-empty">
        <div class="detail-empty-text">选择或新建一个计划</div>
      </div>

      <!-- Edit form -->
      <template v-else-if="mode === 'edit'">
        <div class="detail-header">
          <span class="detail-title">{{ isEdit ? '编辑计划' : '新建计划' }}</span>
          <div class="detail-actions">
            <el-button v-if="isEdit" size="small" @click="cancelEdit">取消</el-button>
            <el-button type="primary" size="small" @click="save" :loading="saving">保存</el-button>
          </div>
        </div>
        <div class="detail-body">
          <div class="edit-status">
            <span class="edit-status-label">状态</span>
            <el-select v-model="form.status" size="small" style="width:140px;">
              <el-option label="进行中" value="in_progress" />
              <el-option label="已完成" value="completed" />
            </el-select>
          </div>
          <el-input v-model="form.title" placeholder="请输入标题" class="title-input" size="large" />
          <el-input
            v-model="form.content"
            type="textarea"
            placeholder="请输入计划详情"
            :autosize="{ minRows: 12 }"
            class="content-input"
          />
        </div>
      </template>

      <!-- Detail view -->
      <template v-else>
        <div class="detail-header">
          <span class="detail-title">{{ plan.title || '计划详情' }}</span>
          <div class="detail-actions">
            <el-tag :type="statusTagType(plan.status)" size="small">{{ statusLabel(plan.status) }}</el-tag>
            <el-button size="small" @click="startEdit">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete">删除</el-button>
          </div>
        </div>
        <div class="detail-body">
          <div v-if="plan.content" class="plan-content">{{ plan.content }}</div>

          <!-- 笔记区 -->
          <div class="section">
            <div class="section-header">
              <span class="section-title">笔记</span>
              <el-button size="small" type="primary" @click="router.push(`/notes/new?plan_id=${planId}`)">新建笔记</el-button>
            </div>
            <div v-if="notes.length === 0" class="section-empty">暂无笔记</div>
            <div class="card-list">
              <div v-for="note in notes" :key="note.id" class="item-card">
                <div class="item-main" @click="router.push(`/notes/${note.id}`)">
                  <div class="item-title">{{ note.title }}</div>
                  <div class="item-time">{{ formatTime(note.updated_at) }}</div>
                </div>
                <el-button size="small" type="danger" @click="handleDeleteNote(note.id)">删除</el-button>
              </div>
            </div>
          </div>

          <!-- 待办区 -->
          <div class="section">
            <div class="section-header">
              <span class="section-title">待办</span>
              <el-button size="small" type="primary" @click="openTodoDialog()">新增待办</el-button>
            </div>
            <div v-if="todos.length === 0" class="section-empty">暂无待办</div>
            <div class="card-list">
              <div v-for="todo in todos" :key="todo.id" class="item-card" :class="{ 'todo-done': todo.done }">
                <div class="todo-row">
                  <div class="todo-left">
                    <el-checkbox :model-value="todo.done" @change="toggleTodoDone(todo)" />
                    <div class="todo-info">
                      <div class="item-title">{{ todo.title }}</div>
                      <div v-if="todo.content" class="todo-content">{{ todo.content }}</div>
                      <div class="item-time">
                        <el-tag :type="priorityTag(todo.priority)" size="small">{{ priorityLabel(todo.priority) }}</el-tag>
                        <span v-if="todo.deadline" class="todo-deadline">截止: {{ formatDate(todo.deadline) }}</span>
                      </div>
                    </div>
                  </div>
                  <div class="item-actions">
                    <el-button size="small" @click="openTodoDialog(todo)">编辑</el-button>
                    <el-button size="small" type="danger" @click="handleDeleteTodo(todo.id)">删除</el-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </template>
  </MasterDetail>

  <!-- 待办弹窗 -->
  <el-dialog v-model="todoDialogVisible" :title="todoIsEdit ? '编辑待办' : '新增待办'" width="420px">
    <el-form label-width="80px">
      <el-form-item label="标题">
        <el-input v-model="todoForm.title" placeholder="请输入待办标题" />
      </el-form-item>
      <el-form-item label="内容">
        <el-input v-model="todoForm.content" type="textarea" :rows="3" placeholder="请输入待办内容" />
      </el-form-item>
      <el-form-item label="优先级">
        <el-select v-model="todoForm.priority" style="width: 100%">
          <el-option label="低" :value="0" />
          <el-option label="中" :value="1" />
          <el-option label="高" :value="2" />
        </el-select>
      </el-form-item>
      <el-form-item label="截止时间">
        <el-date-picker v-model="todoForm.deadline" type="date" placeholder="选择日期" value-format="YYYY-MM-DD" style="width: 100%" :disabled-date="disablePastDate" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="todoDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="handleTodoSubmit">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, onActivated } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getPlans, getPlan, createPlan, updatePlan, deletePlan } from '../api/plan'
import { getNotes, deleteNote } from '../api/note'
import { getTodos, createTodo, updateTodo, deleteTodo } from '../api/todo'
import MasterDetail from '../components/MasterDetail.vue'

const route = useRoute()
const router = useRouter()
const plans = ref([])
const plan = ref({})
const notes = ref([])
const todos = ref([])

const mode = ref('view') // 'view' | 'edit'
const form = ref({ title: '', content: '', status: 'in_progress' })
const saving = ref(false)

const todoDialogVisible = ref(false)
const todoIsEdit = ref(false)
const todoEditId = ref(null)
const todoForm = ref({ title: '', content: '', priority: 0, deadline: '' })

const hasSelection = computed(() => route.params.id !== undefined)
const isEdit = computed(() => {
  const id = route.params.id
  return id !== undefined && id !== 'new'
})
const planId = computed(() => route.params.id)

onActivated(() => {
  loadPlans()
})

watch(() => route.params.id, async (newId) => {
  if (!route.path.startsWith('/plans')) return
  await handleRouteChange(newId)
}, { immediate: true })

async function handleRouteChange(id) {
  if (id === undefined) {
    mode.value = 'view'
    plan.value = {}
    return
  }
  if (id === 'new') {
    mode.value = 'edit'
    form.value = { title: '', content: '', status: 'in_progress' }
    notes.value = []
    todos.value = []
    return
  }
  mode.value = 'view'
  await loadPlan(id)
  await Promise.all([loadNotes(id), loadTodos(id)])
}

async function loadPlans() {
  try {
    const { data } = await getPlans()
    if (data.code === 0) {
      plans.value = data.data || []
    }
  } catch {
    ElMessage.error('加载计划失败')
  }
}

async function loadPlan(id) {
  try {
    const { data } = await getPlan(id)
    if (data.code === 0) {
      plan.value = data.data
    } else {
      ElMessage.error(data.msg)
      router.push('/plans')
    }
  } catch {
    ElMessage.error('加载计划失败')
    router.push('/plans')
  }
}

async function loadNotes(id) {
  try {
    const { data } = await getNotes(null, id)
    if (data.code === 0) {
      notes.value = data.data || []
    }
  } catch {
    ElMessage.error('加载笔记失败')
  }
}

async function loadTodos(id) {
  try {
    const { data } = await getTodos(id)
    if (data.code === 0) {
      todos.value = data.data || []
    }
  } catch {
    ElMessage.error('加载待办失败')
  }
}

function startEdit() {
  form.value = {
    title: plan.value.title || '',
    content: plan.value.content || '',
    status: plan.value.status || 'in_progress',
  }
  mode.value = 'edit'
}

function cancelEdit() {
  mode.value = 'view'
}

async function save() {
  if (!form.value.title.trim()) {
    ElMessage.warning('请输入标题')
    return
  }
  saving.value = true
  try {
    let data
    if (isEdit.value) {
      const res = await updatePlan(planId.value, form.value.title, form.value.content, form.value.status)
      data = res.data
      if (data.code === 0) {
        ElMessage.success('保存成功')
        await loadPlans()
        await loadPlan(planId.value)
        mode.value = 'view'
      } else {
        ElMessage.error(data.msg)
      }
    } else {
      const res = await createPlan(form.value.title, form.value.content, form.value.status)
      data = res.data
      if (data.code === 0) {
        ElMessage.success('保存成功')
        await loadPlans()
        router.push(`/plans/${data.data.id}`)
      } else {
        ElMessage.error(data.msg)
      }
    }
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete() {
  try {
    await ElMessageBox.confirm('确定删除该计划？关联的笔记和待办不会被删除，仅解除关联。', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deletePlan(planId.value)
    if (data.code === 0) {
      ElMessage.success('已删除')
      await loadPlans()
      router.push('/plans')
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
}

async function handleDeleteNote(id) {
  try {
    await ElMessageBox.confirm('确定删除该笔记？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deleteNote(id)
    if (data.code === 0) {
      ElMessage.success('已删除')
      loadNotes(planId.value)
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
}

function openTodoDialog(todo) {
  if (todo) {
    todoIsEdit.value = true
    todoEditId.value = todo.id
    todoForm.value = {
      title: todo.title,
      content: todo.content || '',
      priority: todo.priority,
      deadline: todo.deadline ? formatDate(todo.deadline) : '',
    }
  } else {
    todoIsEdit.value = false
    todoEditId.value = null
    todoForm.value = { title: '', content: '', priority: 0, deadline: '' }
  }
  todoDialogVisible.value = true
}

async function handleTodoSubmit() {
  if (!todoForm.value.title.trim()) {
    ElMessage.warning('请输入标题')
    return
  }
  try {
    const payload = {
      title: todoForm.value.title,
      content: todoForm.value.content,
      priority: todoForm.value.priority,
      deadline: todoForm.value.deadline || '',
      plan_id: Number(planId.value),
    }
    if (todoIsEdit.value) {
      const current = todos.value.find(t => t.id === todoEditId.value)
      payload.done = current ? current.done : false
      const { data } = await updateTodo(todoEditId.value, payload)
      if (data.code !== 0) {
        ElMessage.error(data.msg)
        return
      }
      ElMessage.success('更新成功')
    } else {
      const { data } = await createTodo(payload)
      if (data.code !== 0) {
        ElMessage.error(data.msg)
        return
      }
      ElMessage.success('创建成功')
    }
    todoDialogVisible.value = false
    loadTodos(planId.value)
  } catch {
    ElMessage.error('操作失败')
  }
}

async function toggleTodoDone(todo) {
  try {
    const { data } = await updateTodo(todo.id, {
      title: todo.title,
      content: todo.content || '',
      priority: todo.priority,
      deadline: todo.deadline ? formatDate(todo.deadline) : '',
      done: !todo.done,
    })
    if (data.code === 0) {
      loadTodos(planId.value)
    }
  } catch {
    ElMessage.error('操作失败')
  }
}

async function handleDeleteTodo(id) {
  try {
    await ElMessageBox.confirm('确定删除该待办？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deleteTodo(id)
    if (data.code === 0) {
      ElMessage.success('已删除')
      loadTodos(planId.value)
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
}

function statusLabel(s) {
  return s === 'completed' ? '已完成' : '进行中'
}

function statusTagType(s) {
  return s === 'completed' ? 'success' : 'warning'
}

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function formatDate(t) {
  if (!t) return ''
  const d = new Date(t)
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function priorityLabel(p) {
  return ['低', '中', '高'][p] || '低'
}

function priorityTag(p) {
  return ['info', 'warning', 'danger'][p] || 'info'
}

function disablePastDate(date) {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return date.getTime() < today.getTime()
}

function goBack() {
  router.push('/plans')
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

.plan-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.15s;
}

.plan-item:hover {
  background: #f5f7fa;
}

.plan-item.active {
  background: #ecf5ff;
}

.plan-item-main {
  flex: 1;
  min-width: 0;
}

.plan-item-title {
  font-size: 14px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.plan-item-time {
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
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-shrink: 0;
}

.detail-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.edit-status {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.edit-status-label {
  font-size: 14px;
  color: #606266;
}

.title-input {
  margin-bottom: 16px;
}

.content-input :deep(.el-textarea__inner) {
  font-size: 15px;
  line-height: 1.6;
}

.plan-content {
  background: #fff;
  border-radius: 4px;
  padding: 16px;
  margin-bottom: 20px;
  white-space: pre-wrap;
  line-height: 1.6;
  font-size: 15px;
  color: #333;
}

.section {
  margin-bottom: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.section-title {
  font-size: 16px;
  font-weight: bold;
  color: #333;
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

.item-card {
  background: #fff;
  border-radius: 4px;
  padding: 12px 16px;
  transition: box-shadow 0.2s;
}

.item-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
}

.item-main {
  cursor: pointer;
}

.item-title {
  font-size: 15px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-time {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.item-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.todo-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.todo-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.todo-info {
  flex: 1;
  min-width: 0;
}

.todo-content {
  font-size: 13px;
  color: #666;
  margin-top: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.todo-deadline {
  font-size: 12px;
  color: #999;
}

.todo-done {
  opacity: 0.6;
}

.todo-done .item-title {
  text-decoration: line-through;
}
</style>
