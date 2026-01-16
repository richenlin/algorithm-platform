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
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: var(--space-xl) 0;
  position: sticky;
  top: 0;
  z-index: 100;
  box-shadow: 0 4px 20px rgba(102, 126, 234, 0.3);
}

.container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 var(--space-lg);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.logo {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.logo h1 {
  font-size: var(--font-size-2xl);
  font-weight: 700;
  color: #ffffff;
  margin: 0;
  letter-spacing: -0.02em;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.subtitle {
  font-size: 12px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.85);
  text-transform: uppercase;
  letter-spacing: 0.15em;
}

.nav {
  display: flex;
  gap: var(--space-md);
}

.nav-link {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  color: rgba(255, 255, 255, 0.85);
  text-decoration: none;
  border-radius: 12px;
  font-weight: 500;
  transition: all 0.3s ease;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.nav-link:hover {
  background: rgba(255, 255, 255, 0.2);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.nav-link.router-link-active {
  background: #ffffff;
  color: #667eea;
  font-weight: 600;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.nav-link .icon {
  font-size: 18px;
}
</style>
