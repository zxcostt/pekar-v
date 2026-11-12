<script setup>
import { ref } from 'vue'
import { api } from '../../api'
import { UserPlus, Users, ShieldCheck, CheckCircle2, UserMinus, UserX } from 'lucide-vue-next'

const props = defineProps({
  users: {
    type: Array,
    default: () => []
  },
  currentUser: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['user-created', 'user-dismissed', 'notify'])

const form = ref({
  fullName: '',
  login: '',
  password: '',
  role: 'CASHIER',
  registerNumber: 1,
  workshopStation: 'Печной цех №1',
  warehouseZoneId: 101
})

const isSubmitting = ref(false)

function getRoleLabel(role) {
  switch (role) {
    case 'CASHIER':
      return 'Кассир'
    case 'BAKER':
      return 'Пекарь'
    case 'STOREKEEPER':
      return 'Кладовщик'
    case 'MANAGER':
      return 'Управляющий'
    default:
      return role
  }
}

function getRoleBadgeClass(role) {
  switch (role) {
    case 'CASHIER':
      return 'bg-emerald-50 text-emerald-700 border-emerald-200'
    case 'BAKER':
      return 'bg-amber-50 text-amber-800 border-amber-200'
    case 'STOREKEEPER':
      return 'bg-purple-50 text-purple-700 border-purple-200'
    case 'MANAGER':
      return 'bg-blue-50 text-blue-700 border-blue-200'
    default:
      return 'bg-stone-50 text-stone-700 border-stone-200'
  }
}

async function handleCreateUser() {
  if (!form.value.login.trim() || !form.value.password.trim()) {
    return emit('notify', 'Логин и пароль обязательны для заполнения', 'error')
  }

  isSubmitting.value = true
  try {
    const payload = {
      fullName: form.value.fullName.trim() || form.value.login.trim(),
      login: form.value.login.trim(),
      password: form.value.password.trim(),
      role: form.value.role
    }

    if (form.value.role === 'CASHIER') {
      payload.registerNumber = Number(form.value.registerNumber) || 1
    } else if (form.value.role === 'BAKER') {
      payload.workshopStation = form.value.workshopStation || 'Печной цех №1'
    } else if (form.value.role === 'STOREKEEPER') {
      payload.warehouseZoneId = Number(form.value.warehouseZoneId) || 101
    }

    const created = await api.createUser(payload)
    emit('notify', `Сотрудник ${created.fullName} (${getRoleLabel(created.role)}) успешно добавлен!`)

    // Сброс формы
    form.value = {
      fullName: '',
      login: '',
      password: '',
      role: 'CASHIER',
      registerNumber: 1,
      workshopStation: 'Печной цех №1',
      warehouseZoneId: 101
    }

    emit('user-created', created)
  } catch (err) {
    emit('notify', err.message, 'error')
  } finally {
    isSubmitting.value = false
  }
}

async function handleDismissUser(user) {
  if (user.id === props.currentUser.id) {
    return emit('notify', 'Нельзя уволить самого себя', 'error')
  }

  const actionName = user.isActive ? 'уволить' : 'удалить'
  if (!confirm(`Вы действительно хотите ${actionName} сотрудника «${user.fullName}» (${user.login})?`)) {
    return
  }

  try {
    const res = await api.deleteUser(user.id, props.currentUser.id)
    emit('notify', res.message)
    emit('user-dismissed')
  } catch (err) {
    emit('notify', err.message, 'error')
  }
}
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Форма создания сотрудника -->
    <div class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
      <div class="flex items-center gap-2 mb-2">
        <UserPlus class="w-5 h-5 text-amber-700"/>
        <h2 class="text-lg font-bold">Добавить сотрудника</h2>
      </div>
      <p class="text-xs text-stone-500 mb-5">
        Регистрация нового пользователя системы с назначением прав доступа и роли.
      </p>

      <form @submit.prevent="handleCreateUser" class="space-y-4">
        <div>
          <label class="block text-xs font-semibold uppercase text-stone-600 mb-1">ФИО сотрудника</label>
          <input
            v-model="form.fullName"
            type="text"
            placeholder="Например: Смирнова Елена"
            class="w-full px-3.5 py-2.5 rounded-xl border border-stone-300 text-sm focus:ring-2 focus:ring-amber-500 outline-none"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold uppercase text-stone-600 mb-1">
            Логин в системе <span class="text-red-500">*</span>
          </label>
          <input
            v-model="form.login"
            type="text"
            required
            placeholder="Например: cashier2"
            class="w-full px-3.5 py-2.5 rounded-xl border border-stone-300 text-sm focus:ring-2 focus:ring-amber-500 outline-none"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold uppercase text-stone-600 mb-1">
            Пароль для входа <span class="text-red-500">*</span>
          </label>
          <input
            v-model="form.password"
            type="password"
            required
            placeholder="••••••••"
            class="w-full px-3.5 py-2.5 rounded-xl border border-stone-300 text-sm focus:ring-2 focus:ring-amber-500 outline-none"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold uppercase text-stone-600 mb-1">
            Должность (Роль) <span class="text-red-500">*</span>
          </label>
          <select
            v-model="form.role"
            class="w-full px-3.5 py-2.5 rounded-xl border border-stone-300 text-sm bg-white focus:ring-2 focus:ring-amber-500 outline-none"
          >
            <option value="CASHIER">🛒 Кассир (Кассовый узел / POS)</option>
            <option value="BAKER">🥐 Пекарь (Производственный цех)</option>
            <option value="STOREKEEPER">📦 Кладовщик (Склад сырья)</option>
            <option value="MANAGER">📊 Управляющий (Полный доступ)</option>
          </select>
        </div>

        <!-- Специфичные поля в зависимости от выбранной роли (Single Table Inheritance) -->
        <div v-if="form.role === 'CASHIER'" class="p-3 bg-stone-50 rounded-xl space-y-1">
          <label class="block text-xs font-medium text-stone-600">Номер кассы / фискального регистратора</label>
          <input
            v-model.number="form.registerNumber"
            type="number"
            min="1"
            class="w-full px-3 py-1.5 text-sm bg-white border border-stone-300 rounded-lg"
          />
        </div>

        <div v-else-if="form.role === 'BAKER'" class="p-3 bg-stone-50 rounded-xl space-y-1">
          <label class="block text-xs font-medium text-stone-600">Рабочая станция / цех</label>
          <input
            v-model="form.workshopStation"
            type="text"
            class="w-full px-3 py-1.5 text-sm bg-white border border-stone-300 rounded-lg"
          />
        </div>

        <div v-else-if="form.role === 'STOREKEEPER'" class="p-3 bg-stone-50 rounded-xl space-y-1">
          <label class="block text-xs font-medium text-stone-600">Номер складской зоны</label>
          <input
            v-model.number="form.warehouseZoneId"
            type="number"
            min="1"
            class="w-full px-3 py-1.5 text-sm bg-white border border-stone-300 rounded-lg"
          />
        </div>

        <button
          type="submit"
          :disabled="isSubmitting"
          class="w-full py-3 bg-amber-600 hover:bg-amber-700 disabled:bg-stone-300 text-white font-semibold rounded-xl shadow transition cursor-pointer"
        >
          {{ isSubmitting ? 'Сохранение...' : 'Зарегистрировать сотрудника' }}
        </button>
      </form>
    </div>

    <!-- Список сотрудников -->
    <div class="lg:col-span-2 bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-2">
          <Users class="w-5 h-5 text-amber-700"/>
          <h3 class="font-bold text-lg">Штат сотрудников пекарни</h3>
        </div>
        <span class="text-xs px-2.5 py-1 rounded-full bg-stone-100 font-semibold text-stone-600">
          Всего: {{ users.length }}
        </span>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead class="text-xs uppercase bg-stone-50 text-stone-500">
            <tr>
              <th class="p-3">ID</th>
              <th class="p-3">ФИО</th>
              <th class="p-3">Логин</th>
              <th class="p-3">Должность</th>
              <th class="p-3">Назначение</th>
              <th class="p-3">Статус</th>
              <th class="p-3">Действия</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-stone-200">
            <tr v-for="u in users" :key="u.id">
              <td class="p-3 font-mono font-medium text-stone-400">#{{ u.id }}</td>
              <td class="p-3 font-semibold text-stone-900">{{ u.fullName }}</td>
              <td class="p-3 font-mono text-xs text-stone-600">{{ u.login }}</td>
              <td class="p-3">
                <span
                  :class="[
                    'px-2.5 py-1 rounded-full text-xs font-semibold border',
                    getRoleBadgeClass(u.role)
                  ]"
                >
                  {{ getRoleLabel(u.role) }}
                </span>
              </td>
              <td class="p-3 text-xs text-stone-500">
                <span v-if="u.role === 'CASHIER'">Касса №{{ u.registerNumber || 1 }}</span>
                <span v-else-if="u.role === 'BAKER'">{{ u.workshopStation || 'Цех №1' }}</span>
                <span v-else-if="u.role === 'STOREKEEPER'">Зона склада #{{ u.warehouseZoneId || 101 }}</span>
                <span v-else-if="u.role === 'MANAGER'" class="flex items-center gap-1 text-blue-700">
                  <ShieldCheck class="w-3.5 h-3.5"/> Администрация
                </span>
                <span v-else>—</span>
              </td>
              <td class="p-3">
                <span
                  v-if="u.isActive"
                  class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700"
                >
                  <CheckCircle2 class="w-3 h-3"/> Активен
                </span>
                <span
                  v-else
                  class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-700"
                >
                  <UserX class="w-3 h-3"/> Уволен
                </span>
              </td>
              <td class="p-3">
                <span v-if="u.id === currentUser.id" class="text-xs text-stone-400 italic">
                  (Текущий профиль)
                </span>
                <button
                  v-else-if="u.isActive"
                  @click="handleDismissUser(u)"
                  class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-red-50 hover:bg-red-100 text-red-700 transition cursor-pointer"
                  title="Уволить сотрудника"
                >
                  <UserMinus class="w-3.5 h-3.5"/> Уволить
                </button>
                <button
                  v-else
                  @click="handleDismissUser(u)"
                  class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-stone-100 hover:bg-stone-200 text-stone-600 transition cursor-pointer"
                  title="Удалить запись из базы"
                >
                  Удалить
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
