import { createRouter, createWebHashHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import MainLayout from '../layouts/MainLayout.vue'
import ChatView from '../views/ChatView.vue'
import NotesView from '../views/NotesView.vue'
import TodosView from '../views/TodosView.vue'
import PlansView from '../views/PlansView.vue'
import KnowledgeView from '../views/KnowledgeView.vue'
import ScheduleView from '../views/ScheduleView.vue'

const routes = [
  { path: '/login', name: 'Login', component: Login },
  { path: '/register', name: 'Register', component: Register },
  {
    path: '/',
    component: MainLayout,
    children: [
      { path: '', redirect: '/chat' },
      { path: 'chat', name: 'Chat', component: ChatView },
      { path: 'chat/:sessionId', name: 'ChatSession', component: ChatView },
      { path: 'notes', name: 'Notes', component: NotesView },
      { path: 'notes/:id', name: 'NoteDetail', component: NotesView },
      { path: 'todos', name: 'Todos', component: TodosView },
      { path: 'todos/:id', name: 'TodoDetail', component: TodosView },
      { path: 'plans', name: 'Plans', component: PlansView },
      { path: 'plans/:id', name: 'PlanDetail', component: PlansView },
      { path: 'knowledge', name: 'Knowledge', component: KnowledgeView },
      { path: 'schedules', name: 'Schedules', component: ScheduleView },
      { path: 'schedules/:id', name: 'ScheduleDetail', component: ScheduleView },
    ],
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
