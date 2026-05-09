import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'

import LoginPage from './pages/LoginPage.vue'
import MailPage from './pages/MailPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/mail' },
    { path: '/login', component: LoginPage },
    { path: '/mail', component: MailPage },
    { path: '/mail/:folder', component: MailPage, props: true },
  ],
})

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
