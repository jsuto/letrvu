import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import { useAuthStore } from './stores/auth'

import LoginPage from './pages/LoginPage.vue'
import MailPage from './pages/MailPage.vue'

const pinia = createPinia()

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/mail' },
    { path: '/login', component: LoginPage },
    { path: '/mail', component: MailPage, meta: { requiresAuth: true } },
    { path: '/mail/:folder', component: MailPage, props: true, meta: { requiresAuth: true } },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore(pinia)

  // For protected routes and the login page, verify session state once if unknown.
  if ((to.meta.requiresAuth || to.path === '/login') && !auth.loggedIn) {
    await auth.checkSession()
  }

  if (to.meta.requiresAuth && !auth.loggedIn) return '/login'
  if (to.path === '/login' && auth.loggedIn) return '/mail'
})

const app = createApp(App)
app.use(pinia)
app.use(router)
app.mount('#app')
