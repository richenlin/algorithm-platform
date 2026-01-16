<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useDataStore } from '../stores/data'

const dataStore = useDataStore()
const filters = ref({
  category: ''
})
const showUploadModal = ref(false)
const uploadFormData = ref({
  filename: '',
  category: 'general',
  minio_path: ''
})

const categoryOptions = [
  { value: '', label: '全部' },
  { value: 'general', label: '通用' },
  { value: 'input', label: '输入数据' },
  { value: 'output', label: '输出数据' },
  { value: 'reference', label: '参考数据' }
]

onMounted(() => {
  dataStore.fetchFiles()
})

function fetchFiles() {
  dataStore.fetchFiles({
    category: filters.value.category || undefined
  })
}

async function handleUpload() {
  try {
    await dataStore.uploadFile(uploadFormData.value)
    showUploadModal.value = false
    uploadFormData.value = {
      filename: '',
      category: 'general',
      minio_path: ''
    }
    fetchFiles()
  } catch (error) {
    console.error('Failed to upload file:', error)
  }
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
              <a :href="file.minioUrl" target="_blank" class="url-link">{{ file.minioUrl }}</a>
            </td>
            <td>{{ new Date(file.createdAt).toLocaleString() }}</td>
            <td>
              <a :href="file.minioUrl" target="_blank" download class="action-link">
                下载 <span class="arrow">→</span>
              </a>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="dataStore.files.length === 0" class="empty">
        <span class="empty-icon">📁</span>
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
            <label>文件名 <span class="required">*</span></label>
            <input v-model="uploadFormData.filename" placeholder="例如: data.csv" required />
          </div>
          <div class="form-item">
            <label>类别</label>
            <select v-model="uploadFormData.category">
              <option value="general">通用</option>
              <option value="input">输入数据</option>
              <option value="output">输出数据</option>
              <option value="reference">参考数据</option>
            </select>
          </div>
          <div class="form-item">
            <label>MinIO 路径 <span class="required">*</span></label>
            <input v-model="uploadFormData.minio_path" placeholder="例如: data/input/example.csv" required />
            <span class="hint">例如: data/input/example.csv</span>
          </div>
          <div class="modal-footer">
            <button class="secondary" @click="showUploadModal = false">取消</button>
             <button class="btn-primary">上传</button>
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

.modal-content {
  position: relative;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  max-width: 500px;
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

.hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-muted);
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
</style>
