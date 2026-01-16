<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="app">
    <header class="header fade-in-up">
      <div class="container">
        <div class="logo">
          <h1>Algorithm Platform</h1>
          <span class="subtitle">算 法 管 理 平 台</span>
        </div>
        <nav class="nav">
          <RouterLink to="/algorithms" class="nav-link">
            <span class="icon">◈</span>
            算法
          </RouterLink>
          <RouterLink to="/jobs" class="nav-link">
            <span class="icon">◉</span>
            任务
          </RouterLink>
          <RouterLink to="/data" class="nav-link">
            <span class="icon">◫</span>
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
}

.header {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-subtle);
  padding: var(--space-lg) 0;
  position: sticky;
  top: 0;
  z-index: 100;
  backdrop-filter: blur(10px);
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
  font-weight: 700;
  letter-spacing: -0.02em;
  margin: 0;
  background: linear-gradient(90deg, var(--accent-primary), var(--accent-secondary));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.subtitle {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.15em;
}

.nav {
  display: flex;
  gap: var(--space-md);
  margin-top: var(--space-lg);
}

.nav-link {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius-sm);
  transition: all var(--transition-base);
  position: relative;
  overflow: hidden;
}

.nav-link::before {
  content: '';
  position: absolute;
  bottom: 0;
  left: 50%;
  width: 0;
  height: 2px;
  background: var(--accent-primary);
  transition: all var(--transition-base);
  transform: translateX(-50%);
}

.nav-link:hover {
  color: var(--text-primary);
  background: var(--bg-tertiary);
}

.nav-link:hover::before {
  width: 100%;
}

.nav-link.router-link-active {
  color: var(--accent-primary);
  background: rgba(139, 92, 246, 0.1);
}

.nav-link.router-link-active::before {
  width: 100%;
}

.icon {
  font-size: var(--font-size-lg);
  font-weight: 300;
}

.main {
  flex: 1;
  padding: var(--space-2xl) 0;
}

.footer {
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-subtle);
  padding: var(--space-lg) 0;
  text-align: center;
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
