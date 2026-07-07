<template>
  <nav class="side-nav">
    <div class="nav-items">
      <router-link
        v-for="item in items"
        :key="item.path"
        :to="item.path"
        class="nav-item"
        :title="item.label"
      >
        <img :src="item.icon" class="nav-icon" :alt="item.label" />
        <span class="nav-label">{{ item.label }}</span>
      </router-link>
    </div>
    <div class="nav-bottom">
      <slot name="user-menu" />
    </div>
  </nav>
</template>

<script setup>
import chatIcon from '../assets/icons/sidebar/chat.svg'
import noteIcon from '../assets/icons/sidebar/note.svg'
import todoIcon from '../assets/icons/sidebar/todo.svg'
import planIcon from '../assets/icons/sidebar/plan.svg'
import knowledgeIcon from '../assets/icons/sidebar/knowledge.svg'
import scheduleIcon from '../assets/icons/sidebar/schedule.svg'

const items = [
  { path: '/chat', label: '对话', icon: chatIcon },
  { path: '/notes', label: '笔记', icon: noteIcon },
  { path: '/todos', label: '待办', icon: todoIcon },
  { path: '/plans', label: '计划', icon: planIcon },
  { path: '/knowledge', label: '知识库', icon: knowledgeIcon },
  { path: '/schedules', label: '定时', icon: scheduleIcon },
]
</script>

<style scoped>
.side-nav {
  width: 64px;
  flex-shrink: 0;
  background: #fff;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  height: 100%;
  box-sizing: border-box;
}

.nav-items {
  display: flex;
  flex-direction: column;
  padding-top: 8px;
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  height: 64px;
  text-decoration: none;
  color: #909399;
  transition: background 0.15s, color 0.15s;
  position: relative;
}

.nav-item:hover {
  background: #f5f7fa;
  color: #409eff;
}

.nav-item.router-link-active {
  color: #409eff;
  background: #ecf5ff;
}

.nav-item.router-link-active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 8px;
  bottom: 8px;
  width: 3px;
  background: #409eff;
  border-radius: 0 3px 3px 0;
}

.nav-icon {
  width: 22px;
  height: 22px;
  display: block;
  filter: grayscale(1);
  opacity: 0.55;
  transition: filter 0.15s, opacity 0.15s;
}

.nav-item:hover .nav-icon,
.nav-item.router-link-active .nav-icon {
  filter: none;
  opacity: 1;
}

.nav-label {
  font-size: 11px;
  line-height: 1;
}

.nav-bottom {
  padding: 8px 0 16px;
  display: flex;
  justify-content: center;
}

@media (max-width: 768px) {
  .side-nav {
    width: 56px;
  }

  .nav-item {
    height: 56px;
  }

  .nav-label {
    font-size: 10px;
  }
}
</style>
