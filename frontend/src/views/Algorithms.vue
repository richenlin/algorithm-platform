<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useAlgorithmStore } from '../stores/algorithm'
import { useDataStore } from '../stores/data'

const algorithmStore = useAlgorithmStore()
const dataStore = useDataStore()
const showCreateModal = ref(false)
const formData = ref({
  name: '',
  description: '',
  language: 'python',
  platform: 'LINUX_X86_64',
  category: 'general',
  entrypoint: 'main.py',
  tags: [] as string[],
  presetDataId: ''
})

const newTag = ref('')
const selectedFile = ref<File | undefined>(undefined)

const availableTags = computed(() => {
  const allTags = algorithmStore.algorithms.flatMap(alg => alg.tags || [])
  return [...new Set(allTags)]
})

const platformOptions = [
  { value: 'docker', label: 'Docker' },
  { value: 'LINUX_X86_64', label: 'Linux x86_64' },
  { value: 'LINUX_ARM64', label: 'Linux ARM64' },
  { value: 'WINDOWS_X86_64', label: 'Windows x86_64' },
  { value: 'MACOS_ARM64', label: 'macOS ARM64' }
]

const presetDataList = computed(() => dataStore.files)

onMounted(async () => {
  await algorithmStore.fetchAlgorithms()
  await dataStore.fetchFiles()
})

function addTag() {
  const tag = newTag.value.trim()
  if (tag && !formData.value.tags.includes(tag)) {
    formData.value.tags.push(tag)
  }
  newTag.value = ''
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
  try {
    const formDataToSend = {
      ...formData.value,
      file: selectedFile.value
    }
    await algorithmStore.createAlgorithm(formDataToSend)
    showCreateModal.value = false
    resetForm()
  } catch (error) {
    console.error('Failed to create algorithm:', error)
  }
}

function resetForm() {
  formData.value = {
    name: '',
    description: '',
    language: 'python',
    platform: 'LINUX_X86_64',
    category: 'general',
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

    <div v-else class="algorithms-grid">
      <div v-for="algorithm in algorithmStore.algorithms" :key="algorithm.id" 
           class="algorithm-card">
        <div class="card-header">
          <div>
            <h3>{{ algorithm.name }}</h3>
            <div class="meta-badges">
              <span class="platform-badge">{{ algorithm.platform }}</span>
              <span class="language-tag">{{ algorithm.language }}</span>
            </div>
          </div>
        </div>
        <p class="description">{{ algorithm.description }}</p>
        <div class="meta">
          <span>平台: {{ algorithm.platform }}</span>
          <span>类别: {{ algorithm.category }}</span>
        </div>
        <RouterLink :to="`/algorithms/${algorithm.id}`" class="view-btn">
          查看详情 <span class="arrow">→</span>
        </RouterLink>
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
            <label>名称 <span class="required">*</span></label>
            <input v-model="formData.name" placeholder="输入算法名称" required />
          </div>
          <div class="form-item">
            <label>描述 <span class="required">*</span></label>
            <textarea v-model="formData.description" rows="3" placeholder="描述算法用途" required></textarea>
          </div>
          
          <div class="form-row">
            <div class="form-item">
              <label>语言</label>
              <select v-model="formData.language">
                <option value="python">Python</option>
                <option value="matlab">Matlab</option>
                <option value="cpp">C++</option>
                <option value="java">Java</option>
                <option value="r">R</option>
              </select>
            </div>
            <div class="form-item">
              <label>平台</label>
              <select v-model="formData.platform">
                <option v-for="opt in platformOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </div>
          </div>

          <div class="form-item">
            <label>入口文件 <span class="required">*</span></label>
            <input v-model="formData.entrypoint" placeholder="例如: main.py" required />
          </div>

          <div class="form-item">
            <label>算法标签（可新增，回车或逗号分隔）</label>
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
                placeholder="输入标签..."
              />
              />
            </div>
            <div v-if="availableTags.length > 0" class="available-tags">
              <span class="hint">已存在的标签：</span>
              <span v-for="tag in availableTags" :key="tag" class="suggested-tag" @click="formData.tags.includes(tag) || formData.tags.push(tag)">
                {{ tag }}
              </span>
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

          <div class="form-item">
            <label>上传算法文件（ZIP 格式，可选）</label>
            <div class="file-upload">
              <input 
                type="file" 
                @change="handleFileChange"
                accept=".zip"
                id="algorithm-file"
              />
              <label for="algorithm-file" class="upload-btn">
                <span class="icon">📁</span>
                <span class="text">选择文件</span>
              </label>
              <span v-if="selectedFile" class="file-name">{{ selectedFile.name }}</span>
            </div>
            <span class="hint">文件将自动保存到 MinIO 并创建第一个版本</span>
          </div>

          <div class="modal-footer">
            <button class="secondary" type="button" @click="showCreateModal = false">取消</button>
            <button type="submit" class="btn-primary">创建</button>
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

.algorithms-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: var(--space-lg);
}

.algorithm-card {
  background: var(--bg-card);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  transition: all var(--transition-base);
  position: relative;
  overflow: hidden;
}

.algorithm-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: var(--accent-primary);
  transform: scaleX(0);
  transition: transform var(--transition-base);
}

.algorithm-card:hover {
  border-color: var(--accent-primary);
  box-shadow: var(--shadow-lg);
}

.algorithm-card:hover::before {
  transform: scaleX(1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--space-sm);
}

.card-header h3 {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.meta-badges {
  display: flex;
  gap: var(--space-sm);
  margin-top: var(--space-xs);
}

.platform-badge {
  font-size: 11px;
  padding: 2px 8px;
  background: var(--bg-secondary);
  color: var(--accent-primary);
  border: 1px solid var(--accent-primary);
  border-radius: var(--radius-sm);
  font-weight: 500;
}

.language-tag {
  font-size: 11px;
  padding: 2px 8px;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  font-weight: 500;
}

.description {
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  line-height: 1.6;
  flex: 1;
  margin: 0;
}

.meta {
  display: flex;
  gap: var(--space-lg);
  font-size: 13px;
  color: var(--text-muted);
  flex-wrap: wrap;
}

.view-btn {
  align-self: flex-start;
  padding: var(--space-sm) 0;
  font-size: var(--font-size-sm);
  color: var(--accent-primary);
  font-weight: 500;
  transition: all var(--transition-fast);
  display: inline-flex;
  align-items: center;
  gap: var(--space-xs);
}

.view-btn:hover {
  color: var(--accent-hover);
}

.arrow {
  transition: transform var(--transition-fast);
}

.view-btn:hover .arrow {
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
  max-width: 700px;
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

.available-tags {
  padding: var(--space-sm);
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
}

.hint {
  display: block;
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: var(--space-sm);
}

.suggested-tag {
  display: inline-block;
  padding: 2px 8px;
  margin-right: var(--space-xs);
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.suggested-tag:hover {
  background: var(--accent-primary);
  color: white;
  border-color: var(--accent-primary);
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

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-md);
  margin-top: var(--space-xl);
  padding-top: var(--space-lg);
  border-top: 1px solid var(--border-light);
}

button.btn-primary {
  background: var(--accent-primary);
  color: #ffffff;
  border: none;
}

button.btn-primary:hover {
  background: var(--accent-hover);
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

[type="file"] {
  display: none;
}
</style>
