<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="app">
    <header class="header">
      <div class="container">
        <div class="logo">
          <h1>算法管理平台</h1>
          <span class="subtitle">Algorithm Platform</span>
        </div>
        <nav class="nav">
          <RouterLink to="/data" class="nav-link">
            <span class="icon">📁</span>
            数据
          </RouterLink>
          <RouterLink to="/algorithms" class="nav-link">
            <span class="icon">📊</span>
            算法
          </RouterLink>
          <RouterLink to="/jobs" class="nav-link">
            <span class="icon">⚡</span>
            任务
          </RouterLink>
        </nav>
      </div>
    </header>
    <main class="main">
      <div class="container">
        <RouterView v-slot="{ Component, route }">
          <Transition name="page" mode="out-in">
            <component :is="Component" :key="route.path" />
          </Transition>
        </RouterView>
      </div>
    </main>
    <footer class="footer">
      <div class="container">
        <p>Built with Vue3 + Go + Docker</p>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-secondary);
}

.header {
  background: var(--bg-card);
  border-bottom: 1px solid var(--border-light);
  padding: var(--space-lg) 0;
  position: sticky;
  top: 0;
  z-index: 100;
  box-shadow: var(--shadow-sm);
}

.container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 var(--space-lg);
}

.logo {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.logo h1 {
  font-size: var(--font-size-xl);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.02em;
}

.sub-title {
  font-size: 11px;
  font-weight: 400;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.1em;
}
</style>
