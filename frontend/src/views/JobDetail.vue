<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useJobStore } from '../stores/job'

const route = useRoute()
const jobStore = useJobStore()
const jobId = computed(() => route.params.id as string)

const job = computed(() => jobStore.currentJob)
const isLoading = computed(() => jobStore.loading)
const showLogs = ref(false)
let refreshInterval: number | null = null

onMounted(async () => {
  await fetchJobDetail()
  startAutoRefresh()
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})

async function fetchJobDetail() {
  await jobStore.fetchJobDetail(jobId.value)
}

function startAutoRefresh() {
  if (job.value?.status === 'running') {
    refreshInterval = window.setInterval(async () => {
      const status = await jobStore.checkJobStatus(jobId.value)
      if (status.status !== 'running') {
        clearInterval(refreshInterval!)
        refreshInterval = null
        await fetchJobDetail()
      }
    }, 2000)
  }
}

function getStatusClass(status: string) {
  const map: Record<string, string> = {
    pending: 'status-pending',
    running: 'status-running',
    completed: 'status-success',
    failed: 'status-failed'
  }
  return map[status] || ''
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`
  return `${(ms / 60000).toFixed(2)}min`
}
</script>

<template>
  <div class="job-detail">
    <div v-if="isLoading && !job" class="loading">
      <div class="loading-spinner"></div>
      <span>加载中...</span>
    </div>

    <div v-else-if="job" class="content">
      <div class="header">
        <button class="secondary" @click="$router.back()">
          <span>←</span> 返回
        </button>
        <h2>任务详情</h2>
        <button @click="fetchJobDetail" :disabled="isLoading" class="btn-primary">
          <span v-if="!isLoading">🔄</span>
          <span v-else class="refreshing">⏳</span>
          {{ isLoading ? '刷新中...' : '刷新' }}
        </button>
      </div>

      <div class="status-bar">
        <div class="status-info">
          <span :class="['status-badge', getStatusClass(job.status)]">{{ job.status }}</span>
          <span v-if="job.status === 'running'" class="pulse">运行中...</span>
        </div>
        <div class="time-info">
          <span v-if="job.startedAt">
            开始: {{ new Date(job.startedAt).toLocaleString() }}
          </span>
          <span v-if="job.finishedAt">
            完成: {{ new Date(job.finishedAt).toLocaleString() }}
          </span>
        </div>
      </div>

      <div class="grid">
        <div class="section card">
          <h3>基本信息</h3>
          <div class="info-grid">
            <div>
              <label>任务 ID</label>
              <code>{{ job.jobId }}</code>
            </div>
            <div>
              <label>算法名称</label>
              <span>{{ job.algorithmName }}</span>
            </div>
            <div>
              <label>执行模式</label>
              <span>{{ job.mode }}</span>
            </div>
            <div>
              <label>Worker ID</label>
              <code>{{ job.workerId }}</code>
            </div>
            <div>
              <label>创建时间</label>
              <span>{{ new Date(job.createdAt).toLocaleString() }}</span>
            </div>
            <div>
              <label>耗时</label>
              <span>{{ formatDuration(job.costTimeMs) }}</span>
            </div>
          </div>
        </div>

        <div class="section card">
          <h3>输入参数</h3>
          <div class="code-block">
            <pre>{{ job.inputParams }}</pre>
          </div>
        </div>

        <div class="section card">
          <h3>资源配置</h3>
          <div class="resource-info">
            <span v-if="job.inputUrl">输入: <a :href="job.inputUrl" target="_blank">{{ job.inputUrl }}</a></span>
            <span v-if="job.outputUrl">输出: <a :href="job.outputUrl" target="_blank">{{ job.outputUrl }}</a></span>
            <span v-if="job.logUrl">日志: <a :href="job.logUrl" target="_blank">{{ job.logUrl }}</a></span>
          </div>
        </div>

        <div class="section card">
          <h3>日志</h3>
          <button @click="showLogs = !showLogs" size="small">
            {{ showLogs ? '隐藏' : '显示' }} 日志
          </button>
          <div v-if="showLogs && job.logUrl" class="code-block">
            <div v-if="isLoading" class="loading">
              <div class="loading-spinner"></div>
              <span>加载日志中...</span>
            </div>
            <pre v-else>日志 URL: {{ job.logUrl }}</pre>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="error">
      <span class="error-icon">⚠️</span>
      <span class="error-text">任务不存在</span>
    </div>
  </div>
</template>

<style scoped>
.job-detail {
  min-height: 100vh;
  padding: var(--space-xl);
  background: var(--bg-secondary);
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-xl);
}

.header h2 {
  font-size: var(--font-size-2xl);
  font-weight: 600;
  color: var(--text-primary);
}

.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  padding: var(--space-md);
  margin-bottom: var(--space-xl);
}

.status-info {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}

.time-info {
  display: flex;
  gap: var(--space-lg);
  color: var(--text-secondary);
  font-size: 13px;
}

.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-lg);
}

.section {
  margin-bottom: var(--space-lg);
}

.section h3 {
  font-size: var(--font-size-lg);
  margin-bottom: var(--space-md);
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-md);
}

.info-grid label {
  display: block;
  font-size: 11px;
  color: var(--text-secondary);
  text-transform: uppercase;
  margin-bottom: 4px;
}

.info-grid span {
  font-size: 14px;
  color: var(--text-primary);
}

.info-grid code {
  font-size: 13px;
  color: var(--text-secondary);
  background: var(--bg-secondary);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}

.code-block {
  background: var(--bg-secondary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-sm);
  padding: var(--space-md);
  margin-top: var(--space-md);
  overflow-x: auto;
}

.code-block pre {
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}

.resource-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  font-size: 13px;
}

.resource-info span {
  word-break: break-all;
}

.resource-info a {
  color: var(--accent-primary);
}

.resource-info a:hover {
  color: var(--accent-hover);
  text-decoration: underline;
}

.error {
  text-align: center;
  padding: var(--space-2xl);
  color: var(--text-muted);
}

.error-icon {
  font-size: 48px;
  margin-bottom: var(--space-md);
  display: block;
}

.error-text {
  font-size: var(--font-size-lg);
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-md);
  padding: var(--space-2xl);
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border-default);
  border-top-color: var(--accent-primary);
  border-radius: var(--radius-circle);
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.pulse {
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
