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
      <button @click="showCreateModal = true">创建算法</button>
    </div>

    <div v-if="algorithmStore.loading" class="loading">加载中...</div>

    <div v-else class="algorithms-grid">
      <div v-for="algorithm in algorithmStore.algorithms" :key="algorithm.id" class="algorithm-card card">
        <div class="card-header">
          <h3>{{ algorithm.name }}</h3>
          <span class="language-tag">{{ algorithm.language }}</span>
        </div>
        <p class="description">{{ algorithm.description }}</p>
        <div class="meta">
          <span>类别: {{ algorithm.category }}</span>
          <span>版本: {{ algorithm.current_version_id }}</span>
        </div>
        <RouterLink :to="`/algorithms/${algorithm.id}`" class="view-btn">查看详情</RouterLink>
      </div>
    </div>

    <div v-if="showCreateModal" class="modal">
      <div class="modal-content card">
        <h3>创建算法</h3>
        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label>名称</label>
            <input v-model="formData.name" required />
          </div>
          <div class="form-group">
            <label>描述</label>
            <textarea v-model="formData.description" rows="3" required></textarea>
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
            <input v-model="formData.entrypoint" required />
          </div>
          <div class="modal-actions">
            <button type="button" @click="showCreateModal = false">取消</button>
            <button type="submit">创建</button>
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

.algorithms-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: calc(var(--grid) * 3);
}

.algorithm-card {
  display: flex;
  flex-direction: column;
  gap: calc(var(--grid) * 1.5);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.algorithm-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.card-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.language-tag {
  font-size: 11px;
  padding: 2px 6px;
  background-color: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 2px;
  text-transform: uppercase;
}

.description {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.meta {
  display: flex;
  gap: calc(var(--grid) * 2);
  font-size: 12px;
  color: var(--text-secondary);
}

.view-btn {
  align-self: flex-start;
  padding: calc(var(--grid) / 2) calc(var(--grid));
  font-size: 12px;
}

.modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  max-width: 500px;
  width: 90%;
}

.modal-content h3 {
  margin-bottom: calc(var(--grid) * 2);
}

.form-group {
  margin-bottom: calc(var(--grid) * 2);
}

.form-group label {
  display: block;
  margin-bottom: calc(var(--grid));
  font-size: 12px;
  color: var(--text-secondary);
  text-transform: uppercase;
}

.modal-actions {
  display: flex;
  gap: calc(var(--grid) * 2);
  margin-top: calc(var(--grid) * 3);
}
</style>
