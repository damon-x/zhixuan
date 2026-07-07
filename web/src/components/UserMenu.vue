<template>
  <div class="user-menu">
    <el-dropdown trigger="click" placement="top-start">
      <div class="avatar-wrap">
        <el-avatar :size="36" class="user-avatar">{{ avatarText }}</el-avatar>
      </div>
      <template #dropdown>
        <el-dropdown-menu>
          <div class="dropdown-username">{{ username }}</div>
          <el-dropdown-item @click="showQQBotDialog = true">绑定QQBot</el-dropdown-item>
          <el-dropdown-item @click="showWeChatDialog = true">绑定微信</el-dropdown-item>
          <el-dropdown-item @click="logout" divided>退出</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <!-- QQBot Bind Dialog -->
    <el-dialog v-model="showQQBotDialog" title="绑定QQBot" width="420px" :close-on-click-modal="false" @close="onQQBotDialogClose">
      <div v-if="qqBotBound" class="qqbot-status">
        <el-result icon="success" title="已绑定QQBot" sub-text="QQ机器人已成功绑定">
          <template #extra>
            <div style="display: flex; flex-direction: column; align-items: center; gap: 12px;">
              <div style="display: flex; align-items: center; gap: 8px;">
                <span style="font-size: 14px;">开启QQ对话</span>
                <el-switch v-model="qqChatEnabled" @change="handleToggleQQChat" :loading="qqChatEnabled !== qqChatRunning" />
              </div>
              <el-button type="primary" @click="showQQBotDialog = false">关闭</el-button>
            </div>
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

    <!-- WeChat Bind Dialog -->
    <el-dialog v-model="showWeChatDialog" title="绑定微信" width="420px" :close-on-click-modal="false" @close="onWeChatDialogClose">
      <div v-if="wxBound" class="qqbot-status">
        <el-result icon="success" title="已绑定微信" sub-text="微信已成功绑定">
          <template #extra>
            <div style="display: flex; flex-direction: column; align-items: center; gap: 12px;">
              <div style="display: flex; align-items: center; gap: 8px;">
                <span style="font-size: 14px;">开启微信对话</span>
                <el-switch v-model="wxChatEnabled" @change="handleToggleWeChatChat" :loading="wxChatEnabled !== wxChatRunning" />
              </div>
              <el-button type="primary" @click="showWeChatDialog = false">关闭</el-button>
            </div>
          </template>
        </el-result>
      </div>
      <div v-else-if="wxBinding">
        <div class="qqbot-code-area">
          <p>请用微信扫描以下二维码：</p>
          <div class="wx-qrcode-img">
            <img :src="wxQRDataURL" alt="QR Code" style="max-width: 260px;" />
          </div>
          <p class="qqbot-tip">等待扫码中...</p>
        </div>
      </div>
      <div v-else style="text-align: center; padding: 20px 0;">
        <el-button type="primary" @click="handleWeChatGetQR">获取二维码</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getMe, logout as logoutApi } from '../api/user'
import { startQQBotBind, checkQQBotBind, getQQBotStatus, toggleQQBotChat, getQQBotChatStatus } from '../api/qqbot'
import { getWeChatQRCode, checkWeChatBind, getWeChatStatus, toggleWeChatChat, getWeChatChatStatus } from '../api/wechat'
import QRCode from 'qrcode'

const router = useRouter()
const username = ref('')
const avatarText = computed(() => (username.value || '?').charAt(0).toUpperCase())

const showQQBotDialog = ref(false)
const qqBotBound = ref(false)
const qqBotBinding = ref(false)
const qqBotCode = ref('')
const qqBotAppId = ref('')
const qqBotAppSecret = ref('')
const qqChatEnabled = ref(false)
const qqChatRunning = ref(false)
let qqBotPollTimer = null

const showWeChatDialog = ref(false)
const wxBound = ref(false)
const wxBinding = ref(false)
const wxQRDataURL = ref('')
const wxChatEnabled = ref(false)
const wxChatRunning = ref(false)
let wxPollTimer = null

onMounted(async () => {
  try {
    const { data } = await getMe()
    if (data.code === 0) {
      username.value = data.data.username
    } else {
      router.push('/login')
      return
    }
  } catch {
    router.push('/login')
    return
  }
  loadQQBotStatus()
  loadQQBotChatStatus()
  loadWeChatStatus()
  loadWeChatChatStatus()
})

async function loadQQBotStatus() {
  try {
    const { data } = await getQQBotStatus()
    if (data.code === 0) {
      qqBotBound.value = data.data.bound
      if (data.data.app_id) qqBotAppId.value = data.data.app_id
    }
  } catch { /* ignore */ }
}

async function loadQQBotChatStatus() {
  try {
    const { data } = await getQQBotChatStatus()
    if (data.code === 0) {
      qqChatEnabled.value = data.data.enabled
      qqChatRunning.value = data.data.running
    }
  } catch { /* ignore */ }
}

async function handleToggleQQChat(enabled) {
  try {
    const { data } = await toggleQQBotChat(enabled)
    if (data.code === 0) {
      qqChatEnabled.value = enabled
      await loadQQBotChatStatus()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('操作失败')
  }
}

async function handleStartBind() {
  try {
    const { data } = await startQQBotBind(qqBotAppId.value, qqBotAppSecret.value)
    if (data.code === 0) {
      qqBotCode.value = data.data.code
      qqBotBinding.value = true
      startQQBotPoll()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('启动绑定失败')
  }
}

function startQQBotPoll() {
  stopQQBotPoll()
  qqBotPollTimer = setInterval(async () => {
    try {
      const { data } = await checkQQBotBind()
      if (data.code === 0 && data.data.bound) {
        qqBotBound.value = true
        qqBotBinding.value = false
        stopQQBotPoll()
        ElMessage.success('QQBot 绑定成功')
      }
    } catch { /* ignore */ }
  }, 2000)
}

function stopQQBotPoll() {
  if (qqBotPollTimer) {
    clearInterval(qqBotPollTimer)
    qqBotPollTimer = null
  }
}

function onQQBotDialogClose() {
  stopQQBotPoll()
  qqBotBinding.value = false
}

async function loadWeChatStatus() {
  try {
    const { data } = await getWeChatStatus()
    if (data.code === 0) {
      wxBound.value = data.data.bound
    }
  } catch { /* ignore */ }
}

async function loadWeChatChatStatus() {
  try {
    const { data } = await getWeChatChatStatus()
    if (data.code === 0) {
      wxChatEnabled.value = data.data.enabled
      wxChatRunning.value = data.data.running
    }
  } catch { /* ignore */ }
}

async function handleToggleWeChatChat(enabled) {
  try {
    const { data } = await toggleWeChatChat(enabled)
    if (data.code === 0) {
      wxChatEnabled.value = enabled
      await loadWeChatChatStatus()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('操作失败')
  }
}

async function handleWeChatGetQR() {
  try {
    const { data } = await getWeChatQRCode()
    if (data.code === 0) {
      const qrURL = data.data.qr_image
      try {
        wxQRDataURL.value = await QRCode.toDataURL(qrURL, { width: 260, margin: 2 })
      } catch {
        wxQRDataURL.value = ''
      }
      wxBinding.value = true
      startWeChatPoll()
    } else {
      ElMessage.error(data.msg)
    }
  } catch {
    ElMessage.error('获取二维码失败')
  }
}

function startWeChatPoll() {
  stopWeChatPoll()
  wxPollTimer = setInterval(async () => {
    try {
      const { data } = await checkWeChatBind()
      if (data.code === 0 && data.data.bound) {
        wxBound.value = true
        wxBinding.value = false
        stopWeChatPoll()
        ElMessage.success('微信绑定成功')
      }
    } catch { /* ignore */ }
  }, 2000)
}

function stopWeChatPoll() {
  if (wxPollTimer) {
    clearInterval(wxPollTimer)
    wxPollTimer = null
  }
}

function onWeChatDialogClose() {
  stopWeChatPoll()
  wxBinding.value = false
}

async function logout() {
  try {
    await logoutApi()
  } catch { /* ignore */ }
  ElMessage.success('已退出登录')
  router.push('/login')
}
</script>

<style scoped>
.user-menu {
  display: flex;
  justify-content: center;
}

.avatar-wrap {
  cursor: pointer;
  outline: none;
}

.user-avatar {
  background: #409eff;
  color: #fff;
  font-weight: 600;
  cursor: pointer;
}

.dropdown-username {
  padding: 8px 16px 4px;
  font-size: 13px;
  color: #909399;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 4px;
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.qqbot-code-area {
  text-align: center;
  padding: 20px 0;
}

.qqbot-code {
  font-size: 36px;
  font-weight: bold;
  color: #409eff;
  letter-spacing: 8px;
  margin: 16px 0;
}

.qqbot-tip {
  color: #999;
  font-size: 13px;
}
</style>
