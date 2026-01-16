<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAlgorithmStore } from '../stores/algorithm'
import { useJobStore } from '../stores/job'

const route = useRoute()
const router = useRouter()
const algorithmStore = useAlgorithmStore()
const jobStore = useJobStore()

const showCreateVersion = ref(false)
const showExecuteModal = ref(false)
const creatingVersion = ref(false)
const versionFormData = ref({
  commit_message: ''
})
const selectedVersionFile = ref<File | undefined>(undefined)
const executeFormData = ref({
  mode: 'batch',
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

onUnmounted(() => {
  algorithmStore.versions = []
})

async function handleCreateVersion() {
  if (creatingVersion.value || !selectedVersionFile.value) {
    alert('请选择文件')
    return
  }
  
  creatingVersion.value = true
  try {
    const arrayBuffer = await selectedVersionFile.value.arrayBuffer()
    const uint8Array = new Uint8Array(arrayBuffer)
    const binaryString = Array.from(uint8Array, byte => String.fromCharCode(byte)).join('')
    const fileDataBase64 = btoa(binaryString)
    
    const response = await fetch(`http://localhost:8080/api/v1/algorithms/${algorithmId.value}/versions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        source_code_zip_url: '',
        commit_message: versionFormData.value.commit_message,
        file_name: selectedVersionFile.value.name,
        file_data: fileDataBase64
      })
    })
    
    if (!response.ok) {
      throw new Error('Failed to create version')
    }
    
    await algorithmStore.fetchAlgorithm(algorithmId.value)
    showCreateVersion.value = false
    versionFormData.value = {
      commit_message: ''
    }
    selectedVersionFile.value = undefined
  } catch (error) {
    console.error('Failed to create version:', error)
    alert('创建版本失败，请重试')
  } finally {
    creatingVersion.value = false
  }
}

function handleVersionFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    selectedVersionFile.value = target.files[0]
  }
}

async function handleExecute() {
  try {
    const params = JSON.parse(executeFormData.value.params) || {}
    const data = {
      ...executeFormData.value,
      params
    }
    const result = await jobStore.executeAlgorithm(algorithmId.value, data)
    showExecuteModal.value = false
    router.push(`/jobs/${result.jobId}`)
  } catch (error) {
    console.error('Failed to execute algorithm:', error)
  }
}

async function handleRollback(versionId: string) {
  if (!confirm('确认回滚到此版本？')) return
  
  try {
    const response = await fetch(`http://localhost:8080/api/v1/algorithms/${algorithmId.value}/versions/${versionId}/rollback`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      }
    })
    
    if (!response.ok) {
      throw new Error('Failed to rollback version')
    }
    
    await algorithmStore.fetchAlgorithm(algorithmId.value)
    alert('回滚成功')
  } catch (error) {
    console.error('Failed to rollback version:', error)
    alert('回滚失败，请重试')
  }
}

function getPlatformName(value: number): string {
  const platformOptions = [
    { value: 1, label: 'Linux x86_64' },
    { value: 2, label: 'Linux ARM64' },
    { value: 3, label: 'Windows x86_64' },
    { value: 4, label: 'macOS ARM64' }
  ]
  const platform = platformOptions.find(p => p.value === value)
  return platform ? platform.label : 'Unknown'
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
  <div class="algorithm-detail">
    <div v-if="algorithmStore.loading" class="loading">
      <div class="loading-spinner"></div>
      <span>加载中...</span>
    </div>

    <div v-else-if="algorithm" class="content">
      <div class="header">
        <button class="secondary" @click="router.back()">
          <span>←</span> 返回
        </button>
        <h2>{{ algorithm.name }}</h2>
        <button class="btn-primary" @click="showExecuteModal = true">
          <span>⚡</span> 执行算法
        </button>
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
              <span>{{ getPlatformName(algorithm.platform) }}</span>
            </div>
            <div>
              <label>入口文件</label>
              <span>{{ algorithm.entrypoint }}</span>
            </div>
            <div>
              <label>标签</label>
              <div class="tags-cell">
                <span v-for="tag in algorithm.tags" :key="tag" class="tag">
                  {{ tag }}
                </span>
                <span v-if="!algorithm.tags || algorithm.tags.length === 0" class="empty-tags">-</span>
              </div>
            </div>
            <div>
              <label>创建时间</label>
              <span>{{ new Date(algorithm.createdAt).toLocaleString() }}</span>
            </div>
            <div>
              <label>更新时间</label>
              <span>{{ new Date(algorithm.updatedAt).toLocaleString() }}</span>
            </div>
          </div>
          <p class="description">{{ algorithm.description }}</p>
        </div>

        <div class="section card">
          <div class="section-header">
            <h3>版本历史</h3>
            <button class="btn-primary" @click="showCreateVersion = true">
              <span>+</span> 新版本
            </button>
          </div>
          <div v-if="algorithmStore.versions.length === 0" class="empty">
            <span class="empty-icon">📦</span>
            <span class="empty-text">暂无版本</span>
          </div>
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
                <td>v{{ version.versionNumber }}</td>
                <td>{{ version.commitMessage }}</td>
                <td>{{ new Date(version.createdAt).toLocaleString() }}</td>
                <td>
                  <button v-if="version.id !== algorithm.currentVersionId" 
                          size="small" 
                          @click="handleRollback(version.id)">
                    回滚
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="section card full-width">
          <h3>最近执行记录</h3>
          <div v-if="jobStore.jobs.length === 0" class="empty">
            <span class="empty-icon">📊</span>
            <span class="empty-text">暂无执行记录</span>
          </div>
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
              <tr v-for="job in jobStore.jobs.slice(0, 10)" :key="job.jobId">
                <td><code>{{ job.jobId }}</code></td>
                <td>
                  <span :class="['status-badge', getStatusClass(job.status)]">{{ job.status }}</span>
                </td>
                <td>{{ new Date(job.createdAt).toLocaleString() }}</td>
                <td>{{ job.costTimeMs }}ms</td>
                <td>
                  <RouterLink :to="`/jobs/${job.jobId}`" class="action-link">
                    查看 <span class="arrow">→</span>
                  </RouterLink>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div v-if="showCreateVersion" class="modal">
      <div class="modal-backdrop" @click="showCreateVersion = false"></div>
      <div class="modal-content">
        <div class="modal-header">
          <h3>创建新版本</h3>
          <button class="close-btn" @click="showCreateVersion = false">×</button>
        </div>
        <form @submit.prevent="handleCreateVersion">
          <div class="form-item">
            <label>上传文件 <span class="required">*</span></label>
            <div class="file-upload">
              <input
                type="file"
                @change="handleVersionFileChange"
                accept=".zip"
                id="version-file"
                required
              />
              <label for="version-file" class="upload-btn">
                <span class="icon">📁</span>
                <span class="text">选择文件</span>
              </label>
              <span v-if="selectedVersionFile" class="file-name">{{ selectedVersionFile.name }}</span>
            </div>
          </div>
          <div class="form-item">
            <label>提交信息 <span class="required">*</span></label>
            <textarea v-model="versionFormData.commit_message" rows="3" placeholder="描述本次更新" required></textarea>
          </div>
          <div class="modal-footer">
            <button type="button" class="secondary" @click="showCreateVersion = false" :disabled="creatingVersion">
              取消
            </button>
             <button type="submit" class="btn-primary" :disabled="creatingVersion">
              {{ creatingVersion ? '创建中...' : '创建' }}
            </button>
           </div>
        </form>
      </div>
    </div>

    <div v-if="showExecuteModal" class="modal">
      <div class="modal-backdrop" @click="showExecuteModal = false"></div>
      <div class="modal-content">
        <div class="modal-header">
          <h3>执行算法</h3>
          <button class="close-btn" @click="showExecuteModal = false">×</button>
        </div>
        <form @submit.prevent="handleExecute">
          <div class="form-item">
            <label>执行模式</label>
            <select v-model="executeFormData.mode">
              <option value="batch">批处理</option>
              <option value="streaming">流式</option>
            </select>
          </div>
          <div class="form-item">
            <label>输入参数 (JSON)</label>
            <textarea v-model="executeFormData.params" rows="3" placeholder='{"key": "value"}'></textarea>
          </div>
          <div class="form-item">
            <label>输入数据 URL <span class="required">*</span></label>
            <input v-model="executeFormData.input_source.url" placeholder="https://..." required />
          </div>
          <div class="form-row">
            <div class="form-item">
              <label>CPU 限制</label>
              <input type="number" v-model.number="executeFormData.resource_config.cpu_limit" step="0.1" />
            </div>
            <div class="form-item">
              <label>内存限制</label>
              <input v-model="executeFormData.resource_config.memory_limit" placeholder="例如: 4Gi" />
            </div>
          </div>
          <div class="form-item">
            <label>超时时间 (秒)</label>
            <input type="number" v-model.number="executeFormData.timeout_seconds" />
          </div>
          <div class="modal-footer">
            <button class="secondary" @click="showExecuteModal = false">取消</button>
             <button class="btn-primary">执行</button>
           </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.algorithm-detail {
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

.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-lg);
}

.section {
  margin-bottom: var(--space-lg);
}

.full-width {
  grid-column: span 2;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-md);
}

.section h3 {
  font-size: var(--font-size-lg);
  margin-bottom: var(--space-md);
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-md);
  margin-bottom: var(--space-md);
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

.tags-cell {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.tags-cell .tag {
  padding: 2px 8px;
  background: rgba(64, 158, 255, 0.1);
  border: 1px solid rgba(64, 158, 255, 0.3);
  border-radius: 12px;
  font-size: 11px;
  color: var(--accent-primary);
}

.empty-tags {
  color: var(--text-muted);
  font-size: 12px;
}

.file-upload {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.upload-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-sm) var(--space-md);
  background: var(--bg-secondary);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition-base);
}

.upload-btn:hover {
  background: var(--accent-primary);
  border-color: var(--accent-primary);
  color: white;
}

.upload-btn .icon {
  font-size: var(--font-size-lg);
}

.file-name {
  font-size: var(--font-size-sm);
  color: var(--accent-primary);
  font-weight: 500;
}

[type="file"] {
  display: none;
}

.description {
  color: var(--text-secondary);
  line-height: 1.6;
  margin: 0;
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

.modal-content {
  position: relative;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  max-width: 600px;
  width: 100%;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-lg);
  border-bottom: 1px solid var(--border-light);
}

.modal-header h3 {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.close-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xl);
  color: var(--text-muted);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.close-btn:hover {
  color: var(--text-primary);
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
}

form {
  padding: var(--space-lg);
}

.form-item {
  margin-bottom: var(--space-md);
}

.form-item label {
  display: block;
  margin-bottom: var(--space-sm);
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-primary);
}

.required {
  color: var(--danger);
  margin-left: 2px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-lg);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-md);
  margin-top: var(--space-xl);
  padding-top: var(--space-lg);
  border-top: 1px solid var(--border-light);
}

button.secondary {
  background: #ffffff;
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
}

button.secondary:hover {
  color: var(--text-primary);
  border-color: var(--accent-primary);
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
</style>
