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
          <RouterLink to="/algorithms" class="nav-link">
            <span class="icon">📊</span>
            算法
          </RouterLink>
          <RouterLink to="/jobs" class="nav-link">
            <span class="icon">⚡</span>
            任务
          </RouterLink>
          <RouterLink to="/data" class="nav-link">
            <span class="icon">📁</span>
            数据
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

.subtitle {
  font-size: 11px;
  font-weight: 400;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.1em;
}

.nav {
  display: flex;
  gap: var(--space-md);
  margin-top: var(--space-md);
}

.nav-link {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius-md);
  transition: all var(--transition-base);
  position: relative;
}

.nav-link:hover {
  color: var(--accent-primary);
  background: rgba(64, 158, 255, 0.05);
}

.nav-link.router-link-active {
  color: var(--accent-primary);
  background: rgba(64, 158, 255, 0.1);
  font-weight: 500;
}

.icon {
  font-size: var(--font-size-lg);
  font-weight: 400;
}

.main {
  flex: 1;
  padding: var(--space-2xl) 0;
}

.footer {
  background: var(--bg-card);
  border-top: 1px solid var(--border-light);
  padding: var(--space-lg) 0;
  text-align: center;
  margin-top: auto;
}

.footer p {
  color: var(--text-muted);
  font-size: var(--font-size-sm);
  margin: 0;
}

.page-enter-active,
.page-leave-active {
  transition: all 0.3s ease;
}

.page-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
