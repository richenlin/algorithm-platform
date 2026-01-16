<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useAlgorithmStore } from '../stores/algorithm'

const algorithmStore = useAlgorithmStore()
const showCreateModal = ref(false)
const formData = ref({
  name: '',
  description: '',
  language: 'python',
  platform: 'docker',
  category: 'general',
  entrypoint: 'main.py'
})

onMounted(() => {
  algorithmStore.fetchAlgorithms()
})

async function handleSubmit() {
  try {
    await algorithmStore.createAlgorithm(formData.value)
    showCreateModal.value = false
    formData.value = {
      name: '',
      description: '',
      language: 'python',
      platform: 'docker',
      category: 'general',
      entrypoint: 'main.py'
    }
  } catch (error) {
    console.error('Failed to create algorithm:', error)
  }
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
      <div v-for="algorithm in algorithmStore.algorithms" :key="algorithm.id" class="algorithm-card">
        <div class="card-header">
          <div>
            <h3>{{ algorithm.name }}</h3>
            <span class="language-tag">{{ algorithm.language }}</span>
          </div>
        </div>
        <p class="description">{{ algorithm.description }}</p>
        <div class="meta">
          <span>类别: {{ algorithm.category }}</span>
          <span>版本: {{ algorithm.currentVersionId }}</span>
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
              </select>
            </div>
            <div class="form-item">
              <label>平台</label>
              <select v-model="formData.platform">
                <option value="docker">Docker</option>
              </select>
            </div>
          </div>
          <div class="form-row">
            <div class="form-item">
              <label>类别</label>
              <select v-model="formData.category">
                <option value="general">通用</option>
                <option value="ml">机器学习</option>
                <option value="cv">计算机视觉</option>
                <option value="nlp">自然语言处理</option>
                <option value="optimization">优化</option>
              </select>
            </div>
            <div class="form-item">
              <label>入口文件 <span class="required">*</span></label>
              <input v-model="formData.entrypoint" placeholder="例如: main.py" required />
            </div>
          </div>
          <div class="modal-footer">
            <button class="secondary" @click="showCreateModal = false">取消</button>
             <button class="btn-primary">创建</button>
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

button.btn-primary:hover {
  background: var(--accent-hover);
}

.algorithms-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
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

button.secondary:hover {
  color: var(--text-primary);
  border-color: var(--accent-primary);
}
</style>
