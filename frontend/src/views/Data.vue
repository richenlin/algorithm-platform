<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useDataStore } from '../stores/data'

const dataStore = useDataStore()
const filters = ref({
  category: ''
})
const showUploadModal = ref(false)
const uploading = ref(false)
const uploadFormData = ref({
  filename: '',
  category: '通用',
  tags: [] as string[]
})

const selectedFile = ref<File | undefined>(undefined)

const categoryOptions = [
  { value: '', label: '全部' },
  { value: '通用', label: '通用' },
  { value: '输入数据', label: '输入数据' },
  { value: '输出数据', label: '输出数据' },
  { value: '参考数据', label: '参考数据' }
]

const presetCategories = ['通用', '输入数据', '输出数据', '参考数据']

const availableCategories = computed(() => {
  const allCategories = dataStore.files.map(f => f.category)
  return [...new Set(allCategories)]
})

const newCategory = ref('')

onMounted(() => {
  dataStore.fetchFiles()
})

function fetchFiles() {
  dataStore.fetchFiles({
    category: filters.value.category || undefined
  })
}

function addCategory() {
  const category = newCategory.value.trim()
  if (category && uploadFormData.value.category !== category) {
    uploadFormData.value.category = category
  }
  newCategory.value = ''
}

async function handleUpload() {
  if (uploading.value || !selectedFile.value) {
    alert('请选择文件')
    return
  }
  
  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)
    formData.append('filename', uploadFormData.value.filename || selectedFile.value.name)
    formData.append('category', uploadFormData.value.category)
    
    const response = await fetch('http://localhost:8080/api/v1/data/upload-multipart', {
      method: 'POST',
      body: formData
    })
    
    if (!response.ok) {
      const errorText = await response.text()
      throw new Error(`HTTP error! status: ${response.status}, ${errorText}`)
    }
    
    const result = await response.json()
    
    showUploadModal.value = false
    resetForm()
    fetchFiles()
  } catch (error) {
    console.error('Failed to upload file:', error)
    alert('上传失败，请重试')
  } finally {
    uploading.value = false
  }
}

function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    selectedFile.value = target.files[0]
    if (!uploadFormData.value.filename) {
      uploadFormData.value.filename = target.files[0].name
    }
  }
}

function resetForm() {
  uploadFormData.value = {
    filename: '',
    category: '通用',
    tags: []
  }
  selectedFile.value = undefined
  newCategory.value = ''
}
</script>

<template>
  <div class="data">
    <div class="header-actions">
      <h2>数据管理</h2>
      <button class="btn-primary" @click="showUploadModal = true">
        <span>+</span> 上传数据
      </button>
    </div>

    <div class="filters">
      <select v-model="filters.category" @change="fetchFiles">
        <option v-for="option in categoryOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </div>

    <div v-if="dataStore.loading" class="loading">
      <div class="loading-spinner"></div>
      <span>加载中...</span>
    </div>

    <div v-else class="table-wrapper">
      <table class="table">
        <thead>
          <tr>
            <th>文件 ID</th>
            <th>文件名</th>
            <th>类别</th>
            <th>MinIO URL</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="file in dataStore.files" :key="file.id">
            <td><code>{{ file.id }}</code></td>
            <td>{{ file.filename }}</td>
            <td>{{ file.category }}</td>
            <td>
              <a :href="file.minio_url" target="_blank" class="url-link">{{ file.minio_url }}</a>
            </td>
            <td>{{ file.created_at ? new Date(file.created_at).toLocaleString() : '-' }}</td>
            <td>
              <a :href="file.minio_url" target="_blank" download class="action-link">
                下载 <span class="arrow">→</span>
              </a>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="dataStore.files.length === 0" class="empty">
        <span class="empty-icon">No data</span>
        <span class="empty-text">暂无数据</span>
      </div>
    </div>

    <div v-if="showUploadModal" class="modal">
      <div class="modal-backdrop" @click="showUploadModal = false"></div>
      <div class="modal-content">
        <div class="modal-header">
          <h3>上传数据</h3>
          <button class="close-btn" @click="showUploadModal = false">×</button>
        </div>
        <form @submit.prevent="handleUpload">
          <div class="form-item">
            <label>数据名 <span class="required">*</span></label>
            <input v-model="uploadFormData.filename" placeholder="例如: 数据集1" required />
          </div>
          
          <div class="form-item">
            <label>类别 <span class="required">*</span></label>
            <div class="category-section">
              <div class="preset-categories">
                <span class="section-label">预置类别：</span>
                <span v-for="cat in presetCategories" :key="cat"
                      class="preset-category"
                      :class="{ active: uploadFormData.category === cat }"
                      @click="uploadFormData.category = cat">
                  {{ cat }}
                </span>
              </div>
              <div class="custom-category">
                <input
                  v-model="newCategory"
                  @keyup.enter="addCategory"
                  placeholder="输入新类别（回车添加）..."
                />
              </div>
            </div>
          </div>

          <div class="form-item">
            <label>选取文件 <span class="required">*</span></label>
            <div class="file-upload">
              <input
                type="file"
                @change="handleFileChange"
                accept=".txt,.csv,.json,.xml,.yaml,.yml"
                id="data-file"
                required
              />
              <label for="data-file" class="upload-btn">
                <span class="icon">File</span>
                <span class="text">选择文件</span>
              </label>
              <span v-if="selectedFile" class="file-name">{{ selectedFile.name }}</span>
            </div>
            <span class="hint">支持格式：文本、CSV、JSON、XML、YAML</span>
          </div>

          <div class="modal-footer">
            <button type="button" class="secondary" @click="showUploadModal = false" :disabled="uploading">
              取消
            </button>
            <button type="submit" class="btn-primary" :disabled="uploading">
              {{ uploading ? '上传中...' : '上传' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.data {
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

.filters {
  margin-bottom: var(--space-lg);
}

.filters select {
  width: 200px;
  padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
}

.table-wrapper {
  background: var(--bg-card);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}

.url-link {
  max-width: 300px;
  display: inline-block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
  color: var(--accent-primary);
}

.url-link:hover {
  color: var(--accent-hover);
  text-decoration: underline;
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
  max-width: 500px;
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

.category-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.preset-categories {
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

.preset-category {
  padding: 4px 10px;
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.preset-category:hover {
  background: var(--accent-primary);
  color: white;
  border-color: var(--accent-primary);
  transform: translateY(-1px);
}

.preset-category.active {
  background: var(--accent-primary);
  color: white;
  border-color: var(--accent-primary);
}

.custom-category input {
  width: 100%;
  padding: var(--space-sm);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
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

.hint {
  display: block;
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 4px;
}

[type="file"] {
  display: none;
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
</style>
