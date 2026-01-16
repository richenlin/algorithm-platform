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
    <div v-if="isLoading && !job" class="loading">加载中...</div>

    <div v-else-if="job" class="content">
      <div class="header">
        <button @click="$router.back()">返回</button>
        <h2>任务详情</h2>
        <button @click="fetchJobDetail" :disabled="isLoading">
          {{ isLoading ? '刷新中...' : '刷新' }}
        </button>
      </div>

      <div class="status-bar">
        <div class="status-info">
          <span class="status-badge" :class="getStatusClass(job.status)">{{ job.status }}</span>
          <span v-if="job.status === 'running'" class="pulse">运行中...</span>
        </div>
        <div class="time-info">
          <span v-if="job.started_at">
            开始: {{ new Date(job.started_at).toLocaleString() }}
          </span>
          <span v-if="job.finished_at">
            完成: {{ new Date(job.finished_at).toLocaleString() }}
          </span>
        </div>
      </div>

      <div class="grid">
        <div class="section card">
          <h3>基本信息</h3>
          <div class="info-grid">
            <div>
              <label>任务 ID</label>
              <code>{{ job.job_id }}</code>
            </div>
            <div>
              <label>算法名称</label>
              <span>{{ job.algorithm_name }}</span>
            </div>
            <div>
              <label>执行模式</label>
              <span>{{ job.mode }}</span>
            </div>
            <div>
              <label>Worker ID</label>
              <code>{{ job.worker_id }}</code>
            </div>
            <div>
              <label>创建时间</label>
              <span>{{ new Date(job.created_at).toLocaleString() }}</span>
            </div>
            <div>
              <label>耗时</label>
              <span>{{ formatDuration(job.cost_time_ms) }}</span>
            </div>
          </div>
        </div>

        <div class="section card">
          <h3>输入参数</h3>
          <div class="code-block">
            <pre>{{ job.input_params }}</pre>
          </div>
        </div>

        <div class="section card">
          <h3>资源配置</h3>
          <div class="resource-info">
            <span v-if="job.input_url">输入: <a :href="job.input_url" target="_blank">{{ job.input_url }}</a></span>
            <span v-if="job.output_url">输出: <a :href="job.output_url" target="_blank">{{ job.output_url }}</a></span>
            <span v-if="job.log_url">日志: <a :href="job.log_url" target="_blank">{{ job.log_url }}</a></span>
          </div>
        </div>

        <div class="section card">
          <h3>日志</h3>
          <button @click="showLogs = !showLogs" size="small">
            {{ showLogs ? '隐藏' : '显示' }} 日志
          </button>
          <div v-if="showLogs && job.log_url" class="code-block">
            <div v-if="isLoading" class="loading">加载日志中...</div>
            <pre v-else>日志 URL: {{ job.log_url }}</pre>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="error">
      任务不存在
    </div>
  </div>
</template>

<style scoped>
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: calc(var(--grid) * 3);
}

.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 2px;
  padding: calc(var(--grid) * 2);
  margin-bottom: calc(var(--grid) * 3);
}

.status-info {
  display: flex;
  align-items: center;
  gap: calc(var(--grid) * 2);
}

.time-info {
  display: flex;
  gap: calc(var(--grid) * 2);
  color: var(--text-secondary);
  font-size: 13px;
}

.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: calc(var(--grid) * 3);
}

.section {
  margin-bottom: calc(var(--grid) * 3);
}

.section h3 {
  margin-bottom: calc(var(--grid) * 2);
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: calc(var(--grid) * 2);
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
}

.info-grid code {
  font-size: 13px;
  color: var(--text-secondary);
}

.code-block {
  background-color: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 2px;
  padding: calc(var(--grid) * 2);
  margin-top: calc(var(--grid) * 2);
  overflow-x: auto;
}

.code-block pre {
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-all;
}

.resource-info {
  display: flex;
  flex-direction: column;
  gap: calc(var(--grid));
  font-size: 13px;
}

.resource-info span {
  word-break: break-all;
}

.error {
  text-align: center;
  padding: calc(var(--grid) * 4);
  color: var(--error);
}
</style>
