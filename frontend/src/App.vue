<script setup>
import { ref } from 'vue'
import { api } from './api'

import NotificationToast from './components/layout/NotificationToast.vue'
import AppHeader from './components/layout/AppHeader.vue'
import LoginView from './components/auth/LoginView.vue'

import PosTab from './components/tabs/PosTab.vue'
import RefundsTab from './components/tabs/RefundsTab.vue'
import ProductionTab from './components/tabs/ProductionTab.vue'
import WarehouseTab from './components/tabs/WarehouseTab.vue'
import RecipesTab from './components/tabs/RecipesTab.vue'
import ReportsTab from './components/tabs/ReportsTab.vue'
import EmployeesTab from './components/tabs/EmployeesTab.vue'

// --- Состояние авторизации (UC 1) ---
const currentUser = ref(null)
const authError = ref('')

// Активная вкладка (зависит от роли)
const activeTab = ref('pos')

// Глобальные уведомления
const notification = ref({ message: '', type: 'success' })
function notify(message, type = 'success') {
  notification.value = { message, type }
  setTimeout(() => {
    if (notification.value.message === message) {
      notification.value = { message: '', type: 'success' }
    }
  }, 4500)
}

// --- Данные из БД ---
const products = ref([])
const ingredients = ref([])
const recipes = ref([])
const productionOrders = ref([])
const productWriteOffs = ref([])
const receipts = ref([])
const users = ref([])
const currentShift = ref(null)
const shiftsHistory = ref([])
const shiftReport = ref(null)
const lastPrintedReceipt = ref(null)

// Загрузка всех справочников и документов
async function loadAllData() {
  try {
    const [prodData, ingData, recData, ordData, rcptData, repData, usersData, writeOffsData, shiftData, shiftsData] = await Promise.all([
      api.getProducts(),
      api.getIngredients(),
      api.getRecipes(),
      api.getProductionOrders(),
      api.getReceipts(),
      api.getShiftReport(),
      api.getUsers(),
      api.getProductWriteOffs(),
      api.getCurrentShift(),
      api.getShifts()
    ])
    products.value = prodData || []
    ingredients.value = ingData || []
    recipes.value = recData || []
    productionOrders.value = ordData || []
    receipts.value = rcptData || []
    shiftReport.value = repData
    users.value = usersData || []
    productWriteOffs.value = writeOffsData || []
    currentShift.value = shiftData?.shift || null
    shiftsHistory.value = shiftsData || []
  } catch (err) {
    notify(err.message, 'error')
  }
}

// --- UC 1: Вход в систему ---
async function handleLogin({ login, password }) {
  authError.value = ''
  try {
    const res = await api.login(login, password)
    currentUser.value = res.user
    // Выбираем стартовую вкладку по роли сотрудника
    if (res.user.role === 'CASHIER') activeTab.value = 'pos'
    else if (res.user.role === 'BAKER') activeTab.value = 'production'
    else if (res.user.role === 'STOREKEEPER') activeTab.value = 'warehouse'
    else activeTab.value = 'reports'

    await loadAllData()
    notify(`Добро пожаловать, ${res.user.fullName}!`)
  } catch (err) {
    authError.value = err.message
  }
}

function handleLogout() {
  currentUser.value = null
  lastPrintedReceipt.value = null
}

function handleCheckoutSuccess(res) {
  lastPrintedReceipt.value = res.receipt
  notify(res.message)
  loadAllData()
}
</script>

<template>
  <div class="min-h-screen bg-stone-100 text-stone-800">
    <!-- Всплывающее уведомление -->
    <NotificationToast :notification="notification" />

    <!-- ЭКРАН 1: АВТОРИЗАЦИЯ (UC 1) -->
    <LoginView
      v-if="!currentUser"
      :auth-error="authError"
      @login="handleLogin"
    />

    <!-- ЭКРАН 2: РАБОЧЕЕ МЕСТО СОТРУДНИКА -->
    <div v-else class="min-h-screen flex flex-col">
      <AppHeader
        :current-user="currentUser"
        :active-tab="activeTab"
        @update:active-tab="activeTab = $event"
        @logout="handleLogout"
      />

      <!-- ОСНОВНОЙ КОНТЕНТ -->
      <main class="flex-1 p-6 max-w-7xl w-full mx-auto">
        <!-- МОДУЛЬ 1: КАССА ПРОДАЖ (UC 8, 9, 10, 11, 12, 13) -->
        <PosTab
          v-if="activeTab === 'pos'"
          :products="products"
          :current-user="currentUser"
          :current-shift="currentShift"
          :last-printed-receipt="lastPrintedReceipt"
          @checkout-success="handleCheckoutSuccess"
          @writeoff-success="loadAllData"
          @shift-updated="loadAllData"
          @notify="notify"
        />

        <!-- МОДУЛЬ 2: ИСТОРИЯ ЧЕКОВ И ВОЗВРАТ ТОВАРА (UC 14) -->
        <RefundsTab
          v-if="activeTab === 'refunds'"
          :receipts="receipts"
          @refund-success="loadAllData"
          @notify="notify"
        />

        <!-- МОДУЛЬ 3: ПРОИЗВОДСТВЕННЫЙ ЦЕХ — ВЫПУСК ВЫПЕЧКИ (UC 6 + UC 7) И СПИСАНИЕ ВИТРИНЫ -->
        <ProductionTab
          v-if="activeTab === 'production'"
          :products="products"
          :production-orders="productionOrders"
          :product-write-offs="productWriteOffs"
          :current-user="currentUser"
          @produce-success="loadAllData"
          @writeoff-success="loadAllData"
          @notify="notify"
        />

        <!-- МОДУЛЬ 4: СКЛАД СЫРЯ (UC 4 + UC 5) -->
        <WarehouseTab
          v-if="activeTab === 'warehouse'"
          :ingredients="ingredients"
          @supply-success="loadAllData"
          @writeoff-success="loadAllData"
          @notify="notify"
        />

        <!-- МОДУЛЬ 5: СПРАВОЧНИК РЕЦЕПТУР / ТЕХКАРТ (UC 2) -->
        <RecipesTab
          v-if="activeTab === 'recipes'"
          :recipes="recipes"
          :ingredients="ingredients"
          :current-user="currentUser"
          @create-recipe-success="loadAllData"
          @notify="notify"
        />

        <!-- МОДУЛЬ 6: СМЕННЫЙ ОТЧЁТ (UC 3) -->
        <ReportsTab
          v-if="activeTab === 'reports'"
          :shift-report="shiftReport"
          :current-shift="currentShift"
          :shifts-history="shiftsHistory"
          @shift-updated="loadAllData"
          @notify="notify"
        />

        <!-- МОДУЛЬ 7: УПРАВЛЕНИЕ ПЕРСОНАЛОМ (ТОЛЬКО ДЛЯ УПРАВЛЯЮЩЕГО) -->
        <EmployeesTab
          v-if="activeTab === 'employees'"
          :users="users"
          :current-user="currentUser"
          @user-created="loadAllData"
          @user-dismissed="loadAllData"
          @notify="notify"
        />
      </main>
    </div>
  </div>
</template>
