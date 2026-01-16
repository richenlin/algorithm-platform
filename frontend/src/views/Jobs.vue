<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useJobStore } from '../stores/job'

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
      <button :class="{ active: autoRefresh }" @click="toggleAutoRefresh">
        {{ autoRefresh ? '停止自动刷新' : '自动刷新' }}
      </button>
    </div>

    <div class="filters">
      <input v-model="filters.algorithm_id" placeholder="算法 ID" @input="fetchJobs" />
      <select v-model="filters.status" @change="fetchJobs">
        <option v-for="option in statusOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </div>

    <div v-if="jobStore.loading" class="loading">加载中...</div>

    <div v-else class="table-container">
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
          <tr v-for="job in jobStore.jobs" :key="job.job_id">
            <td><code>{{ job.job_id }}</code></td>
            <td>{{ job.algorithm_name }}</td>
            <td>
              <span :class="['status-badge', getStatusClass(job.status)]">{{ job.status }}</span>
            </td>
            <td>{{ new Date(job.created_at).toLocaleString() }}</td>
            <td>{{ job.cost_time_ms }}ms</td>
            <td>
              <RouterLink :to="`/jobs/${job.job_id}`">查看详情</RouterLink>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="jobStore.jobs.length === 0" class="empty">暂无任务</div>
    </div>
  </div>
</template>

<style scoped>
.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: calc(var(--grid) * 3);
}

.filters {
  display: flex;
  gap: calc(var(--grid) * 2);
  margin-bottom: calc(var(--grid) * 3);
}

.filters input {
  width: 200px;
}

.filters select {
  width: 150px;
}

button.active {
  background-color: var(--accent);
  color: var(--bg-primary);
  border-color: var(--accent);
}

.table-container {
  background-color: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 2px;
  overflow: hidden;
}

code {
  font-size: 12px;
  color: var(--text-secondary);
}

.empty {
  text-align: center;
  padding: calc(var(--grid) * 4);
  color: var(--text-secondary);
}
</style>
