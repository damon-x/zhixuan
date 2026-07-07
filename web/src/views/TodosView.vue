<template>
  <MasterDetail :has-selection="hasSelection" @back="goBack">
    <template #list>
      <div class="list-header">
        <span class="list-title">待办</span>
        <el-button size="small" @click="router.push('/todos/new')">新增</el-button>
      </div>
      <div class="list-scroll">
        <div v-if="todos.length === 0" class="list-empty">暂无待办</div>
        <div
          v-for="todo in todos"
          :key="todo.id"
          class="todo-item"
          :class="{ active: String(route.params.id) === String(todo.id), done: todo.done }"
          @click="router.push(`/todos/${todo.id}`)"
        >
          <el-checkbox :model-value="todo.done" @click.stop @change="toggleDone(todo)" />
          <div class="todo-item-main">
            <div class="todo-item-title">{{ todo.title }}</div>
            <div class="todo-item-meta">
              <el-tag :type="priorityTag(todo.priority)" size="small">{{ priorityLabel(todo.priority) }}</el-tag>
              <span v-if="todo.deadline" class="todo-item-deadline">{{ formatDate(todo.deadline) }}</span>
            </div>
          </div>
          <el-button
            class="todo-item-del"
            size="small"
            type="danger"
            circle
            @click.stop="handleDelete(todo.id)"
          >删</el-button>
        </div>
      </div>
    </template>

    <template #detail>
      <div v-if="!hasSelection" class="detail-empty">
        <div class="detail-empty-text">选择或新增一个待办</div>
      </div>
      <template v-else>
        <div class="detail-header">
          <span class="detail-title">{{ isEdit ? '编辑待办' : '新增待办' }}</span>
          <div class="detail-actions">
            <el-button v-if="isEdit" size="small" type="danger" @click="handleDelete(editId)">删除</el-button>
            <el-button size="small" @click="goBack">取消</el-button>
            <el-button type="primary" size="small" @click="handleSubmit" :loading="saving">保存</el-button>
          </div>
        </div>
        <div class="detail-body">
          <el-form label-width="80px">
            <el-form-item label="标题">
              <el-input v-model="form.title" placeholder="请输入待办标题" />
            </el-form-item>
            <el-form-item label="内容">
              <el-input v-model="form.content" type="textarea" :rows="3" placeholder="请输入待办内容" />
            </el-form-item>
            <el-form-item label="优先级">
              <el-select v-model="form.priority" style="width: 100%">
                <el-option label="低" :value="0" />
                <el-option label="中" :value="1" />
                <el-option label="高" :value="2" />
              </el-select>
            </el-form-item>
            <el-form-item label="截止时间">
              <el-date-picker v-model="form.deadline" type="date" placeholder="选择日期" value-format="YYYY-MM-DD" style="width: 100%" :disabled-date="disablePastDate" />
            </el-form-item>
          </el-form>
        </div>
      </template>
    </template>
  </MasterDetail>
</template>

<script setup>
import { ref, computed, watch, onActivated } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getTodos, createTodo, updateTodo, deleteTodo } from '../api/todo'
import MasterDetail from '../components/MasterDetail.vue'

const route = useRoute()
const router = useRouter()
const todos = ref([])
const saving = ref(false)
const editId = ref(null)
const form = ref({ title: '', content: '', priority: 0, deadline: '' })

const hasSelection = computed(() => route.params.id !== undefined)
const isEdit = computed(() => {
  const id = route.params.id
  return id !== undefined && id !== 'new'
})

onActivated(() => {
  loadTodos()
})

watch(() => route.params.id, (newId) => {
  if (!route.path.startsWith('/todos')) return
  loadForm(newId)
}, { immediate: true })

async function loadTodos() {
  try {
    const { data } = await getTodos()
    if (data.code === 0) {
      todos.value = data.data || []
    }
  } catch {
    ElMessage.error('加载待办失败')
  }
}

function loadForm(id) {
  if (id === undefined || id === 'new') {
    editId.value = null
    form.value = { title: '', content: '', priority: 0, deadline: '' }
    return
  }
  const todo = todos.value.find(t => String(t.id) === String(id))
  if (todo) {
    editId.value = todo.id
    form.value = {
      title: todo.title,
      content: todo.content || '',
      priority: todo.priority,
      deadline: todo.deadline ? formatDate(todo.deadline) : '',
    }
  } else {
    editId.value = Number(id)
    form.value = { title: '', content: '', priority: 0, deadline: '' }
  }
}

function disablePastDate(date) {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return date.getTime() < today.getTime()
}

function priorityLabel(p) {
  return ['低', '中', '高'][p] || '低'
}

function priorityTag(p) {
  return ['info', 'warning', 'danger'][p] || 'info'
}

function formatDate(t) {
  if (!t) return ''
  const d = new Date(t)
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

async function handleSubmit() {
  if (!form.value.title.trim()) {
    ElMessage.warning('请输入标题')
    return
  }
  saving.value = true
  try {
    const payload = {
      title: form.value.title,
      content: form.value.content,
      priority: form.value.priority,
      deadline: form.value.deadline || '',
    }
    if (isEdit.value) {
      const current = todos.value.find(t => t.id === editId.value)
      payload.done = current ? current.done : false
      const { data } = await updateTodo(editId.value, payload)
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
    await loadTodos()
    router.push('/todos')
  } catch {
    ElMessage.error('操作失败')
  } finally {
    saving.value = false
  }
}

async function toggleDone(todo) {
  try {
    const { data } = await updateTodo(todo.id, {
      title: todo.title,
      content: todo.content || '',
      priority: todo.priority,
      deadline: todo.deadline ? formatDate(todo.deadline) : '',
      done: !todo.done,
    })
    if (data.code === 0) {
      loadTodos()
    }
  } catch {
    ElMessage.error('操作失败')
  }
}

async function handleDelete(id) {
  try {
    await ElMessageBox.confirm('确定删除该待办？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deleteTodo(id)
    if (data.code === 0) {
      ElMessage.success('已删除')
      await loadTodos()
      if (String(route.params.id) === String(id)) {
        router.push('/todos')
      }
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
}

function goBack() {
  router.push('/todos')
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

.todo-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.15s;
}

.todo-item:hover {
  background: #f5f7fa;
}

.todo-item.active {
  background: #ecf5ff;
}

.todo-item.done .todo-item-title {
  text-decoration: line-through;
  color: #999;
}

.todo-item-main {
  flex: 1;
  min-width: 0;
}

.todo-item-title {
  font-size: 14px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.todo-item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.todo-item-deadline {
  font-size: 12px;
  color: #999;
}

.todo-item-del {
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s;
  width: 28px;
  height: 28px;
  padding: 0;
  font-size: 12px;
}

.todo-item:hover .todo-item-del {
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
</style>
