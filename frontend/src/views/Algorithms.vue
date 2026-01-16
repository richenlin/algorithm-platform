<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useAlgorithmStore } from '../stores/algorithm'
import { useDataStore } from '../stores/data'
import { serverApi } from '../api/types'

const algorithmStore = useAlgorithmStore()
const dataStore = useDataStore()
const showCreateModal = ref(false)
const submitting = ref(false)
const formData = ref({
  name: '',
  description: '',
  language: 'python',
  platform: 1,
  entrypoint: 'main.py',
  tags: [] as string[],
  presetDataId: ''
})

const newTag = ref('')
const selectedFile = ref<File | undefined>(undefined)

const platformOptions = [
  { value: 1, label: 'Linux x86_64' },
  { value: 2, label: 'Linux ARM64' },
  { value: 3, label: 'Windows x86_64' },
  { value: 4, label: 'macOS ARM64' }
]

const presetTags = [
  '通用',
  '机器学习',
  '计算机视觉',
  '自然语言处理',
  '优化'
]

const presetDataList = computed(() => dataStore.files)

const availableTags = computed(() => {
  const allTags = algorithmStore.algorithms.flatMap(alg => alg.tags || [])
  return [...new Set(allTags)]
})

function getPlatformName(value: number): string {
  const platform = platformOptions.find(p => p.value === value)
  return platform ? platform.label : 'Unknown'
}

onMounted(async () => {
  await algorithmStore.fetchAlgorithms()
  await dataStore.fetchFiles()
  
  try {
    const response = await serverApi.info()
    formData.value.platform = response.data.platform
  } catch (error) {
    console.error('Failed to fetch server info:', error)
    formData.value.platform = 1
  }
})

function addTag() {
  const tag = newTag.value.trim()
  if (tag && !formData.value.tags.includes(tag)) {
    formData.value.tags.push(tag)
  }
  newTag.value = ''
}

function addPresetTag(tag: string) {
  if (!formData.value.tags.includes(tag)) {
    formData.value.tags.push(tag)
  }
}

function removeTag(tag: string) {
  const index = formData.value.tags.indexOf(tag)
  if (index !== -1) {
    formData.value.tags.splice(index, 1)
  }
}

function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    selectedFile.value = target.files[0]
  }
}

async function handleSubmit() {
  if (submitting.value) return
  
  submitting.value = true
  try {
    let fileDataBase64 = ''
    let fileName = ''
    
    if (selectedFile.value) {
      const arrayBuffer = await selectedFile.value.arrayBuffer()
      const uint8Array = new Uint8Array(arrayBuffer)
      const binaryString = Array.from(uint8Array, byte => String.fromCharCode(byte)).join('')
      fileDataBase64 = btoa(binaryString)
      fileName = selectedFile.value.name
    }
    
    const algorithmData = {
      name: formData.value.name,
      description: formData.value.description || '',
      language: formData.value.language,
      platform: formData.value.platform,
      entrypoint: formData.value.entrypoint,
      tags: formData.value.tags,
      preset_data_id: formData.value.presetDataId || '',
      file_name: fileName,
      file_data: fileDataBase64
    }
    
    const response = await fetch('http://localhost:8080/api/v1/algorithms', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(algorithmData)
    })
    
    if (!response.ok) {
      const errorText = await response.text()
      throw new Error(`HTTP error! status: ${response.status}, ${errorText}`)
    }
    
    await algorithmStore.fetchAlgorithms()
    showCreateModal.value = false
    resetForm()
  } catch (error) {
    console.error('Failed to create algorithm:', error)
    alert('创建算法失败，请重试')
  } finally {
    submitting.value = false
  }
}

function resetForm() {
  formData.value = {
    name: '',
    description: '',
    language: 'python',
    platform: 1,
    entrypoint: 'main.py',
    tags: [],
    presetDataId: ''
  }
  selectedFile.value = undefined
  newTag.value = ''
}
</script>

<template>
  <div class="algorithms">
    <div class="header-actions">
      <h2>算法列表</h2>
      <button class="btn-primary" @click="showCreateModal = true">
        <span>+</span> 创建算法
      </button>
    </div>

    <div v-if="algorithmStore.loading" class="loading">
      <div class="loading-spinner"></div>
      <span>加载中...</span>
    </div>

    <div v-else class="table-wrapper">
      <table class="table">
        <thead>
          <tr>
            <th>算法 ID</th>
            <th>名称</th>
            <th>语言</th>
            <th>平台</th>
            <th>标签</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="algorithm in algorithmStore.algorithms" :key="algorithm.id">
            <td><code>{{ algorithm.id }}</code></td>
            <td>
              <div class="name-cell">
                <strong>{{ algorithm.name }}</strong>
                <span class="description">{{ algorithm.description || '-' }}</span>
              </div>
            </td>
            <td>{{ algorithm.language }}</td>
            <td>{{ getPlatformName(algorithm.platform) }}</td>
            <td>
              <div class="tags-cell">
                <span v-for="tag in algorithm.tags" :key="tag" class="tag">
                  {{ tag }}
                </span>
                <span v-if="!algorithm.tags || algorithm.tags.length === 0" class="empty-tags">-</span>
              </div>
            </td>
            <td>{{ new Date(algorithm.createdAt).toLocaleString() }}</td>
            <td>
              <RouterLink :to="`/algorithms/${algorithm.id}`" class="action-link">
                查看详情 <span class="arrow">→</span>
              </RouterLink>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="algorithmStore.algorithms.length === 0" class="empty">
        <span class="empty-icon">📦</span>
        <span class="empty-text">暂无算法</span>
      </div>
    </div>

    <div v-if="showCreateModal" class="modal">
      <div class="modal-backdrop" @click="showCreateModal = false"></div>
      <div class="modal-content">
        <div class="modal-header">
          <h3>创建算法</h3>
          <button class="close-btn" @click="showCreateModal = false">×</button>
        </div>
        <form @submit.prevent="handleSubmit">
          <div class="form-item">
            <label>算法名称 <span class="required">*</span></label>
            <input v-model="formData.name" placeholder="输入算法名称" required />
          </div>
          
          <div class="form-row">
            <div class="form-item">
              <label>运行平台 <span class="required">*</span></label>
              <select v-model="formData.platform" required>
                <option v-for="opt in platformOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </div>
            <div class="form-item">
              <label>编程语言 <span class="required">*</span></label>
              <select v-model="formData.language" required>
                <option value="python">Python</option>
                <option value="matlab">Matlab</option>
                <option value="cpp">C++</option>
                <option value="java">Java</option>
              </select>
            </div>
          </div>

          <div class="form-item">
            <label>入口文件 <span class="required">*</span></label>
            <input v-model="formData.entrypoint" placeholder="例如: main.py" required />
          </div>

          <div class="form-item">
            <label>算法文件 <span class="required">*</span></label>
            <div class="file-upload">
              <input
                type="file"
                @change="handleFileChange"
                accept=".zip"
                id="algorithm-file"
                required
              />
              <label for="algorithm-file" class="upload-btn">
                <span class="icon">📁</span>
                <span class="text">选择文件</span>
              </label>
              <span v-if="selectedFile" class="file-name">{{ selectedFile.name }}</span>
            </div>
            <span class="hint">提交后自动上传到 MinIO 并创建第一个版本</span>
          </div>

          <div class="form-item">
            <label>算法描述</label>
            <textarea v-model="formData.description" rows="3" placeholder="描述算法用途（可选）"></textarea>
          </div>

          <div class="form-item">
            <label>算法标签</label>
            <div class="tags-section">
              <div class="preset-tags">
                <span class="section-label">预置标签：</span>
                <span v-for="tag in presetTags" :key="tag" class="preset-tag" @click="addPresetTag(tag)">
                  {{ tag }}
                </span>
              </div>
              <div class="tags-input">
                <div class="tags-list">
                  <span v-for="tag in formData.tags" :key="tag" class="tag">
                    {{ tag }}
                    <span @click="removeTag(tag)" class="remove-tag">×</span>
                  </span>
                </div>
                <input
                  v-model="newTag"
                  @keyup.enter="addTag"
                  @keyup.comma="addTag"
                  placeholder="输入新标签（回车添加）..."
                />
              </div>
            </div>
          </div>

          <div class="form-item">
            <label>选择预置数据（作为计算依据）</label>
            <select v-model="formData.presetDataId">
              <option value="">不选择预置数据</option>
              <option v-for="data in presetDataList" :key="data.id" :value="data.id">
                {{ data.filename }} - {{ data.category }}
              </option>
            </select>
          </div>

          <div class="modal-footer">
            <button type="button" class="secondary" @click="showCreateModal = false" :disabled="submitting">
              取消
            </button>
            <button type="submit" class="btn-primary" :disabled="submitting">
              {{ submitting ? '创建中...' : '创建算法' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.algorithms {
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

button.btn-primary {
  background: var(--accent-primary);
  color: #ffffff;
  border: none;
}

button.btn-primary:hover:not(:disabled) {
  background: var(--accent-hover);
}

button.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.table-wrapper {
  background: var(--bg-card);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}

.name-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.name-cell strong {
  color: var(--text-primary);
  font-weight: 600;
}

.name-cell .description {
  font-size: 12px;
  color: var(--text-muted);
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

.modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: var(--space-lg);
  background: rgba(0, 0, 0, 0.5);
}

.modal-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
}

.modal-content {
  position: relative;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  max-width: 600px;
  width: 100%;
  max-height: 90vh;
  overflow-y: auto;
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

button.secondary:hover:not(:disabled) {
  color: var(--text-primary);
  border-color: var(--accent-primary);
}

button.secondary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.tags-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.preset-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-sm);
  align-items: center;
  padding: var(--space-sm);
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
}

.section-label {
  font-size: 12px;
  color: var(--text-muted);
  font-weight: 500;
}

.preset-tag {
  padding: 4px 10px;
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.preset-tag:hover {
  background: var(--accent-primary);
  color: white;
  border-color: var(--accent-primary);
  transform: translateY(-1px);
}

.tags-input {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.tags-list {
  display: flex;
  gap: var(--space-sm);
  flex-wrap: wrap;
}

.tags-list .tag {
  background: rgba(64, 158, 255, 0.15);
  border: 1px solid rgba(64, 158, 255, 0.3);
}

.remove-tag {
  cursor: pointer;
  margin-left: 4px;
  color: var(--text-muted);
  transition: color var(--transition-fast);
}

.remove-tag:hover {
  color: var(--danger);
}

.hint {
  display: block;
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 4px;
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
</style>
