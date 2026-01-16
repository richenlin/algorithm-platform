import { defineStore } from 'pinia'
import { ref } from 'vue'
import { jobApi } from '../api/types'
import type { JobSummary, JobDetail } from '../api/types'

export const useJobStore = defineStore('job', () => {
  const jobs = ref<JobSummary[]>([])
  const currentJob = ref<JobDetail | null>(null)
  const loading = ref(false)

  async function fetchJobs(params?: { algorithm_id?: string; status?: string; page?: number; page_size?: number }) {
    loading.value = true
    try {
      const response = await jobApi.list(params || {})
      jobs.value = response.data.jobs
    } catch (error) {
      console.error('Failed to fetch jobs:', error)
    } finally {
      loading.value = false
    }
  }

  async function fetchJobDetail(jobId: string) {
    loading.value = true
    try {
      const response = await jobApi.detail(jobId)
      currentJob.value = response.data
      return response.data
    } catch (error) {
      console.error('Failed to fetch job detail:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  function getCurrentJob() {
    return currentJob.value
  }

  async function checkJobStatus(jobId: string) {
    try {
      const response = await jobApi.status(jobId)
      const index = jobs.value.findIndex(j => j.jobId === jobId)
      if (index !== -1 && jobs.value[index]) {
        jobs.value[index].status = response.data.status
      }
      return response.data
    } catch (error) {
      console.error('Failed to check job status:', error)
      throw error
    }
  }

  async function executeAlgorithm(algorithmId: string, data: {
    mode: string
    params: Record<string, string>
    input_source: { type: string; url: string }
    webhook_url?: string
    force_refresh?: boolean
    resource_config?: { cpu_limit: number; memory_limit: string }
    timeout_seconds?: number
  }) {
    loading.value = true
    try {
      const response = await jobApi.execute(algorithmId, data)
      return response.data
    } catch (error) {
      console.error('Failed to execute algorithm:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  return {
    jobs,
    currentJob,
    loading,
    fetchJobs,
    fetchJobDetail,
    getCurrentJob,
    checkJobStatus,
    executeAlgorithm
  }
})
