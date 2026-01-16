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
    <div class="header-actions fade-in-up">
      <h2>算法列表</h2>
      <button @click="showCreateModal = true">+ 创建算法</button>
    </div>

    <div v-if="algorithmStore.loading" class="loading fade-in-up">加载中...</div>

    <div v-else class="algorithms-grid">
      <div v-for="(algorithm, index) in algorithmStore.algorithms" :key="algorithm.id" 
           class="algorithm-card card fade-in-up" 
           :style="{ animationDelay: `${index * 50}ms` }">
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
        <RouterLink :to="`/algorithms/${algorithm.id}`" class="view-btn">查看详情 →</RouterLink>
      </div>
    </div>

    <div v-if="showCreateModal" class="modal fade-in-up">
      <div class="modal-backdrop" @click="showCreateModal = false"></div>
      <div class="modal-content card">
        <h3>创建算法</h3>
        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label>名称</label>
            <input v-model="formData.name" placeholder="输入算法名称" required />
          </div>
          <div class="form-group">
            <label>描述</label>
            <textarea v-model="formData.description" rows="3" placeholder="描述算法用途" required></textarea>
          </div>
          <div class="form-group">
            <label>语言</label>
            <select v-model="formData.language">
              <option value="python">Python</option>
              <option value="matlab">Matlab</option>
              <option value="cpp">C++</option>
              <option value="java">Java</option>
            </select>
          </div>
          <div class="form-group">
            <label>平台</label>
            <select v-model="formData.platform">
              <option value="docker">Docker</option>
            </select>
          </div>
          <div class="form-group">
            <label>类别</label>
            <select v-model="formData.category">
              <option value="general">通用</option>
              <option value="ml">机器学习</option>
              <option value="cv">计算机视觉</option>
              <option value="nlp">自然语言处理</option>
              <option value="optimization">优化</option>
            </select>
          </div>
          <div class="form-group">
            <label>入口文件</label>
            <input v-model="formData.entrypoint" placeholder="例如: main.py" required />
          </div>
          <div class="modal-actions">
            <button type="button" @click="showCreateModal = false" class="secondary">取消</button>
            <button type="submit">创建</button>
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

.algorithms-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: var(--space-lg);
}

.algorithm-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  min-height: 220px;
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
  background: linear-gradient(90deg, var(--accent-primary), var(--accent-secondary));
  opacity: 0;
  transition: opacity var(--transition-base);
}

.algorithm-card:hover::before {
  opacity: 1;
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
  padding: 4px 10px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-default);
  border-radius: 20px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-secondary);
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
}

.view-btn:hover {
  color: var(--accent-secondary);
  transform: translateX(4px);
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
}

.modal-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
}

.modal-content {
  position: relative;
  max-width: 520px;
  width: 100%;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-content h3 {
  font-size: var(--font-size-xl);
  margin-bottom: var(--space-lg);
  padding-bottom: var(--space-md);
  border-bottom: 1px solid var(--border-subtle);
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
