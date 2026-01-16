<script setup lang="ts">
import { RouterView, RouterLink } from 'vue-router'
</script>

<template>
  <div class="app">
    <header class="header">
      <div class="container">
        <h1 class="logo">算法管理平台</h1>
        <nav class="nav">
          <RouterLink to="/algorithms" class="nav-link">算法</RouterLink>
          <RouterLink to="/jobs" class="nav-link">任务</RouterLink>
          <RouterLink to="/data" class="nav-link">数据</RouterLink>
        </nav>
      </div>
    </header>
    <main class="main">
      <div class="container">
        <RouterView />
      </div>
    </main>
  </div>
</template>

<style scoped>
.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.header {
  background-color: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  padding: calc(var(--grid) * 2) 0;
}

.container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 calc(var(--grid) * 3);
}

.logo {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.nav {
  display: flex;
  gap: calc(var(--grid) * 3);
  margin-top: calc(var(--grid) * 2);
}

.nav-link {
  color: var(--text-secondary);
  font-size: 14px;
  padding: calc(var(--grid) / 2) 0;
  border-bottom: 2px solid transparent;
  transition: all 0.2s ease;
}

.nav-link:hover {
  color: var(--text-primary);
  border-bottom-color: var(--border);
}

.nav-link.router-link-active {
  color: var(--accent);
  border-bottom-color: var(--accent);
}

.main {
  flex: 1;
  padding: calc(var(--grid) * 4) 0;
}
</style>
