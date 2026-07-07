<template>
  <MasterDetail :has-selection="hasSelection" @back="goBack">
    <template #list>
      <div class="list-header">
        <span class="list-title">定时任务</span>
        <el-button size="small" @click="router.push('/schedules/new')">新增</el-button>
      </div>
      <div class="list-scroll">
        <div v-if="schedules.length === 0" class="list-empty">暂无定时任务</div>
        <div
          v-for="item in schedules"
          :key="item.id"
          class="sched-item"
          :class="{ active: String(route.params.id) === String(item.id) }"
          @click="router.push(`/schedules/${item.id}`)"
        >
          <div class="sched-item-main">
            <div class="sched-item-name">{{ item.name }}</div>
            <div class="sched-item-meta">
              <el-tag size="small">{{ typeLabel(item.type) }}</el-tag>
              <el-tag :type="item.schedule_mode === 'once' ? 'warning' : 'info'" size="small">{{ item.schedule_mode === 'once' ? '单次' : '周期' }}</el-tag>
              <span class="sched-cron">{{ item.schedule_mode === 'once' ? `执行: ${item.cron}` : `Cron: ${item.cron}` }}</span>
              <el-tag v-if="item.qq_notify" type="success" size="small">QQ通知</el-tag>
            </div>
          </div>
          <div class="sched-item-actions">
            <el-switch :model-value="item.enabled" @click.stop @change="toggleEnabled(item)" />
            <el-button size="small" @click.stop="showLogs(item)">日志</el-button>
          </div>
        </div>
      </div>
    </template>

    <template #detail>
      <div v-if="!hasSelection" class="detail-empty">
        <div class="detail-empty-text">选择或新增一个定时任务</div>
      </div>
      <template v-else>
        <div class="detail-header">
          <span class="detail-title">{{ isEdit ? '编辑定时任务' : '新增定时任务' }}</span>
          <div class="detail-actions">
            <el-button v-if="isEdit" size="small" type="danger" @click="handleDelete(editId)">删除</el-button>
            <el-button size="small" @click="goBack">取消</el-button>
            <el-button type="primary" size="small" @click="handleSubmit" :loading="saving">保存</el-button>
          </div>
        </div>
        <div class="detail-body">
          <el-form label-width="100px">
            <el-form-item label="名称">
              <el-input v-model="form.name" placeholder="请输入任务名称" />
            </el-form-item>
            <el-form-item label="任务类型">
              <el-select v-model="form.type" style="width: 100%">
                <el-option v-for="t in types" :key="t.value" :label="t.label" :value="t.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="调度方式">
              <el-radio-group v-model="form.schedule_mode">
                <el-radio value="cron">周期（Cron）</el-radio>
                <el-radio value="once">单次</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="form.schedule_mode === 'once'" label="执行时间">
              <el-date-picker v-model="form.cron" type="datetime" placeholder="选择执行时间" value-format="YYYY-MM-DD HH:mm" style="width: 100%" :disabled-date="disablePastDate" />
            </el-form-item>
            <template v-else>
              <el-form-item label="执行频率">
                <el-select v-model="cronPreset" style="width: 100%" @change="onCronPresetChange">
                  <el-option label="每隔几分钟" value="interval_minute" />
                  <el-option label="每隔几小时" value="interval_hour" />
                  <el-option label="每天" value="daily" />
                  <el-option label="每周" value="weekly" />
                  <el-option label="每月" value="monthly" />
                  <el-option label="自定义 Cron" value="custom" />
                </el-select>
              </el-form-item>
              <el-form-item v-if="cronPreset === 'interval_minute'" label="间隔(分钟)">
                <el-input-number v-model="cronConfig.interval" :min="1" :max="59" @change="buildCron" />
              </el-form-item>
              <el-form-item v-if="cronPreset === 'interval_hour'" label="间隔(小时)">
                <el-input-number v-model="cronConfig.interval" :min="1" :max="23" @change="buildCron" />
              </el-form-item>
              <el-form-item v-if="cronPreset === 'daily'" label="执行时间">
                <el-time-picker v-model="cronConfig.time" format="HH:mm" value-format="HH:mm" placeholder="选择时间" @change="buildCron" />
              </el-form-item>
              <el-form-item v-if="cronPreset === 'weekly'" label="星期">
                <el-select v-model="cronConfig.dayOfWeek" style="width: 100%" @change="buildCron">
                  <el-option v-for="d in weekDays" :key="d.value" :label="d.label" :value="d.value" />
                </el-select>
              </el-form-item>
              <el-form-item v-if="cronPreset === 'weekly'" label="执行时间">
                <el-time-picker v-model="cronConfig.time" format="HH:mm" value-format="HH:mm" placeholder="选择时间" @change="buildCron" />
              </el-form-item>
              <el-form-item v-if="cronPreset === 'monthly'" label="日期">
                <el-input-number v-model="cronConfig.dayOfMonth" :min="1" :max="31" @change="buildCron" />
              </el-form-item>
              <el-form-item v-if="cronPreset === 'monthly'" label="执行时间">
                <el-time-picker v-model="cronConfig.time" format="HH:mm" value-format="HH:mm" placeholder="选择时间" @change="buildCron" />
              </el-form-item>
              <el-form-item v-if="cronPreset === 'custom'" label="Cron 表达式">
                <el-input v-model="form.cron" placeholder="如 */5 * * * *（每5分钟）" />
              </el-form-item>
            </template>
            <el-form-item label="Prompt">
              <el-input v-model="form.params" type="textarea" :rows="4" placeholder="请输入自然语言指令" />
            </el-form-item>
            <el-form-item label="QQ 通知">
              <el-switch v-model="form.qq_notify" />
            </el-form-item>
          </el-form>
        </div>
      </template>
    </template>
  </MasterDetail>

  <el-dialog v-model="logDialogVisible" title="执行日志" width="600px">
    <div v-if="logs.length === 0" class="log-empty">暂无执行日志</div>
    <div v-else class="log-list">
      <div v-for="log in logs" :key="log.id" class="log-item">
        <div class="log-time">{{ formatTime(log.created_at) }}</div>
        <div v-if="log.error" class="log-error">{{ log.error }}</div>
        <div v-else class="log-result">{{ log.result }}</div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, onActivated } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getSchedules, createSchedule, updateSchedule, deleteSchedule, getScheduleTypes, getScheduleLogs } from '../api/schedule'
import MasterDetail from '../components/MasterDetail.vue'

const route = useRoute()
const router = useRouter()
const schedules = ref([])
const types = ref([])
const saving = ref(false)
const editId = ref(null)
const form = ref({ name: '', type: 'agent', schedule_mode: 'cron', cron: '', params: '', qq_notify: false })

const cronPreset = ref('daily')
const cronConfig = ref({ interval: 5, time: '09:00', dayOfWeek: '1', dayOfMonth: 1 })

const logDialogVisible = ref(false)
const logs = ref([])

const weekDays = [
  { label: '周一', value: '1' },
  { label: '周二', value: '2' },
  { label: '周三', value: '3' },
  { label: '周四', value: '4' },
  { label: '周五', value: '5' },
  { label: '周六', value: '6' },
  { label: '周日', value: '0' },
]

const hasSelection = computed(() => route.params.id !== undefined)
const isEdit = computed(() => {
  const id = route.params.id
  return id !== undefined && id !== 'new'
})

onActivated(() => {
  loadSchedules()
  loadTypes()
})

watch(() => route.params.id, (newId) => {
  if (!route.path.startsWith('/schedules')) return
  loadForm(newId)
}, { immediate: true })

function onCronPresetChange() {
  buildCron()
}

function buildCron() {
  const t = cronConfig.value.time || '00:00'
  const [h, m] = t.split(':').map(Number)
  switch (cronPreset.value) {
    case 'interval_minute':
      form.value.cron = `*/${cronConfig.value.interval} * * * *`
      break
    case 'interval_hour':
      form.value.cron = `0 */${cronConfig.value.interval} * * *`
      break
    case 'daily':
      form.value.cron = `${m} ${h} * * *`
      break
    case 'weekly':
      form.value.cron = `${m} ${h} * * ${cronConfig.value.dayOfWeek}`
      break
    case 'monthly':
      form.value.cron = `${m} ${h} ${cronConfig.value.dayOfMonth} * *`
      break
  }
}

async function loadSchedules() {
  try {
    const { data } = await getSchedules()
    if (data.code === 0) {
      schedules.value = data.data || []
    }
  } catch {
    ElMessage.error('加载定时任务失败')
  }
}

async function loadTypes() {
  try {
    const { data } = await getScheduleTypes()
    if (data.code === 0) {
      types.value = data.data || []
    }
  } catch { /* ignore */ }
}

function typeLabel(type) {
  const t = types.value.find(item => item.value === type)
  return t ? t.label : type
}

function disablePastDate(date) {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return date.getTime() < today.getTime()
}

function loadForm(id) {
  if (id === undefined || id === 'new') {
    editId.value = null
    form.value = { name: '', type: 'agent', schedule_mode: 'cron', cron: '', params: '', qq_notify: false }
    cronPreset.value = 'daily'
    cronConfig.value = { interval: 5, time: '09:00', dayOfWeek: '1', dayOfMonth: 1 }
    buildCron()
    return
  }
  const item = schedules.value.find(s => String(s.id) === String(id))
  if (item) {
    editId.value = item.id
    let params = ''
    if (item.params) {
      try {
        const obj = JSON.parse(item.params)
        params = obj.prompt || ''
      } catch { params = item.params }
    }
    form.value = {
      name: item.name,
      type: item.type,
      schedule_mode: item.schedule_mode || 'cron',
      cron: item.cron,
      params,
      qq_notify: item.qq_notify,
    }
    if (item.schedule_mode !== 'once' && item.cron) {
      parseCronToConfig(item.cron)
    }
  } else {
    editId.value = Number(id)
    form.value = { name: '', type: 'agent', schedule_mode: 'cron', cron: '', params: '', qq_notify: false }
    buildCron()
  }
}

function parseCronToConfig(cron) {
  const parts = cron.trim().split(/\s+/)
  if (parts.length !== 5) {
    cronPreset.value = 'custom'
    return
  }
  const [min, hour, day, month, dow] = parts
  if (/^\*\/\d+$/.test(min) && hour === '*' && day === '*' && month === '*' && dow === '*') {
    cronPreset.value = 'interval_minute'
    cronConfig.value.interval = parseInt(min.slice(2))
  } else if (/^\d+$/.test(min) && /^\*\/\d+$/.test(hour) && day === '*' && month === '*' && dow === '*') {
    cronPreset.value = 'interval_hour'
    cronConfig.value.interval = parseInt(hour.slice(2))
    cronConfig.value.time = `${hour.startsWith('*/') ? '00' : hour.padStart(2, '0')}:${min.padStart(2, '0')}`
  } else if (/^\d+$/.test(min) && /^\d+$/.test(hour) && day === '*' && month === '*' && dow === '*') {
    cronPreset.value = 'daily'
    cronConfig.value.time = `${hour.padStart(2, '0')}:${min.padStart(2, '0')}`
  } else if (/^\d+$/.test(min) && /^\d+$/.test(hour) && day === '*' && month === '*' && /^\d+$/.test(dow)) {
    cronPreset.value = 'weekly'
    cronConfig.value.dayOfWeek = dow
    cronConfig.value.time = `${hour.padStart(2, '0')}:${min.padStart(2, '0')}`
  } else if (/^\d+$/.test(min) && /^\d+$/.test(hour) && /^\d+$/.test(day) && month === '*' && dow === '*') {
    cronPreset.value = 'monthly'
    cronConfig.value.dayOfMonth = parseInt(day)
    cronConfig.value.time = `${hour.padStart(2, '0')}:${min.padStart(2, '0')}`
  } else {
    cronPreset.value = 'custom'
  }
}

async function handleSubmit() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请输入名称')
    return
  }
  if (!form.value.cron.trim()) {
    ElMessage.warning(form.value.schedule_mode === 'once' ? '请选择执行时间' : '请输入 Cron 表达式')
    return
  }
  if (form.value.schedule_mode === 'cron' && cronPreset.value !== 'custom') {
    buildCron()
  }
  saving.value = true
  try {
    const paramsJSON = JSON.stringify({ prompt: form.value.params })
    const payload = {
      name: form.value.name,
      type: form.value.type,
      schedule_mode: form.value.schedule_mode,
      cron: form.value.cron,
      params: paramsJSON,
      qq_notify: form.value.qq_notify,
    }
    if (isEdit.value) {
      const current = schedules.value.find(s => s.id === editId.value)
      payload.enabled = current ? current.enabled : true
      const { data } = await updateSchedule(editId.value, payload)
      if (data.code !== 0) {
        ElMessage.error(data.msg)
        return
      }
      ElMessage.success('更新成功')
    } else {
      const { data } = await createSchedule(payload)
      if (data.code !== 0) {
        ElMessage.error(data.msg)
        return
      }
      ElMessage.success('创建成功')
    }
    await loadSchedules()
    router.push('/schedules')
  } catch {
    ElMessage.error('操作失败')
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(item) {
  try {
    const { data } = await updateSchedule(item.id, {
      name: item.name,
      type: item.type,
      schedule_mode: item.schedule_mode || 'cron',
      cron: item.cron,
      params: item.params,
      enabled: !item.enabled,
      qq_notify: item.qq_notify,
    })
    if (data.code === 0) {
      loadSchedules()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('操作失败')
  }
}

async function handleDelete(id) {
  try {
    await ElMessageBox.confirm('确定删除该定时任务？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deleteSchedule(id)
    if (data.code === 0) {
      ElMessage.success('已删除')
      await loadSchedules()
      if (String(route.params.id) === String(id)) {
        router.push('/schedules')
      }
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
}

async function showLogs(item) {
  try {
    const { data } = await getScheduleLogs(item.id)
    if (data.code === 0) {
      logs.value = data.data || []
      logDialogVisible.value = true
    }
  } catch {
    ElMessage.error('加载日志失败')
  }
}

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

function goBack() {
  router.push('/schedules')
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

.sched-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.15s;
}

.sched-item:hover {
  background: #f5f7fa;
}

.sched-item.active {
  background: #ecf5ff;
}

.sched-item-main {
  flex: 1;
  min-width: 0;
}

.sched-item-name {
  font-size: 14px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sched-item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  flex-wrap: wrap;
}

.sched-cron {
  font-size: 12px;
  color: #999;
}

.sched-item-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
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

.log-empty {
  text-align: center;
  color: #999;
  padding: 24px 0;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.log-item {
  border-bottom: 1px solid #eee;
  padding-bottom: 12px;
}

.log-item:last-child {
  border-bottom: none;
}

.log-time {
  font-size: 12px;
  color: #999;
  margin-bottom: 4px;
}

.log-result {
  font-size: 14px;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-error {
  font-size: 14px;
  color: #f56c6c;
}
</style>
