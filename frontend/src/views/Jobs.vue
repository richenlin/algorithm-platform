<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useJobStore } from '../stores/job'
import { RouterLink } from 'vue-router'

const jobStore = useJobStore()
const filters = ref({
  algorithm_id: '',
  status: ''
})
const autoRefresh = ref(false)
let refreshInterval: number | null = null

const statusOptions = [
  { value: '', label: '全部' },
  { value: 'pending', label: '等待中' },
  { value: 'running', label: '运行中' },
  { value: 'completed', label: '已完成' },
  { value: 'failed', label: '失败' }
]

onMounted(() => {
  fetchJobs()
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})

function fetchJobs() {
  jobStore.fetchJobs({
    algorithm_id: filters.value.algorithm_id || undefined,
    status: filters.value.status || undefined
  })
}

function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) {
    refreshInterval = window.setInterval(fetchJobs, 5000)
  } else if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
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
</script>

<template>
  <div class="jobs">
    <div class="header-actions">
      <h2>任务列表</h2>
      <div class="actions">
        <button :class="{ 'refresh-btn': true, active: autoRefresh }" @click="toggleAutoRefresh">
          <span>{{ autoRefresh ? '⏸' : '↻' }}</span>
          {{ autoRefresh ? '停止刷新' : '自动刷新' }}
        </button>
      </div>
    </div>

    <div class="filters">
      <input v-model="filters.algorithm_id" placeholder="算法 ID" @input="fetchJobs" />
      <select v-model="filters.status" @change="fetchJobs">
        <option v-for="option in statusOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </div>

    <div v-if="jobStore.loading" class="loading">
      <div class="loading-spinner"></div>
      <span>加载中...</span>
    </div>

    <div v-else class="table-wrapper">
      <table class="table">
        <thead>
          <tr>
            <th>任务 ID</th>
            <th>算法</th>
            <th>状态</th>
            <th>创建时间</th>
            <th>耗时</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="job in jobStore.jobs" :key="job.jobId">
            <td><code>{{ job.jobId }}</code></td>
            <td>{{ job.algorithmName }}</td>
            <td>
              <span :class="['status-badge', getStatusClass(job.status)]">{{ job.status }}</span>
            </td>
            <td>{{ new Date(job.createdAt).toLocaleString() }}</td>
            <td>{{ job.costTimeMs }}ms</td>
            <td>
              <RouterLink :to="`/jobs/${job.jobId}`" class="action-link">
                查看详情 <span class="arrow">→</span>
              </RouterLink>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="jobStore.jobs.length === 0" class="empty">
        <span class="empty-icon">📭</span>
        <span class="empty-text">暂无任务</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.jobs {
  min-height: 100vh;
  padding: var(--space-xl);
  background: var(--bg-secondary);
}

.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-xl);
}

.header-actions h2 {
  font-size: var(--font-size-2xl);
  font-weight: 600;
  color: var(--text-primary);
}

.actions {
  display: flex;
  gap: var(--space-md);
}

.refresh-btn {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-sm) var(--space-md);
  background: var(--bg-card);
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
}

.refresh-btn.active {
  background: var(--accent-primary);
  color: #ffffff;
  border-color: var(--accent-primary);
}

.refresh-btn:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}

.filters {
  display: flex;
  gap: var(--space-md);
  margin-bottom: var(--space-lg);
  background: var(--bg-card);
  padding: var(--space-md);
  border-radius: var(--radius-md);
}

.filters input {
  width: 200px;
  border: 1px solid var(--border-default);
  padding: var(--space-sm) var(--space-md);
}

.filters select {
  width: 150px;
  border: 1px solid var(--border-default);
  padding: var(--space-sm) var(--space-md);
}

.table-wrapper {
  background: var(--bg-card);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}

.action-link {
  color: var(--accent-primary);
  font-weight: 500;
  transition: all var(--transition-fast);
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
}

.action-link:hover {
  color: var(--accent-hover);
}

.arrow {
  transition: transform var(--transition-fast);
  display: inline-block;
}

.action-link:hover .arrow {
  transform: translateX(4px);
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

.empty {
  text-align: center;
  padding: var(--space-2xl);
  color: var(--text-muted);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: var(--space-md);
  display: block;
}

.empty-text {
  font-size: var(--font-size-base);
}
</style>
