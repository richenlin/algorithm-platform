import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/algorithms'
  },
  {
    path: '/algorithms',
    name: 'Algorithms',
    component: () => import('../views/Algorithms.vue')
  },
  {
    path: '/algorithms/:id',
    name: 'AlgorithmDetail',
    component: () => import('../views/AlgorithmDetail.vue')
  },
  {
    path: '/jobs',
    name: 'Jobs',
    component: () => import('../views/Jobs.vue')
  },
  {
    path: '/jobs/:id',
    name: 'JobDetail',
    component: () => import('../views/JobDetail.vue')
  },
  {
    path: '/data',
    name: 'Data',
    component: () => import('../views/Data.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
