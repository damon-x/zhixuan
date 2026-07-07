<template>
  <el-dialog
    :model-value="modelValue"
    @update:model-value="$emit('update:modelValue', $event)"
    title="记忆管理"
    width="680px"
    :close-on-click-modal="false"
  >
    <!-- 列表视图 -->
    <div v-if="!current">
      <div class="mem-toolbar">
        <span class="mem-hint">由记忆 agent 自动整理；可查看与删除</span>
        <span class="mem-count">{{ memories.length }} 条</span>
      </div>
      <el-table :data="memories" size="small" v-loading="loading" empty-text="暂无记忆" max-height="460">
        <el-table-column label="类型" width="90">
          <template #default="{ row }">
            <el-tag :type="typeTag(row.type)" size="small" effect="plain">{{ typeLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="content" label="内容" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ row.content }}</span>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="140">
          <template #default="{ row }">
            <span class="mem-time">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center">
          <template #default="{ row }">
            <el-button size="small" link @click="current = row">查看</el-button>
            <el-button size="small" link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 详情视图 -->
    <div v-else class="mem-detail">
      <div class="mem-detail-row">
        <span class="mem-detail-label">类型</span>
        <el-tag :type="typeTag(current.type)" size="small">{{ typeLabel(current.type) }}</el-tag>
      </div>
      <div class="mem-detail-row">
        <span class="mem-detail-label">内容</span>
        <div class="mem-detail-content">{{ current.content }}</div>
      </div>
      <div v-if="current.tags" class="mem-detail-row">
        <span class="mem-detail-label">标签</span>
        <div class="mem-detail-tags">{{ current.tags }}</div>
      </div>
      <div class="mem-detail-row">
        <span class="mem-detail-label">记录时间</span>
        <div class="mem-detail-time">{{ formatTime(current.created_at) }}</div>
      </div>
      <div class="mem-detail-actions">
        <el-button size="small" type="danger" @click="handleDelete(current)">删除</el-button>
        <el-button size="small" @click="current = null">返回</el-button>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getMemories, deleteMemory } from '../api/memory'

const props = defineProps({ modelValue: Boolean })
const emit = defineEmits(['update:modelValue'])

const memories = ref([])
const loading = ref(false)
const current = ref(null)

watch(
  () => props.modelValue,
  (v) => {
    if (v) {
      current.value = null
      load()
    }
  }
)

async function load() {
  loading.value = true
  try {
    const { data } = await getMemories()
    if (data.code === 0) memories.value = data.data || []
    else ElMessage.error(data.msg)
  } catch {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm('确定删除该记忆？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deleteMemory(row.id)
    if (data.code === 0) {
      ElMessage.success('已删除')
      if (current.value && current.value.id === row.id) current.value = null
      await load()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
}

function typeLabel(t) {
  return {
    preference: '偏好',
    fact: '事实',
    relationship: '关系',
    event: '事件',
    goal: '目标',
  }[t] || t
}

function typeTag(t) {
  return {
    preference: 'success',
    fact: 'info',
    relationship: 'warning',
    event: 'primary',
    goal: 'danger',
  }[t] || 'info'
}

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>

<style scoped>
.mem-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.mem-hint {
  font-size: 12px;
  color: #909399;
}

.mem-count {
  font-size: 12px;
  color: #999;
}

.mem-time {
  font-size: 12px;
  color: #909399;
}

.mem-detail {
  padding: 4px 2px;
}

.mem-detail-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 20px;
}

.mem-detail-label {
  flex-shrink: 0;
  width: 64px;
  font-size: 13px;
  color: #909399;
  line-height: 24px;
}

.mem-detail-content {
  font-size: 14px;
  color: #303133;
  line-height: 1.6;
  flex: 1;
  min-width: 0;
  word-break: break-word;
}

.mem-detail-tags {
  font-size: 13px;
  color: #606266;
  line-height: 24px;
}

.mem-detail-time {
  font-size: 13px;
  color: #909399;
  line-height: 24px;
}

.mem-detail-actions {
  text-align: right;
  margin-top: 8px;
}
</style>
