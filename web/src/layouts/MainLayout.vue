<template>
  <div class="app-shell">
    <SideNav>
      <template #user-menu>
        <UserMenu />
      </template>
    </SideNav>
    <div class="app-content">
      <router-view v-slot="{ Component }">
        <keep-alive>
          <component :is="Component" />
        </keep-alive>
      </router-view>
    </div>
  </div>
</template>

<script setup>
import SideNav from '../components/SideNav.vue'
import UserMenu from '../components/UserMenu.vue'
</script>

<style scoped>
.app-shell {
  display: flex;
  height: 100vh;
  width: 100%;
  overflow: hidden;
  background: #f5f5f5;
}

.app-content {
  flex: 1;
  min-width: 0;
  height: 100vh;
  overflow: hidden;
}
</style>
