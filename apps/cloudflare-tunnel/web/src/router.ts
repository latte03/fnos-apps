import { createRouter, createWebHashHistory } from 'vue-router'
import OverviewView from './views/OverviewView.vue'
import TunnelsView from './views/TunnelsView.vue'
import RoutesView from './views/RoutesView.vue'
import LogsView from './views/LogsView.vue'
import ApiView from './views/ApiView.vue'
import SettingsView from './views/SettingsView.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/overview' },
    { path: '/overview', name: 'overview', component: OverviewView, meta: { title: '概览' } },
    { path: '/tunnels', name: 'tunnels', component: TunnelsView, meta: { title: '隧道' } },
    { path: '/routes', name: 'routes', component: RoutesView, meta: { title: '路由' } },
    { path: '/logs', name: 'logs', component: LogsView, meta: { title: '运行日志' } },
    { path: '/api', name: 'api', component: ApiView, meta: { title: '接入' } },
    { path: '/settings', name: 'settings', component: SettingsView, meta: { title: '设置' } },
    { path: '/:pathMatch(.*)*', redirect: '/overview' },
  ],
  scrollBehavior() {
    return { top: 0 }
  },
})

export default router
