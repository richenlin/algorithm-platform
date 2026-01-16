import { defineStore } from 'pinia'
import { ref } from 'vue'
import { algorithmApi } from '../api/types'
import type { Algorithm, Version } from '../api/types'

export const useAlgorithmStore = defineStore('algorithm', () => {
  const algorithms = ref<Algorithm[]>([])
  const currentAlgorithm = ref<Algorithm | null>(null)
  const versions = ref<Version[]>([])
  const loading = ref(false)

  async function fetchAlgorithms(params?: { category?: string; language?: string; page?: number; page_size?: number }) {
    loading.value = true
    try {
      const response = await algorithmApi.list(params || {})
      algorithms.value = response.data.algorithms
    } catch (error) {
      console.error('Failed to fetch algorithms:', error)
    } finally {
      loading.value = false
    }
  }

  async function fetchAlgorithm(id: string) {
    loading.value = true
    try {
      const response = await algorithmApi.list({ page: 1, page_size: 100 })
      currentAlgorithm.value = response.data.algorithms.find(a => a.id === id) || null
    } catch (error) {
      console.error('Failed to fetch algorithm:', error)
    } finally {
      loading.value = false
    }
  }

  async function createAlgorithm(data: { name: string; description: string; language: string; platform: string; category: string; entrypoint: string }) {
    loading.value = true
    try {
      const response = await algorithmApi.create(data)
      algorithms.value.push(response.data)
      return response.data
    } catch (error) {
      console.error('Failed to create algorithm:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function updateAlgorithm(id: string, data: { name: string; description: string; category: string }) {
    loading.value = true
    try {
      const response = await algorithmApi.update(id, data)
      const index = algorithms.value.findIndex(a => a.id === id)
      if (index !== -1) {
        algorithms.value[index] = response.data
      }
      return response.data
    } catch (error) {
      console.error('Failed to update algorithm:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function createVersion(algorithmId: string, data: { source_code_zip_url: string; commit_message: string }) {
    loading.value = true
    try {
      const response = await algorithmApi.createVersion(algorithmId, data)
      versions.value.push(response.data)
      return response.data
    } catch (error) {
      console.error('Failed to create version:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function rollbackVersion(algorithmId: string, versionId: string) {
    loading.value = true
    try {
      const response = await algorithmApi.rollbackVersion(algorithmId, versionId)
      currentAlgorithm.value = response.data
      return response.data
    } catch (error) {
      console.error('Failed to rollback version:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  return {
    algorithms,
    currentAlgorithm,
    versions,
    loading,
    fetchAlgorithms,
    fetchAlgorithm,
    createAlgorithm,
    updateAlgorithm,
    createVersion,
    rollbackVersion
  }
})
