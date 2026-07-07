<template>
  <div style="display:flex;justify-content:center;align-items:center;height:100vh;">
    <el-card style="width:400px;">
      <template #header><span style="font-size:20px;font-weight:bold;">登录</span></template>
      <el-form :model="form" @submit.prevent="handleLogin">
        <el-form-item label="用户名">
          <el-input v-model="form.username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" native-type="submit" :loading="loading" style="width:100%;">登录</el-button>
        </el-form-item>
        <div style="text-align:center;">
          <el-link type="primary" @click="$router.push('/register')">没有账号？去注册</el-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { login } from '../api/user'

const router = useRouter()
const form = ref({ username: '', password: '' })
const loading = ref(false)

async function handleLogin() {
  if (!form.value.username || !form.value.password) {
    ElMessage.warning('请填写用户名和密码')
    return
  }
  loading.value = true
  try {
    const { data } = await login(form.value.username, form.value.password)
    if (data.code === 0) {
      ElMessage.success('登录成功')
      router.push('/')
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('网络错误')
  } finally {
    loading.value = false
  }
}
</script>
