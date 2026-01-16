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
    <div class="header-actions fade-in-up">
      <h2>任务列表</h2>
      <div class="actions">
        <button @click="toggleAutoRefresh" :class="{ active: autoRefresh }">
          {{ autoRefresh ? '⏸ 停止' : '↻ 自动刷新' }}
        </button>
      </div>
    </div>

    <div class="filters fade-in-up">
      <input v-model="filters.algorithm_id" placeholder="算法 ID" @input="fetchJobs" />
      <select v-model="filters.status" @change="fetchJobs">
        <option v-for="option in statusOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </div>

    <div v-if="jobStore.loading" class="loading fade-in-up">加载中...</div>

    <div v-else class="table-container fade-in-up">
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
              <RouterLink :to="`/jobs/${job.jobId}`" class="action-link">查看详情 →</RouterLink>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="jobStore.jobs.length === 0" class="empty">暂无任务</div>
    </div>
  </div>
</template>

<style scoped>
.jobs {
  min-height: 100vh;
  padding: var(--space-xl);
}

.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-xl);
}

.header-actions h2 {
  font-size: var(--font-size-2xl);
  font-weight: 700;
}

.actions {
  display: flex;
  gap: var(--space-md);
}

button.active {
  background: var(--accent-primary);
  color: white;
  border-color: var(--accent-primary);
  box-shadow: var(--shadow-glow);
}

.filters {
  display: flex;
  gap: var(--space-md);
  margin-bottom: var(--space-lg);
}

.filters input {
  width: 200px;
}

.filters select {
  width: 150px;
}

.table-container {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}

.action-link {
  color: var(--accent-primary);
  font-weight: 500;
  transition: all var(--transition-fast);
}

.action-link:hover {
  color: var(--accent-secondary);
  transform: translateX(4px);
}
</style>
