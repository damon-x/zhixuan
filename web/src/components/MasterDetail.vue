<template>
  <div class="md-layout">
    <div class="md-list" :class="{ 'mobile-hidden': isMobile && hasSelection }">
      <slot name="list" />
    </div>
    <div class="md-detail" :class="{ 'mobile-hidden': isMobile && !hasSelection }">
      <div v-if="isMobile && hasSelection" class="md-back" @click="$emit('back')">
        <span class="md-back-arrow">&larr;</span> 返回
      </div>
      <slot name="detail" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'

defineProps({
  hasSelection: { type: Boolean, default: false },
})
defineEmits(['back'])

const isMobile = ref(false)
let mql = null

function onMediaChange(e) {
  isMobile.value = e.matches
}

onMounted(() => {
  mql = window.matchMedia('(max-width: 768px)')
  isMobile.value = mql.matches
  mql.addEventListener('change', onMediaChange)
})

onBeforeUnmount(() => {
  if (mql) mql.removeEventListener('change', onMediaChange)
})
</script>

<style scoped>
.md-layout {
  display: flex;
  height: 100%;
  width: 100%;
  overflow: hidden;
}

.md-list {
  width: 280px;
  flex-shrink: 0;
  height: 100%;
  overflow: hidden;
  background: #fff;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
}

.md-detail {
  flex: 1;
  min-width: 0;
  height: 100%;
  overflow: hidden;
  background: #f5f5f5;
  display: flex;
  flex-direction: column;
}

.md-back {
  display: none;
  align-items: center;
  gap: 6px;
  padding: 10px 14px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  font-size: 14px;
  color: #409eff;
  cursor: pointer;
  flex-shrink: 0;
}

.md-back-arrow {
  font-size: 16px;
}

@media (max-width: 768px) {
  .md-list {
    width: 100%;
    border-right: none;
  }

  .md-detail {
    width: 100%;
  }

  .mobile-hidden {
    display: none;
  }

  .md-back {
    display: flex;
  }
}
</style>
