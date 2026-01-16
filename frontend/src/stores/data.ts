import { defineStore } from 'pinia'
import { ref } from 'vue'
import { dataApi } from '../api/types'
import type { PresetData } from '../api/types'

export const useDataStore = defineStore('data', () => {
  const files = ref<PresetData[]>([])
  const loading = ref(false)

  async function fetchFiles(params?: { category?: string; page?: number; page_size?: number }) {
    loading.value = true
    try {
      const response = await dataApi.list(params || {})
      files.value = response.data.files
    } catch (error) {
      console.error('Failed to fetch files:', error)
    } finally {
      loading.value = false
    }
  }

  async function uploadFile(data: { filename: string; category: string; minio_path: string }) {
    loading.value = true
    try {
      const response = await dataApi.upload(data)
      return response.data
    } catch (error) {
      console.error('Failed to upload file:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  return {
    files,
    loading,
    fetchFiles,
    uploadFile
  }
})
