<script setup>
import { ref } from 'vue'
import { Croissant } from 'lucide-vue-next'

defineProps({
  authError: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['login'])

const loginForm = ref({ login: '', password: '' })

function handleSubmit() {
  emit('login', {
    login: loginForm.value.login,
    password: loginForm.value.password
  })
}

function quickLogin(login, password) {
  loginForm.value.login = login
  loginForm.value.password = password
  handleSubmit()
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-4">
    <div class="bg-white w-full max-w-md rounded-2xl shadow-xl p-8 border border-stone-200">
      <div class="flex flex-col items-center mb-6">
        <div class="w-16 h-16 bg-amber-100 text-amber-700 rounded-2xl flex items-center justify-center mb-3">
          <Croissant class="w-9 h-9"/>
        </div>
        <h1 class="text-2xl font-bold text-stone-900">АИС «Пекарь-В»</h1>
        <p class="text-sm text-stone-500">Пекарня «Восход» — Вход в систему</p>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-stone-700 mb-1">Логин сотрудника</label>
          <input
            v-model="loginForm.login"
            type="text"
            required
            placeholder="Например: cashier"
            class="w-full px-4 py-2.5 rounded-xl border border-stone-300 focus:ring-2 focus:ring-amber-500 outline-none"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-stone-700 mb-1">Пароль</label>
          <input
            v-model="loginForm.password"
            type="password"
            required
            placeholder="•••••••"
            class="w-full px-4 py-2.5 rounded-xl border border-stone-300 focus:ring-2 focus:ring-amber-500 outline-none"
          />
        </div>

        <div v-if="authError" class="p-3 rounded-xl bg-red-50 border border-red-200 text-red-700 text-sm">
          {{ authError }}
        </div>

        <button
          type="submit"
          class="w-full py-3 bg-amber-600 hover:bg-amber-700 text-white font-semibold rounded-xl shadow transition cursor-pointer"
        >
          Войти в систему
        </button>
      </form>

      <!-- Быстрый вход по ролям для демонстрации на защите -->
      <div class="mt-6 pt-5 border-t border-stone-200">
        <p class="text-xs text-stone-500 uppercase tracking-wider font-semibold mb-3 text-center">
          Быстрый вход (Демонстрация ролей)
        </p>
        <div class="grid grid-cols-2 gap-2 text-xs">
          <button
            @click="quickLogin('cashier', 'pass123')"
            class="p-2.5 rounded-lg bg-stone-100 hover:bg-amber-50 border border-stone-200 font-medium text-left cursor-pointer"
          >
            🛒 <b>Кассир</b><br /><span class="text-stone-500">cashier / pass123</span>
          </button>
          <button
            @click="quickLogin('baker', 'pass123')"
            class="p-2.5 rounded-lg bg-stone-100 hover:bg-amber-50 border border-stone-200 font-medium text-left cursor-pointer"
          >
            🥐 <b>Пекарь</b><br /><span class="text-stone-500">baker / pass123</span>
          </button>
          <button
            @click="quickLogin('store', 'pass123')"
            class="p-2.5 rounded-lg bg-stone-100 hover:bg-amber-50 border border-stone-200 font-medium text-left cursor-pointer"
          >
            📦 <b>Кладовщик</b><br /><span class="text-stone-500">store / pass123</span>
          </button>
          <button
            @click="quickLogin('admin', 'admin123')"
            class="p-2.5 rounded-lg bg-stone-100 hover:bg-amber-50 border border-stone-200 font-medium text-left cursor-pointer"
          >
            📊 <b>Управляющий</b><br /><span class="text-stone-500">admin / admin123</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
