<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAlgorithmStore } from '../stores/algorithm'
import { useJobStore } from '../stores/job'

const route = useRoute()
const router = useRouter()
const algorithmStore = useAlgorithmStore()
const jobStore = useJobStore()

const showCreateVersion = ref(false)
const showExecuteModal = ref(false)
const versionFormData = ref({
  source_code_zip_url: '',
  commit_message: ''
})
const executeFormData = ref({
  mode: 'batch' as const,
  params: '{}',
  input_source: { type: 'minio', url: '' },
  resource_config: { cpu_limit: 2, memory_limit: '4Gi' },
  timeout_seconds: 300
})

const algorithmId = computed(() => route.params.id as string)
const algorithm = computed(() => algorithmStore.currentAlgorithm)

onMounted(async () => {
  await algorithmStore.fetchAlgorithm(algorithmId.value)
  await jobStore.fetchJobs({ algorithm_id: algorithmId.value })
})

async function handleCreateVersion() {
  try {
    await algorithmStore.createVersion(algorithmId.value, versionFormData.value)
    showCreateVersion.value = false
    versionFormData.value = {
      source_code_zip_url: '',
      commit_message: ''
    }
  } catch (error) {
    console.error('Failed to create version:', error)
  }
}

async function handleExecute() {
  try {
    const params = JSON.parse(executeFormData.value.params as any) || {}
    const data = {
      ...executeFormData.value,
      params
    }
    const result = await jobStore.executeAlgorithm(algorithmId.value, data)
    showExecuteModal.value = false
    router.push(`/jobs/${result.job_id}`)
  } catch (error) {
    console.error('Failed to execute algorithm:', error)
  }
}

async function handleRollback(versionId: string) {
  try {
    await algorithmStore.rollbackVersion(algorithmId.value, versionId)
  } catch (error) {
    console.error('Failed to rollback version:', error)
  }
}
</script>

<template>
  <div class="algorithm-detail">
    <div v-if="algorithmStore.loading" class="loading">加载中...</div>

    <div v-else-if="algorithm" class="content">
      <div class="header">
        <button @click="router.back()">返回</button>
        <h2>{{ algorithm.name }}</h2>
        <button @click="showExecuteModal = true">执行算法</button>
      </div>

      <div class="grid">
        <div class="section card">
          <h3>基本信息</h3>
          <div class="info-grid">
            <div>
              <label>语言</label>
              <span>{{ algorithm.language }}</span>
            </div>
            <div>
              <label>平台</label>
              <span>{{ algorithm.platform }}</span>
            </div>
            <div>
              <label>类别</label>
              <span>{{ algorithm.category }}</span>
            </div>
            <div>
              <label>入口文件</label>
              <span>{{ algorithm.entrypoint }}</span>
            </div>
            <div>
              <label>当前版本</label>
              <span>{{ algorithm.current_version_id }}</span>
            </div>
            <div>
              <label>创建时间</label>
              <span>{{ new Date(algorithm.created_at).toLocaleString() }}</span>
            </div>
          </div>
          <p class="description">{{ algorithm.description }}</p>
        </div>

        <div class="section card">
          <div class="section-header">
            <h3>版本历史</h3>
            <button @click="showCreateVersion = true">新版本</button>
          </div>
          <div v-if="algorithmStore.versions.length === 0" class="empty">暂无版本</div>
          <table v-else class="table">
            <thead>
              <tr>
                <th>版本号</th>
                <th>提交信息</th>
                <th>创建时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="version in algorithmStore.versions" :key="version.id">
                <td>v{{ version.version_number }}</td>
                <td>{{ version.commit_message }}</td>
                <td>{{ new Date(version.created_at).toLocaleString() }}</td>
                <td>
                  <button v-if="version.id !== algorithm.current_version_id" @click="handleRollback(version.id)" size="small">
                    回滚
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="section card full-width">
          <h3>最近执行记录</h3>
          <div v-if="jobStore.jobs.length === 0" class="empty">暂无执行记录</div>
          <table v-else class="table">
            <thead>
              <tr>
                <th>任务ID</th>
                <th>状态</th>
                <th>创建时间</th>
                <th>耗时</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="job in jobStore.jobs.slice(0, 10)" :key="job.job_id">
                <td>{{ job.job_id }}</td>
                <td>
                  <span :class="['status-badge', `status-${job.status}`]">{{ job.status }}</span>
                </td>
                <td>{{ new Date(job.created_at).toLocaleString() }}</td>
                <td>{{ job.cost_time_ms }}ms</td>
                <td>
                  <RouterLink :to="`/jobs/${job.job_id}`">查看</RouterLink>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div v-if="showCreateVersion" class="modal">
      <div class="modal-content card">
        <h3>创建新版本</h3>
        <form @submit.prevent="handleCreateVersion">
          <div class="form-group">
            <label>源代码 ZIP URL</label>
            <input v-model="versionFormData.source_code_zip_url" required />
          </div>
          <div class="form-group">
            <label>提交信息</label>
            <textarea v-model="versionFormData.commit_message" rows="3" required></textarea>
          </div>
          <div class="modal-actions">
            <button type="button" @click="showCreateVersion = false">取消</button>
            <button type="submit">创建</button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showExecuteModal" class="modal">
      <div class="modal-content card">
        <h3>执行算法</h3>
        <form @submit.prevent="handleExecute">
          <div class="form-group">
            <label>执行模式</label>
            <select v-model="executeFormData.mode">
              <option value="batch">批处理</option>
              <option value="streaming">流式</option>
            </select>
          </div>
          <div class="form-group">
            <label>输入参数 (JSON)</label>
            <textarea v-model="executeFormData.params" rows="3" placeholder='{"key": "value"}'></textarea>
          </div>
          <div class="form-group">
            <label>输入数据 URL</label>
            <input v-model="executeFormData.input_source.url" required />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label>CPU 限制</label>
              <input type="number" v-model.number="executeFormData.resource_config.cpu_limit" step="0.1" />
            </div>
            <div class="form-group">
              <label>内存限制</label>
              <input v-model="executeFormData.resource_config.memory_limit" />
            </div>
          </div>
          <div class="form-group">
            <label>超时时间 (秒)</label>
            <input type="number" v-model.number="executeFormData.timeout_seconds" />
          </div>
          <div class="modal-actions">
            <button type="button" @click="showExecuteModal = false">取消</button>
            <button type="submit">执行</button>
          </div>
        </form>
      </div>
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

.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: calc(var(--grid) * 3);
}

.section {
  margin-bottom: calc(var(--grid) * 3);
}

.full-width {
  grid-column: span 2;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: calc(var(--grid) * 2);
}

.section h3 {
  margin-bottom: calc(var(--grid) * 2);
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: calc(var(--grid) * 2);
  margin-bottom: calc(var(--grid) * 2);
}

.info-grid div label {
  display: block;
  font-size: 11px;
  color: var(--text-secondary);
  text-transform: uppercase;
  margin-bottom: 4px;
}

.info-grid div span {
  font-size: 14px;
}

.description {
  color: var(--text-secondary);
  line-height: 1.6;
}

.empty {
  text-align: center;
  padding: calc(var(--grid) * 4);
  color: var(--text-secondary);
}

.modal-content {
  max-width: 600px;
  width: 90%;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: calc(var(--grid) * 2);
}

button[size="small"] {
  padding: 4px 8px;
  font-size: 12px;
}
</style>
