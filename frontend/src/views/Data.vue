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
    <div class="header-actions fade-in-up">
      <h2>数据管理</h2>
      <button @click="showUploadModal = true">+ 上传数据</button>
    </div>

    <div class="filters fade-in-up">
      <select v-model="filters.category" @change="fetchFiles">
        <option v-for="option in categoryOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </div>

    <div v-if="dataStore.loading" class="loading fade-in-up">加载中...</div>

    <div v-else class="table-container fade-in-up">
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
              <a :href="file.minioUrl" target="_blank" download class="action-link">下载 →</a>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="dataStore.files.length === 0" class="empty">暂无数据</div>
    </div>

    <div v-if="showUploadModal" class="modal fade-in-up">
      <div class="modal-backdrop" @click="showUploadModal = false"></div>
      <div class="modal-content card">
        <h3>上传数据</h3>
        <form @submit.prevent="handleUpload">
          <div class="form-group">
            <label>文件名</label>
            <input v-model="uploadFormData.filename" placeholder="例如: data.csv" required />
          </div>
          <div class="form-group">
            <label>类别</label>
            <select v-model="uploadFormData.category">
              <option value="general">通用</option>
              <option value="input">输入数据</option>
              <option value="output">输出数据</option>
              <option value="reference">参考数据</option>
            </select>
          </div>
          <div class="form-group">
            <label>MinIO 路径</label>
            <input v-model="uploadFormData.minio_path" placeholder="例如: data/input/example.csv" required />
            <small class="hint">例如: data/input/example.csv</small>
          </div>
          <div class="modal-actions">
            <button type="button" @click="showUploadModal = false" class="secondary">取消</button>
            <button type="submit">上传</button>
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

.filters {
  margin-bottom: var(--space-lg);
}

.filters select {
  width: 200px;
}

.table-container {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
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
  color: var(--text-secondary);
}

.url-link:hover {
  color: var(--accent-primary);
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

.modal-content {
  position: relative;
  max-width: 500px;
  width: 100%;
}

.form-group {
  margin-bottom: var(--space-md);
}

.form-group label {
  display: block;
  margin-bottom: var(--space-sm);
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-muted);
}

.modal-actions {
  display: flex;
  gap: var(--space-md);
  margin-top: var(--space-xl);
  padding-top: var(--space-lg);
  border-top: 1px solid var(--border-subtle);
}

button.secondary {
  background: transparent;
  border: 1px solid var(--border-default);
}

button.secondary:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border-color: var(--border-default);
  box-shadow: none;
}
</style>
