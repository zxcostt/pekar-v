<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from './api'
import {
  Croissant, LogOut, ShoppingCart, Package, ChefHat, BarChart3,
  Plus, Minus, CreditCard, Banknote, RotateCcw, FileText, CheckCircle2, AlertCircle
} from 'lucide-vue-next'

// --- Состояние авторизации (UC 1) ---
const currentUser = ref(null)
const loginForm = ref({ login: '', password: '' })
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
const receipts = ref([])
const shiftReport = ref(null)

// --- Состояние Кассы (UC 8..14) ---
const cart = ref([])
const discountRate = ref(0)
const paymentType = ref('CASH')
const tenderedCash = ref(500)
const cardPan = ref('**** **** **** 4242')
const lastPrintedReceipt = ref(null)
const refundForm = ref({ fiscalDocNumber: '', reason: 'Возврат по обращению покупателя' })

// --- Состояние Пекаря (UC 6 + UC 7) ---
const produceForm = ref({ productId: 1, producedUnits: 10 })

// --- Состояние Кладовщика (UC 4 + UC 5) ---
const supplyForm = ref({ ingredientId: 1, amount: 10 })
const writeOffForm = ref({ ingredientId: 1, amount: 1, reason: 'Истечение срока годности / порча' })

// --- Состояние Управляющего: Создание техкарты (UC 2) ---
const newRecipe = ref({
  recipeTitle: '',
  yieldQuantity: 10,
  items: [{ ingredientId: 1, requiredQuantity: 0.5 }]
})

// Загрузка всех справочников и документов
async function loadAllData() {
  try {
    const [prodData, ingData, recData, ordData, rcptData, repData] = await Promise.all([
      api.getProducts(),
      api.getIngredients(),
      api.getRecipes(),
      api.getProductionOrders(),
      api.getReceipts(),
      api.getShiftReport()
    ])
    products.value = prodData || []
    ingredients.value = ingData || []
    recipes.value = recData || []
    productionOrders.value = ordData || []
    receipts.value = rcptData || []
    shiftReport.value = repData
  } catch (err) {
    notify(err.message, 'error')
  }
}

// --- UC 1: Вход в систему ---
async function handleLogin() {
  authError.value = ''
  try {
    const res = await api.login(loginForm.value.login, loginForm.value.password)
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
    loginForm.value.password = ''
  }
}

function quickLogin(login, password) {
  loginForm.value.login = login
  loginForm.value.password = password
  handleLogin()
}

function logout() {
  currentUser.value = null
  cart.value = []
  lastPrintedReceipt.value = null
}

// Доступ к модулям по ролям
function canAccess(tab) {
  if (!currentUser.value) return false
  const role = currentUser.value.role
  if (role === 'MANAGER') return true
  if (role === 'CASHIER') return ['pos', 'refunds', 'reports'].includes(tab)
  if (role === 'BAKER') return ['production', 'recipes'].includes(tab)
  if (role === 'STOREKEEPER') return ['warehouse'].includes(tab)
  return false
}

// --- Логика Кассы (UC 8..13) ---
function addToCart(product) {
  const existing = cart.value.find(item => item.productId === product.productId)
  const currentInCart = existing ? existing.quantity : 0
  if (currentInCart + 1 > product.currentQuantity) {
    return notify(`На витрине осталось всего ${product.currentQuantity} шт. «${product.name}»`, 'error')
  }
  if (existing) {
    existing.quantity++
  } else {
    cart.value.push({
      productId: product.productId,
      name: product.name,
      price: product.retailPrice,
      quantity: 1
    })
  }
}

function changeCartQty(item, delta) {
  item.quantity += delta
  if (item.quantity <= 0) {
    cart.value = cart.value.filter(i => i.productId !== item.productId)
  }
}

const cartSubtotal = computed(() =>
  cart.value.reduce((sum, i) => sum + i.price * i.quantity, 0)
)

const cartGrandTotal = computed(() => {
  const total = cartSubtotal.value * (1 - discountRate.value / 100)
  return Math.round(total * 100) / 100
})

const calculatedChange = computed(() => {
  if (paymentType.value !== 'CASH') return 0
  return Math.max(0, Math.round((tenderedCash.value - cartGrandTotal.value) * 100) / 100)
})

async function handleCheckout() {
  try {
    const payload = {
      cashierId: currentUser.value.id,
      discountRate: Number(discountRate.value),
      paymentType: paymentType.value,
      tenderedCash: paymentType.value === 'CASH' ? Number(tenderedCash.value) : 0,
      cardPan: paymentType.value === 'CARD' ? cardPan.value : '',
      items: cart.value.map(i => ({ productId: i.productId, quantity: i.quantity }))
    }
    const res = await api.checkout(payload)
    lastPrintedReceipt.value = res.receipt
    cart.value = []
    discountRate.value = 0
    notify(res.message)
    await loadAllData()
  } catch (err) {
    notify(err.message, 'error')
  }
}

// --- UC 14: Возврат чека ---
async function handleRefund(fiscalDoc) {
  const docToRefund = fiscalDoc || refundForm.value.fiscalDocNumber
  if (!docToRefund) return notify('Укажите фискальный номер чека', 'error')
  try {
    const res = await api.refund(docToRefund, refundForm.value.reason)
    notify(`${res.message} (${res.refundFiscalSlip})`)
    refundForm.value.fiscalDocNumber = ''
    await loadAllData()
  } catch (err) {
    notify(err.message, 'error')
  }
}

// --- UC 6 + UC 7: Выпуск продукции со списанием сырья по техкарте ---
async function handleProduce() {
  try {
    const res = await api.produceProduct(
      Number(produceForm.value.productId),
      currentUser.value.id,
      Number(produceForm.value.producedUnits)
    )
    notify(res.message)
    await loadAllData()
  } catch (err) {
    notify(err.message, 'error')
  }
}

// --- UC 4 + UC 5: Приход и списание сырья на складе ---
async function handleSupply() {
  try {
    await api.supplyIngredient(Number(supplyForm.value.ingredientId), Number(supplyForm.value.amount))
    notify('Поставка сырья успешно проведена, остаток увеличен')
    await loadAllData()
  } catch (err) {
    notify(err.message, 'error')
  }
}

async function handleWriteOff() {
  try {
    await api.writeOffIngredient(
      Number(writeOffForm.value.ingredientId),
      Number(writeOffForm.value.amount),
      writeOffForm.value.reason
    )
    notify('Акт списания сырья проведён')
    await loadAllData()
  } catch (err) {
    notify(err.message, 'error')
  }
}

// --- UC 2: Создание новой рецептуры (техкарты) ---
function addRecipeRow() {
  newRecipe.value.items.push({ ingredientId: ingredients.value[0]?.ingredientId || 1, requiredQuantity: 0.1 })
}

async function handleCreateRecipe() {
  try {
    await api.createRecipe({
      recipeTitle: newRecipe.value.recipeTitle,
      yieldQuantity: Number(newRecipe.value.yieldQuantity),
      items: newRecipe.value.items.map(i => ({
        ingredientId: Number(i.ingredientId),
        requiredQuantity: Number(i.requiredQuantity)
      }))
    })
    notify('Рецептура (техкарта) успешно сохранена в справочнике')
    newRecipe.value = { recipeTitle: '', yieldQuantity: 10, items: [{ ingredientId: 1, requiredQuantity: 0.5 }] }
    await loadAllData()
  } catch (err) {
    notify(err.message, 'error')
  }
}

// --- UC 3: Печать / Экспорт сменного отчёта в PDF ---
function exportShiftReportPdf() {
  window.print()
}
</script>

<template>
  <div class="min-h-screen bg-stone-100 text-stone-800">
    <!-- Всплывающее уведомление -->
    <div
      v-if="notification.message"
      :class="[
        'fixed top-4 right-4 z-50 flex items-center gap-2 px-5 py-3 rounded-xl shadow-lg text-white font-medium transition-all',
        notification.type === 'error' ? 'bg-red-600' : 'bg-emerald-600'
      ]"
    >
      <AlertCircle class="w-5 h-5" v-if="notification.type === 'error'"/>
      <CheckCircle2 class="w-5 h-5" v-else/>
      <span>{{ notification.message }}</span>
    </div>

    <!-- ЭКРАН 1: АВТОРИЗАЦИЯ (UC 1) -->
    <div v-if="!currentUser" class="min-h-screen flex items-center justify-center p-4">
      <div class="bg-white w-full max-w-md rounded-2xl shadow-xl p-8 border border-stone-200">
        <div class="flex flex-col items-center mb-6">
          <div class="w-16 h-16 bg-amber-100 text-amber-700 rounded-2xl flex items-center justify-center mb-3">
            <Croissant class="w-9 h-9"/>
          </div>
          <h1 class="text-2xl font-bold text-stone-900">АИС «Пекарь-В»</h1>
          <p class="text-sm text-stone-500">Пекарня «Восход» — Вход в систему</p>
        </div>

        <form @submit.prevent="handleLogin" class="space-y-4">
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

    <!-- ЭКРАН 2: РАБОЧЕЕ МЕСТО СОТРУДНИКА -->
    <div v-else class="min-h-screen flex flex-col">
      <!-- Верхняя шапка -->
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
            @click="activeTab = 'pos'"
            :class="[
              'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
              activeTab === 'pos' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
            ]"
          >
            <ShoppingCart class="w-4 h-4"/> Касса (Продажи)
          </button>

          <button
            v-if="canAccess('refunds')"
            @click="activeTab = 'refunds'"
            :class="[
              'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
              activeTab === 'refunds' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
            ]"
          >
            <RotateCcw class="w-4 h-4"/> Чеки и Возвраты
          </button>

          <button
            v-if="canAccess('production')"
            @click="activeTab = 'production'"
            :class="[
              'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
              activeTab === 'production' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
            ]"
          >
            <ChefHat class="w-4 h-4"/> Цех (Выпуск выпечки)
          </button>

          <button
            v-if="canAccess('warehouse')"
            @click="activeTab = 'warehouse'"
            :class="[
              'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
              activeTab === 'warehouse' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
            ]"
          >
            <Package class="w-4 h-4"/> Склад сырья
          </button>

          <button
            v-if="canAccess('recipes')"
            @click="activeTab = 'recipes'"
            :class="[
              'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition cursor-pointer',
              activeTab === 'recipes' ? 'bg-amber-600 text-white' : 'bg-stone-100 hover:bg-stone-200 text-stone-700'
            ]"
          >
            <FileText class="w-4 h-4"/> Техкарты (Рецептуры)
          </button>

          <button
            v-if="canAccess('reports')"
            @click="activeTab = 'reports'"
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
            @click="logout"
            class="p-2 rounded-xl bg-stone-100 hover:bg-red-50 hover:text-red-600 transition cursor-pointer"
            title="Выйти"
          >
            <LogOut class="w-5 h-5"/>
          </button>
        </div>
      </header>

      <!-- ОСНОВНОЙ КОНТЕНТ -->
      <main class="flex-1 p-6 max-w-7xl w-full mx-auto">
        <!-- МОДУЛЬ 1: КАССА ПРОДАЖ (UC 8, 9, 10, 11, 12, 13) -->
        <div v-if="activeTab === 'pos'" class="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <!-- Витрина готовой выпечки -->
          <div class="lg:col-span-2 space-y-4">
            <h2 class="text-lg font-bold">Витрина готовой продукции (выберите товар)</h2>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div
                v-for="p in products"
                :key="p.productId"
                @click="addToCart(p)"
                class="bg-white p-5 rounded-2xl border border-stone-200 shadow-sm hover:border-amber-500 hover:shadow transition cursor-pointer flex justify-between items-center"
              >
                <div>
                  <div class="font-bold text-base">{{ p.name }}</div>
                  <div class="text-xs text-stone-400 mt-0.5">Штрихкод: {{ p.barcode }}</div>
                  <div class="mt-2 inline-block px-2.5 py-0.5 rounded-full text-xs font-medium"
                    :class="p.currentQuantity > 0 ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700'">
                    Остаток на витрине: {{ p.currentQuantity }} шт.
                  </div>
                </div>
                <div class="text-xl font-bold text-amber-700">{{ p.retailPrice }} ₽</div>
              </div>
            </div>

            <!-- Отображение напечатанного фискального чека (UC 10) -->
            <div v-if="lastPrintedReceipt" class="bg-white p-6 rounded-2xl border-2 border-dashed border-stone-300 mt-6">
              <div class="text-center border-b border-stone-200 pb-3 mb-3">
                <div class="font-bold uppercase tracking-wider text-sm">Пекарня «Восход» — Кассовый чек</div>
                <div class="text-xs text-stone-500">Фискальный регистратор (Онлайн-касса)</div>
                <div class="text-xs font-mono mt-1 font-semibold text-amber-800">
                  {{ lastPrintedReceipt.fiscalDocNumber }}
                </div>
              </div>
              <div class="space-y-1 text-sm mb-3">
                <div v-for="item in lastPrintedReceipt.items" :key="item.receiptItemId" class="flex justify-between">
                  <span>{{ item.productName }} x {{ item.quantity }}</span>
                  <span class="font-medium">{{ item.lineTotal }} ₽</span>
                </div>
              </div>
              <div class="border-t border-stone-200 pt-2 text-sm space-y-1">
                <div class="flex justify-between text-stone-500">
                  <span>Скидка:</span>
                  <span>{{ lastPrintedReceipt.discountRate }}%</span>
                </div>
                <div class="flex justify-between font-bold text-base">
                  <span>ИТОГО:</span>
                  <span>{{ lastPrintedReceipt.grandTotal }} ₽</span>
                </div>
                <div v-if="lastPrintedReceipt.payment?.paymentType === 'CASH'" class="text-xs text-stone-600 pt-1">
                  Оплата НАЛИЧНЫМИ: внесено {{ lastPrintedReceipt.payment.tenderedAmount }} ₽ |
                  <b>Сдача: {{ lastPrintedReceipt.payment.changeAmount }} ₽</b>
                </div>
                <div v-else class="text-xs text-stone-600 pt-1">
                  Оплата КАРТОЙ ({{ lastPrintedReceipt.payment?.cardPanMasked }}) |
                  Код авторизации: <b>{{ lastPrintedReceipt.payment?.authorizationCode }}</b>
                </div>
              </div>
            </div>
          </div>

          <!-- Корзина чека и оплата -->
          <div class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm flex flex-col justify-between">
            <div>
              <div class="flex justify-between items-center mb-4">
                <h3 class="font-bold text-lg">Текущий чек</h3>
                <button
                  v-if="cart.length"
                  @click="cart = []"
                  class="text-xs text-red-600 hover:underline cursor-pointer"
                >
                  Очистить чек
                </button>
              </div>

              <div v-if="!cart.length" class="text-center py-10 text-stone-400 text-sm">
                Корзина чека пуста.<br />Нажмите на выпечку слева.
              </div>

              <div v-else class="space-y-3 mb-6">
                <div
                  v-for="item in cart"
                  :key="item.productId"
                  class="flex items-center justify-between bg-stone-50 p-3 rounded-xl"
                >
                  <div>
                    <div class="font-medium text-sm">{{ item.name }}</div>
                    <div class="text-xs text-stone-500">{{ item.price }} ₽ × {{ item.quantity }} шт.</div>
                  </div>
                  <div class="flex items-center gap-2">
                    <button @click="changeCartQty(item, -1)" class="p-1 bg-white rounded border cursor-pointer">
                      <Minus class="w-3.5 h-3.5"/>
                    </button>
                    <span class="font-semibold text-sm w-6 text-center">{{ item.quantity }}</span>
                    <button @click="changeCartQty(item, 1)" class="p-1 bg-white rounded border cursor-pointer">
                      <Plus class="w-3.5 h-3.5"/>
                    </button>
                  </div>
                </div>
              </div>

              <!-- UC 9: Применение скидки -->
              <div class="border-t border-stone-200 pt-4 space-y-4">
                <div>
                  <label class="block text-xs font-semibold uppercase text-stone-500 mb-1">
                    Скидка по акции (UC 9)
                  </label>
                  <div class="flex gap-2">
                    <button
                      v-for="rate in [0, 10, 20]"
                      :key="rate"
                      @click="discountRate = rate"
                      :class="[
                        'flex-1 py-1.5 rounded-lg text-xs font-semibold border cursor-pointer',
                        discountRate === rate
                          ? 'bg-amber-100 border-amber-500 text-amber-800'
                          : 'bg-stone-50 border-stone-200 text-stone-600'
                      ]"
                    >
                      {{ rate === 0 ? 'Без скидки' : `Скидка ${rate}%` }}
                    </button>
                  </div>
                </div>

                <!-- UC 11..13: Выбор способа оплаты -->
                <div>
                  <label class="block text-xs font-semibold uppercase text-stone-500 mb-1">
                    Способ оплаты (UC 11–13)
                  </label>
                  <div class="grid grid-cols-2 gap-2">
                    <button
                      @click="paymentType = 'CASH'"
                      :class="[
                        'flex items-center justify-center gap-2 py-2.5 rounded-xl text-sm font-medium border cursor-pointer',
                        paymentType === 'CASH'
                          ? 'bg-emerald-50 border-emerald-500 text-emerald-800'
                          : 'bg-stone-50 border-stone-200'
                      ]"
                    >
                      <Banknote class="w-4 h-4"/> Наличные
                    </button>
                    <button
                      @click="paymentType = 'CARD'"
                      :class="[
                        'flex items-center justify-center gap-2 py-2.5 rounded-xl text-sm font-medium border cursor-pointer',
                        paymentType === 'CARD'
                          ? 'bg-blue-50 border-blue-500 text-blue-800'
                          : 'bg-stone-50 border-stone-200'
                      ]"
                    >
                      <CreditCard class="w-4 h-4"/> Карта
                    </button>
                  </div>
                </div>

                <!-- Поле внесения наличных и расчёт сдачи (UC 12) -->
                <div v-if="paymentType === 'CASH'" class="bg-stone-50 p-3 rounded-xl space-y-2">
                  <div class="flex justify-between items-center">
                    <span class="text-xs font-medium text-stone-600">Внесено покупателем (₽):</span>
                    <input
                      v-model.number="tenderedCash"
                      type="number"
                      min="0"
                      class="w-28 px-2 py-1 text-right font-bold bg-white border border-stone-300 rounded-lg"
                    />
                  </div>
                  <div class="flex justify-between items-center text-sm">
                    <span class="text-stone-600">Сдача клиенту:</span>
                    <span class="font-bold text-emerald-700 text-base">{{ calculatedChange }} ₽</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- Итог и кнопка фискализации -->
            <div class="border-t border-stone-200 pt-4 mt-4">
              <div class="flex justify-between text-sm text-stone-500 mb-1">
                <span>Подытог:</span>
                <span>{{ cartSubtotal }} ₽</span>
              </div>
              <div class="flex justify-between text-xl font-bold mb-4">
                <span>К оплате:</span>
                <span class="text-amber-700">{{ cartGrandTotal }} ₽</span>
              </div>
              <button
                @click="handleCheckout"
                :disabled="!cart.length"
                class="w-full py-3.5 bg-amber-600 hover:bg-amber-700 disabled:bg-stone-300 text-white font-bold rounded-xl shadow transition cursor-pointer"
              >
                Пробить фискальный чек
              </button>
            </div>
          </div>
        </div>

        <!-- МОДУЛЬ 2: ИСТОРИЯ ЧЕКОВ И ВОЗВРАТ ТОВАРА (UC 14) -->
        <div v-if="activeTab === 'refunds'" class="space-y-6">
          <div class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
            <h2 class="text-lg font-bold mb-4">Оформить возврат по номеру фискального документа (UC 14)</h2>
            <form @submit.prevent="handleRefund()" class="flex flex-wrap gap-3">
              <input
                v-model="refundForm.fiscalDocNumber"
                type="text"
                placeholder="Номер ФД (например: ФД-2026...)"
                class="flex-1 min-w-64 px-4 py-2.5 rounded-xl border border-stone-300"
              />
              <input
                v-model="refundForm.reason"
                type="text"
                placeholder="Причина возврата"
                class="flex-1 min-w-64 px-4 py-2.5 rounded-xl border border-stone-300"
              />
              <button
                type="submit"
                class="px-6 py-2.5 bg-red-600 hover:bg-red-700 text-white font-semibold rounded-xl cursor-pointer"
              >
                Выполнить возврат
              </button>
            </form>
          </div>

          <div class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
            <h3 class="font-bold text-lg mb-4">Журнал фискальных чеков</h3>
            <div class="overflow-x-auto">
              <table class="w-full text-sm text-left">
                <thead class="text-xs uppercase bg-stone-50 text-stone-500">
                  <tr>
                    <th class="p-3">Фискальный №</th>
                    <th class="p-3">Кассир</th>
                    <th class="p-3">Позиции</th>
                    <th class="p-3">Скидка</th>
                    <th class="p-3">Сумма</th>
                    <th class="p-3">Оплата</th>
                    <th class="p-3">Статус</th>
                    <th class="p-3">Действие</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-stone-200">
                  <tr v-for="r in receipts" :key="r.receiptId">
                    <td class="p-3 font-mono font-semibold">{{ r.fiscalDocNumber }}</td>
                    <td class="p-3">{{ r.cashierName }}</td>
                    <td class="p-3">
                      <div v-for="i in r.items" :key="i.receiptItemId" class="text-xs">
                        {{ i.productName }} × {{ i.quantity }} шт.
                      </div>
                    </td>
                    <td class="p-3">{{ r.discountRate }}%</td>
                    <td class="p-3 font-bold">{{ r.grandTotal }} ₽</td>
                    <td class="p-3">{{ r.payment?.paymentType === 'CASH' ? 'Наличные' : 'Карта' }}</td>
                    <td class="p-3">
                      <span
                        :class="[
                          'px-2.5 py-1 rounded-full text-xs font-semibold',
                          r.isRefunded ? 'bg-red-100 text-red-700' : 'bg-emerald-100 text-emerald-700'
                        ]"
                      >
                        {{ r.isRefunded ? 'ВОЗВРАЩЁН' : 'ПРОВЕДЁН' }}
                      </span>
                    </td>
                    <td class="p-3">
                      <button
                        v-if="!r.isRefunded"
                        @click="handleRefund(r.fiscalDocNumber)"
                        class="text-xs px-3 py-1.5 rounded-lg bg-red-50 hover:bg-red-100 text-red-700 font-medium cursor-pointer"
                      >
                        Оформить возврат
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <!-- МОДУЛЬ 3: ПРОИЗВОДСТВЕННЫЙ ЦЕХ — ВЫПУСК ВЫПЕЧКИ (UC 6 + UC 7) -->
        <div v-if="activeTab === 'production'" class="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
            <h2 class="text-lg font-bold mb-2">Акт выпуска продукции (UC 6 + UC 7)</h2>
            <p class="text-xs text-stone-500 mb-4">
              При подтверждении выпуска система автоматически списывает сырьё со склада по техкарте и увеличивает остаток выпечки на витрине.
            </p>
            <form @submit.prevent="handleProduce" class="space-y-4">
              <div>
                <label class="block text-sm font-medium mb-1">Готовое изделие</label>
                <select
                  v-model="produceForm.productId"
                  class="w-full px-3 py-2.5 rounded-xl border border-stone-300 bg-white"
                >
                  <option v-for="p in products" :key="p.productId" :value="p.productId">
                    {{ p.name }} (на витрине: {{ p.currentQuantity }} шт.)
                  </option>
                </select>
              </div>
              <div>
                <label class="block text-sm font-medium mb-1">Количество выпеченных единиц (шт.)</label>
                <input
                  v-model.number="produceForm.producedUnits"
                  type="number"
                  min="1"
                  class="w-full px-4 py-2.5 rounded-xl border border-stone-300"
                />
              </div>
              <button
                type="submit"
                class="w-full py-3 bg-amber-600 hover:bg-amber-700 text-white font-bold rounded-xl cursor-pointer"
              >
                Подтвердить выпуск партии
              </button>
            </form>
          </div>

          <div class="lg:col-span-2 bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
            <h3 class="font-bold text-lg mb-4">Журнал производственных нарядов</h3>
            <table class="w-full text-sm text-left">
              <thead class="text-xs uppercase bg-stone-50 text-stone-500">
                <tr>
                  <th class="p-3">№ Наряда</th>
                  <th class="p-3">Техкарта</th>
                  <th class="p-3">Пекарь</th>
                  <th class="p-3">Выпущено</th>
                  <th class="p-3">Статус</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-stone-200">
                <tr v-for="o in productionOrders" :key="o.orderId">
                  <td class="p-3 font-bold">#{{ o.orderId }}</td>
                  <td class="p-3">{{ o.recipeTitle }}</td>
                  <td class="p-3">{{ o.bakerName }}</td>
                  <td class="p-3 font-semibold">{{ o.producedUnits }} шт.</td>
                  <td class="p-3">
                    <span class="px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-700">
                      {{ o.orderStatus }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- МОДУЛЬ 4: СКЛАД СЫРЬЯ (UC 4 + UC 5) -->
        <div v-if="activeTab === 'warehouse'" class="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div class="space-y-6">
            <!-- UC 4: Приход сырья -->
            <div class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
              <h3 class="font-bold text-base mb-3 text-emerald-800">➕ Оформить поступление сырья (UC 4)</h3>
              <form @submit.prevent="handleSupply" class="space-y-3">
                <select v-model="supplyForm.ingredientId" class="w-full px-3 py-2 rounded-xl border border-stone-300">
                  <option v-for="ing in ingredients" :key="ing.ingredientId" :value="ing.ingredientId">
                    {{ ing.name }} ({{ ing.unit }})
                  </option>
                </select>
                <input
                  v-model.number="supplyForm.amount"
                  type="number"
                  step="0.1"
                  placeholder="Принятый объём"
                  class="w-full px-3 py-2 rounded-xl border border-stone-300"
                />
                <button
                  type="submit"
                  class="w-full py-2.5 bg-emerald-600 hover:bg-emerald-700 text-white font-semibold rounded-xl cursor-pointer"
                >
                  Провести приходную накладную
                </button>
              </form>
            </div>

            <!-- UC 5: Списание сырья -->
            <div class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
              <h3 class="font-bold text-base mb-3 text-red-800">➖ Акт списания ТМЦ / порчи (UC 5)</h3>
              <form @submit.prevent="handleWriteOff" class="space-y-3">
                <select v-model="writeOffForm.ingredientId" class="w-full px-3 py-2 rounded-xl border border-stone-300">
                  <option v-for="ing in ingredients" :key="ing.ingredientId" :value="ing.ingredientId">
                    {{ ing.name }} (остаток: {{ ing.currentStock }} {{ ing.unit }})
                  </option>
                </select>
                <input
                  v-model.number="writeOffForm.amount"
                  type="number"
                  step="0.1"
                  placeholder="Количество к списанию"
                  class="w-full px-3 py-2 rounded-xl border border-stone-300"
                />
                <input
                  v-model="writeOffForm.reason"
                  type="text"
                  placeholder="Причина списания"
                  class="w-full px-3 py-2 rounded-xl border border-stone-300"
                />
                <button
                  type="submit"
                  class="w-full py-2.5 bg-red-600 hover:bg-red-700 text-white font-semibold rounded-xl cursor-pointer"
                >
                  Провести списание сырья
                </button>
              </form>
            </div>
          </div>

          <!-- Таблица остатков сырья -->
          <div class="lg:col-span-2 bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
            <h2 class="font-bold text-lg mb-4">Текущие складские остатки сырья</h2>
            <table class="w-full text-sm text-left">
              <thead class="text-xs uppercase bg-stone-50 text-stone-500">
                <tr>
                  <th class="p-3">ID</th>
                  <th class="p-3">Наименование сырья</th>
                  <th class="p-3">Текущий остаток</th>
                  <th class="p-3">Ед. изм.</th>
                  <th class="p-3">Закупочная цена</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-stone-200">
                <tr v-for="ing in ingredients" :key="ing.ingredientId">
                  <td class="p-3">#{{ ing.ingredientId }}</td>
                  <td class="p-3 font-medium">{{ ing.name }}</td>
                  <td class="p-3 font-bold text-base">{{ ing.currentStock.toFixed(3) }}</td>
                  <td class="p-3">{{ ing.unit }}</td>
                  <td class="p-3">{{ ing.purchasePrice }} ₽</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- МОДУЛЬ 5: СПРАВОЧНИК РЕЦЕПТУР / ТЕХКАРТ (UC 2) -->
        <div v-if="activeTab === 'recipes'" class="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div v-if="currentUser.role === 'MANAGER'" class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
            <h3 class="font-bold text-lg mb-3">Добавить техкарту (UC 2)</h3>
            <form @submit.prevent="handleCreateRecipe" class="space-y-3">
              <div>
                <label class="block text-xs font-medium text-stone-600 mb-1">Название техкарты</label>
                <input
                  v-model="newRecipe.recipeTitle"
                  type="text"
                  required
                  placeholder="Например: ТК №2 Багет французский"
                  class="w-full px-3 py-2 rounded-xl border border-stone-300"
                />
              </div>
              <div>
                <label class="block text-xs font-medium text-stone-600 mb-1">Выход партии (шт.)</label>
                <input
                  v-model.number="newRecipe.yieldQuantity"
                  type="number"
                  min="1"
                  class="w-full px-3 py-2 rounded-xl border border-stone-300"
                />
              </div>

              <div class="space-y-2 pt-2">
                <div class="flex justify-between items-center">
                  <span class="text-xs font-semibold uppercase text-stone-500">Нормы закладки сырья</span>
                  <button
                    type="button"
                    @click="addRecipeRow"
                    class="text-xs text-amber-700 font-semibold hover:underline cursor-pointer"
                  >
                    + Добавить ингредиент
                  </button>
                </div>
                <div v-for="(row, idx) in newRecipe.items" :key="idx" class="flex gap-2">
                  <select v-model="row.ingredientId" class="flex-1 px-2 py-1.5 text-sm rounded-lg border">
                    <option v-for="ing in ingredients" :key="ing.ingredientId" :value="ing.ingredientId">
                      {{ ing.name }} ({{ ing.unit }})
                    </option>
                  </select>
                  <input
                    v-model.number="row.requiredQuantity"
                    type="number"
                    step="0.01"
                    class="w-24 px-2 py-1.5 text-sm rounded-lg border"
                  />
                </div>
              </div>

              <button
                type="submit"
                class="w-full py-2.5 bg-amber-600 hover:bg-amber-700 text-white font-semibold rounded-xl cursor-pointer mt-2"
              >
                Сохранить рецептуру
              </button>
            </form>
          </div>

          <div :class="currentUser.role === 'MANAGER' ? 'lg:col-span-2' : 'lg:col-span-3'" class="space-y-4">
            <h2 class="font-bold text-lg">Действующие технологические карты</h2>
            <div v-for="rec in recipes" :key="rec.recipeId" class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
              <div class="flex justify-between items-center border-b border-stone-200 pb-3 mb-3">
                <div>
                  <h3 class="font-bold text-base">{{ rec.recipeTitle }}</h3>
                  <p class="text-xs text-stone-500">Выход готовых изделий: {{ rec.yieldQuantity }} шт.</p>
                </div>
                <div class="text-right">
                  <span class="text-xs text-stone-500">Расчётная себестоимость партии:</span>
                  <div class="font-bold text-lg text-amber-700">{{ rec.totalCost.toFixed(2) }} ₽</div>
                </div>
              </div>
              <table class="w-full text-sm">
                <thead class="text-xs text-stone-400 uppercase text-left">
                  <tr>
                    <th class="pb-2">Сырьё</th>
                    <th class="pb-2">Норма закладки</th>
                    <th class="pb-2">Цена за ед.</th>
                    <th class="pb-2">Стоимость в партии</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-stone-100">
                  <tr v-for="item in rec.items" :key="item.recipeItemId">
                    <td class="py-1.5 font-medium">{{ item.ingredientName }}</td>
                    <td class="py-1.5">{{ item.requiredQuantity }} {{ item.unit }}</td>
                    <td class="py-1.5">{{ item.purchasePrice }} ₽</td>
                    <td class="py-1.5 font-semibold">{{ (item.requiredQuantity * item.purchasePrice).toFixed(2) }} ₽</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <!-- МОДУЛЬ 6: СМЕННЫЙ ОТЧЁТ (UC 3) -->
        <div v-if="activeTab === 'reports' && shiftReport" class="space-y-6">
          <div class="bg-white p-8 rounded-2xl border border-stone-200 shadow-sm">
            <div class="flex justify-between items-start mb-6">
              <div>
                <h2 class="text-xl font-bold">Сводный сменный отчёт (ShiftReportDTO — UC 3)</h2>
                <p class="text-sm text-stone-500">
                  Пекарня «Восход» • Смена №{{ shiftReport.shiftId }}
                </p>
              </div>
              <button
                @click="exportShiftReportPdf"
                class="px-5 py-2.5 bg-amber-600 hover:bg-amber-700 text-white font-semibold rounded-xl shadow cursor-pointer"
              >
                🖨️ Экспорт в PDF / Печать
              </button>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
              <div class="p-4 rounded-xl bg-amber-50 border border-amber-200">
                <div class="text-xs text-amber-800 uppercase font-semibold">Общая выручка</div>
                <div class="text-2xl font-bold text-amber-900 mt-1">{{ shiftReport.totalRevenue.toFixed(2) }} ₽</div>
              </div>
              <div class="p-4 rounded-xl bg-emerald-50 border border-emerald-200">
                <div class="text-xs text-emerald-800 uppercase font-semibold">Наличными (CASH)</div>
                <div class="text-2xl font-bold text-emerald-900 mt-1">{{ shiftReport.cashSubtotal.toFixed(2) }} ₽</div>
              </div>
              <div class="p-4 rounded-xl bg-blue-50 border border-blue-200">
                <div class="text-xs text-blue-800 uppercase font-semibold">Картами (CARD)</div>
                <div class="text-2xl font-bold text-blue-900 mt-1">{{ shiftReport.cardSubtotal.toFixed(2) }} ₽</div>
              </div>
              <div class="p-4 rounded-xl bg-stone-50 border border-stone-200">
                <div class="text-xs text-stone-600 uppercase font-semibold">Проведено чеков</div>
                <div class="text-2xl font-bold text-stone-900 mt-1">{{ shiftReport.receiptCount }}</div>
              </div>
              <div class="p-4 rounded-xl bg-red-50 border border-red-200">
                <div class="text-xs text-red-800 uppercase font-semibold">Возвратов</div>
                <div class="text-2xl font-bold text-red-900 mt-1">{{ shiftReport.refundedCount }}</div>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  </div>
</template>