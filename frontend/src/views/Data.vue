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
      <button @click="showUploadModal = true">上传数据</button>
    </div>

    <div class="filters">
      <select v-model="filters.category" @change="fetchFiles">
        <option v-for="option in categoryOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </div>

    <div v-if="dataStore.loading" class="loading">加载中...</div>

    <div v-else class="table-container">
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
            <td>{{ new Date(file.created_at).toLocaleString() }}</td>
            <td>
              <a :href="file.minio_url" target="_blank" download>下载</a>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="dataStore.files.length === 0" class="empty">暂无数据</div>
    </div>

    <div v-if="showUploadModal" class="modal">
      <div class="modal-content card">
        <h3>上传数据</h3>
        <form @submit.prevent="handleUpload">
          <div class="form-group">
            <label>文件名</label>
            <input v-model="uploadFormData.filename" required />
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
            <input v-model="uploadFormData.minio_path" required />
            <small class="hint">例如: data/input/example.csv</small>
          </div>
          <div class="modal-actions">
            <button type="button" @click="showUploadModal = false">取消</button>
            <button type="submit">上传</button>
          </div>
        </form>
      </div>
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
  margin-bottom: calc(var(--grid) * 3);
}

.filters select {
  width: 200px;
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

.url-link {
  max-width: 300px;
  display: inline-block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}

.empty {
  text-align: center;
  padding: calc(var(--grid) * 4);
  color: var(--text-secondary);
}

.modal-content {
  max-width: 500px;
  width: 90%;
}

.form-group label {
  display: block;
  margin-bottom: calc(var(--grid));
  font-size: 12px;
  color: var(--text-secondary);
  text-transform: uppercase;
}

.hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.modal-actions {
  display: flex;
  gap: calc(var(--grid) * 2);
  margin-top: calc(var(--grid) * 3);
}
</style>
