<template>
  <el-dialog
    :model-value="modelValue"
    @update:model-value="$emit('update:modelValue', $event)"
    title="Skill 管理"
    width="680px"
    :close-on-click-modal="false"
  >
    <!-- 列表视图 -->
    <div v-if="!editing">
      <div class="skill-toolbar">
        <span class="skill-hint">摘要随每轮对话注入；详情由 agent 按需调用 load_skill 加载</span>
        <el-button size="small" @click="startCreate">新建</el-button>
      </div>
      <el-table :data="skills" size="small" v-loading="loading" empty-text="暂无 skill">
        <el-table-column prop="name" label="名称" width="140" />
        <el-table-column prop="summary" label="摘要" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.summary">{{ row.summary }}</span>
            <span v-else style="color:#bbb;">—</span>
          </template>
        </el-table-column>
        <el-table-column label="详情" width="70" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.detail" size="small" type="success">有</el-tag>
            <span v-else style="color:#bbb;">无</span>
          </template>
        </el-table-column>
        <el-table-column label="启用" width="70" align="center">
          <template #default="{ row }">
            <el-switch :model-value="row.enabled" @change="handleToggle(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" align="center">
          <template #default="{ row }">
            <el-button size="small" link @click="startEdit(row)">编辑</el-button>
            <el-button size="small" link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 编辑视图 -->
    <div v-else>
      <el-form label-width="72px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="skill 名称（load_skill 入参）" />
        </el-form-item>
        <el-form-item label="摘要">
          <el-input
            v-model="form.summary"
            type="textarea"
            :rows="2"
            placeholder="一句话描述，会随每轮对话注入上下文"
          />
        </el-form-item>
        <el-form-item label="详情">
          <el-input
            v-model="form.detail"
            type="textarea"
            :rows="6"
            placeholder="详细提示词，agent 按需调用 load_skill 加载。可留空"
          />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" controls-position="right" />
          <span class="sort-hint">数值小的排前面</span>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <div class="skill-edit-actions">
        <el-button @click="editing = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getSkills, createSkill, updateSkill, toggleSkill, deleteSkill } from '../api/skill'

const props = defineProps({ modelValue: Boolean })
const emit = defineEmits(['update:modelValue'])

const skills = ref([])
const loading = ref(false)
const editing = ref(false)
const saving = ref(false)
const form = ref({ id: null, name: '', summary: '', detail: '', enabled: false, sort: 0 })

watch(
  () => props.modelValue,
  (v) => {
    if (v) {
      editing.value = false
      load()
    }
  }
)

async function load() {
  loading.value = true
  try {
    const { data } = await getSkills()
    if (data.code === 0) skills.value = data.data
    else ElMessage.error(data.msg)
  } catch {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function startCreate() {
  form.value = { id: null, name: '', summary: '', detail: '', enabled: false, sort: 0 }
  editing.value = true
}

function startEdit(row) {
  form.value = {
    id: row.id,
    name: row.name,
    summary: row.summary,
    detail: row.detail,
    enabled: row.enabled,
    sort: row.sort,
  }
  editing.value = true
}

async function handleSave() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请填写名称')
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.value.name,
      summary: form.value.summary,
      detail: form.value.detail,
      enabled: form.value.enabled,
      sort: form.value.sort,
    }
    const { data } = form.value.id
      ? await updateSkill(form.value.id, payload)
      : await createSkill(payload)
    if (data.code === 0) {
      ElMessage.success('保存成功')
      editing.value = false
      await load()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function handleToggle(row) {
  try {
    const { data } = await toggleSkill(row.id, !row.enabled)
    if (data.code === 0) {
      row.enabled = !row.enabled
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('操作失败')
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`确认删除 skill「${row.name}」？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    const { data } = await deleteSkill(row.id)
    if (data.code === 0) {
      ElMessage.success('已删除')
      await load()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('删除失败')
  }
}
</script>

<style scoped>
.skill-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.skill-hint {
  font-size: 12px;
  color: #909399;
}

.skill-edit-actions {
  text-align: right;
  margin-top: 8px;
}

.sort-hint {
  margin-left: 8px;
  font-size: 12px;
  color: #909399;
}
</style>
