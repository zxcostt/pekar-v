<script setup>
import {
  Croissant,
  ShoppingCart,
  RotateCcw,
  ChefHat,
  Package,
  FileText,
  BarChart3,
  LogOut
} from 'lucide-vue-next'

const props = defineProps({
  currentUser: {
    type: Object,
    required: true
  },
  activeTab: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['update:activeTab', 'logout'])

function canAccess(tab) {
  if (!props.currentUser) return false
  const role = props.currentUser.role
  if (role === 'MANAGER') return true
  if (role === 'CASHIER') return ['pos', 'refunds', 'reports'].includes(tab)
  if (role === 'BAKER') return ['production', 'recipes'].includes(tab)
  if (role === 'STOREKEEPER') return ['warehouse'].includes(tab)
  return false
}
</script>

<template>
  <header class="bg-white border-b border-stone-200 px-6 py-3 flex items-center justify-between">
    <div class="flex items-center gap-3">
      <div class="w-10 h-10 bg-amber-100 text-amber-700 rounded-xl flex items-center justify-center">
        <Croissant class="w-6 h-6"/>
      </div>
      <div>
        <h1 class="font-bold text-lg leading-tight">АИС «Пекарь-В»</h1>
        <p class="text-xs text-stone-500">Пекарня «Восход»</p>
      </div>
    </div>

    <!-- Навигация по доступным модулям -->
    <nav class="flex gap-2">
      <button
        v-if="canAccess('pos')"
        @click="emit('update:activeTab', 'pos')"
        :class="[
          'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
          activeTab === 'pos' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
        ]"
      >
        <ShoppingCart class="w-4 h-4"/> Касса (Продажи)
      </button>

      <button
        v-if="canAccess('refunds')"
        @click="emit('update:activeTab', 'refunds')"
        :class="[
          'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
          activeTab === 'refunds' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
        ]"
      >
        <RotateCcw class="w-4 h-4"/> Чеки и Возвраты
      </button>

      <button
        v-if="canAccess('production')"
        @click="emit('update:activeTab', 'production')"
        :class="[
          'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
          activeTab === 'production' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
        ]"
      >
        <ChefHat class="w-4 h-4"/> Цех (Выпуск выпечки)
      </button>

      <button
        v-if="canAccess('warehouse')"
        @click="emit('update:activeTab', 'warehouse')"
        :class="[
          'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
          activeTab === 'warehouse' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
        ]"
      >
        <Package class="w-4 h-4"/> Склад сырья
      </button>

      <button
        v-if="canAccess('recipes')"
        @click="emit('update:activeTab', 'recipes')"
        :class="[
          'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
          activeTab === 'recipes' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
        ]"
      >
        <FileText class="w-4 h-4"/> Техкарты (Рецептуры)
      </button>

      <button
        v-if="canAccess('reports')"
        @click="emit('update:activeTab', 'reports')"
        :class="[
          'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
          activeTab === 'reports' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
        ]"
      >
        <BarChart3 class="w-4 h-4"/> Сменный отчёт
      </button>
    </nav>

    <!-- Профиль пользователя -->
    <div class="flex items-center gap-4">
      <div class="text-right">
        <div class="text-sm font-semibold">{{ currentUser.fullName }}</div>
        <div class="text-xs text-amber-700 font-medium">{{ currentUser.role }}</div>
      </div>
      <button
        @click="emit('logout')"
        class="p-2 rounded-xl bg-stone-100 hover:bg-red-50 hover:text-red-600 transition cursor-pointer"
        title="Выйти"
      >
        <LogOut class="w-5 h-5"/>
      </button>
    </div>
  </header>
</template>
