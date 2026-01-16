import api from './index'

export interface Algorithm {
  id: string
  name: string
  description: string
  language: string
  platform: string
  category: string
  entrypoint: string
  current_version_id: string
  created_at: string
  updated_at: string
}

export interface Version {
  id: string
  algorithm_id: string
  version_number: number
  minio_path: string
  commit_message: string
  created_at: string
}

export interface PresetData {
  id: string
  filename: string
  category: string
  minio_url: string
  created_at: string
}

export interface JobSummary {
  job_id: string
  algorithm_id: string
  algorithm_name: string
  status: string
  created_at: string
  cost_time_ms: number
}

export interface JobDetail {
  job_id: string
  algorithm_id: string
  algorithm_name: string
  mode: string
  status: string
  input_params: string
  input_url: string
  output_url: string
  log_url: string
  created_at: string
  started_at: string
  finished_at: string
  cost_time_ms: number
  worker_id: string
}

export const algorithmApi = {
  list: (params: { category?: string; language?: string; page?: number; page_size?: number }) =>
    api.get<{ algorithms: Algorithm[]; total: number }>('/api/v1/algorithms', { params }),

  create: (data: { name: string; description: string; language: string; platform: string; category: string; entrypoint: string }) =>
    api.post<Algorithm>('/api/v1/algorithms', data),

  update: (id: string, data: { name: string; description: string; category: string }) =>
    api.put<Algorithm>(`/api/v1/algorithms/${id}`, data),

  createVersion: (algorithmId: string, data: { source_code_zip_url: string; commit_message: string }) =>
    api.post<Version>(`/api/v1/algorithms/${algorithmId}/versions`, data),

  rollbackVersion: (algorithmId: string, versionId: string) =>
    api.post<Algorithm>(`/api/v1/algorithms/${algorithmId}/versions/${versionId}/rollback`)
}

export const dataApi = {
  list: (params: { category?: string; page?: number; page_size?: number }) =>
    api.get<{ files: PresetData[]; total: number }>('/api/v1/data', { params }),

  upload: (data: { filename: string; category: string; minio_path: string }) =>
    api.post<{ file_id: string; minio_url: string }>('/api/v1/data/upload', data)
}

export const jobApi = {
  list: (params: { algorithm_id?: string; status?: string; page?: number; page_size?: number }) =>
    api.get<{ jobs: JobSummary[]; total: number }>('/api/v1/jobs', { params }),

  detail: (jobId: string) =>
    api.get<JobDetail>(`/api/v1/jobs/${jobId}/detail`),

  status: (jobId: string) =>
    api.get<{ job_id: string; status: string; result_url: string; started_at: string; finished_at: string; cost_time_ms: number }>(`/api/v1/jobs/${jobId}`),

  execute: (algorithmId: string, data: {
    mode: string
    params: Record<string, string>
    input_source: { type: string; url: string }
    webhook_url?: string
    force_refresh?: boolean
    resource_config?: { cpu_limit: number; memory_limit: string }
    timeout_seconds?: number
  }) =>
    api.post<{ job_id: string; status: string; result_url: string; message: string }>(`/api/v1/algorithms/${algorithmId}/execute`, data)
}
