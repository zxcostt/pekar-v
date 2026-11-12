<script setup>
import { ref, computed } from 'vue'
import { api } from '../../api'
import {
  Minus,
  Plus,
  Banknote,
  CreditCard,
  PackageX,
  Trash2,
  X,
  Lock,
  Unlock,
  FileText,
  Printer,
  AlertTriangle,
  CheckCircle2
} from 'lucide-vue-next'

const props = defineProps({
  products: {
    type: Array,
    default: () => []
  },
  currentUser: {
    type: Object,
    required: true
  },
  currentShift: {
    type: Object,
    default: null
  },
  lastPrintedReceipt: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['checkout-success', 'writeoff-success', 'shift-updated', 'notify'])

// Состояние Кассы (UC 8..14)
const cart = ref([])
const discountRate = ref(0)
const paymentType = ref('CASH')
const tenderedCash = ref(500)
const cardPan = ref('**** **** **** 4242')

function addToCart(product) {
  if (!props.currentShift) {
    return emit('notify', 'Кассовая смена закрыта. Сначала откройте смену!', 'error')
  }
  const existing = cart.value.find(item => item.productId === product.productId)
  const currentInCart = existing ? existing.quantity : 0
  if (currentInCart + 1 > product.currentQuantity) {
    return emit('notify', `На витрине осталось всего ${product.currentQuantity} шт. «${product.name}»`, 'error')
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
  if (!props.currentShift) {
    return emit('notify', 'Кассовая смена закрыта. Откройте смену для проведения продаж', 'error')
  }
  try {
    const payload = {
      cashierId: props.currentUser.id,
      discountRate: Number(discountRate.value),
      paymentType: paymentType.value,
      tenderedCash: paymentType.value === 'CASH' ? Number(tenderedCash.value) : 0,
      cardPan: paymentType.value === 'CARD' ? cardPan.value : '',
      items: cart.value.map(i => ({ productId: i.productId, quantity: i.quantity }))
    }
    const res = await api.checkout(payload)
    cart.value = []
    discountRate.value = 0
    emit('checkout-success', res)
  } catch (err) {
    emit('notify', err.message, 'error')
  }
}

// ==========================================
// 1. УПРАВЛЕНИЕ КАССОВЫМИ СМЕНАМИ
// ==========================================

// Открытие смены
const showOpenShiftModal = ref(false)
const openShiftCash = ref(3000)
const isOpenShiftLoading = ref(false)

async function handleOpenShift() {
  if (openShiftCash.value < 0) {
    return emit('notify', 'Сумма начального размена не может быть отрицательной', 'error')
  }
  isOpenShiftLoading.value = true
  try {
    const res = await api.openShift(props.currentUser.id, Number(openShiftCash.value))
    emit('notify', res.message || 'Кассовая смена открыта')
    emit('shift-updated')
    showOpenShiftModal.value = false
  } catch (err) {
    emit('notify', err.message, 'error')
  } finally {
    isOpenShiftLoading.value = false
  }
}

// X-отчёт (без гашения)
const showXReportModal = ref(false)
const xReportData = ref(null)
const isXReportLoading = ref(false)

async function openXReport() {
  isXReportLoading.value = true
  showXReportModal.value = true
  try {
    const data = await api.getXReport()
    xReportData.value = data
  } catch (err) {
    emit('notify', err.message, 'error')
    showXReportModal.value = false
  } finally {
    isXReportLoading.value = false
  }
}

// Закрытие смены (Z-отчёт и инкассация)
const showCloseShiftModal = ref(false)
const actualCashInDrawer = ref(0)
const encashmentSum = ref(0)
const isClosingShift = ref(false)
const closeShiftResult = ref(null)

async function openCloseShiftModal() {
  try {
    const data = await api.getXReport()
    xReportData.value = data
    // По умолчанию предлагаем фактическую сумму равной расчетной
    actualCashInDrawer.value = data.expectedCashInDrawer
    // По умолчанию предлагаем сдать выручку наличными
    encashmentSum.value = Math.max(0, data.cashRevenue)
    closeShiftResult.value = null
    showCloseShiftModal.value = true
  } catch (err) {
    emit('notify', err.message, 'error')
  }
}

const discrepancy = computed(() => {
  if (!xReportData.value) return 0
  return Math.round((actualCashInDrawer.value - xReportData.value.expectedCashInDrawer) * 100) / 100
})

async function handleCloseShift() {
  if (actualCashInDrawer.value < 0) {
    return emit('notify', 'Фактическая сумма в кассе не может быть отрицательной', 'error')
  }
  if (encashmentSum.value < 0) {
    return emit('notify', 'Сумма инкассации не может быть отрицательной', 'error')
  }
  if (encashmentSum.value > actualCashInDrawer.value) {
    return emit('notify', 'Нельзя инкассировать больше, чем фактически есть в кассе', 'error')
  }

  isClosingShift.value = true
  try {
    const res = await api.closeShift(
      props.currentUser.id,
      Number(actualCashInDrawer.value),
      Number(encashmentSum.value)
    )
    closeShiftResult.value = res
    emit('notify', res.message)
    emit('shift-updated')
    cart.value = []
  } catch (err) {
    emit('notify', err.message, 'error')
  } finally {
    isClosingShift.value = false
  }
}

// ==========================================
// 2. МОДАЛКА СПИСАНИЯ ВЫПЕЧКИ
// ==========================================
const showWriteOffModal = ref(false)
const posWriteOffForm = ref({
  productId: null,
  quantity: 1,
  reason: 'Истечение срока годности (вечерний остаток)'
})
const isWritingOffPos = ref(false)

const predefinedReasons = [
  'Истечение срока годности (вечерний остаток)',
  'Брак выпечки (подгорело / пересушено)',
  'Потеря товарного вида / деформация',
  'Дегустация / витринный образец'
]

function openWriteOffModal(product) {
  posWriteOffForm.value.productId = product?.productId || props.products[0]?.productId || null
  posWriteOffForm.value.quantity = 1
  posWriteOffForm.value.reason = 'Истечение срока годности (вечерний остаток)'
  showWriteOffModal.value = true
}

const selectedPosWriteOffProduct = computed(() => {
  return props.products.find(p => p.productId === posWriteOffForm.value.productId)
})

async function submitPosWriteOff() {
  const prod = selectedPosWriteOffProduct.value
  if (!prod) return
  if (posWriteOffForm.value.quantity <= 0) {
    return emit('notify', 'Количество должно быть больше 0', 'error')
  }
  if (posWriteOffForm.value.quantity > prod.currentQuantity) {
    return emit('notify', `На витрине всего ${prod.currentQuantity} шт.`, 'error')
  }

  isWritingOffPos.value = true
  try {
    const res = await api.writeOffProduct(
      prod.productId,
      props.currentUser.id,
      Number(posWriteOffForm.value.quantity),
      posWriteOffForm.value.reason.trim()
    )
    emit('notify', res.message || 'Акт списания выпечки проведён')
    emit('writeoff-success')
    showWriteOffModal.value = false
  } catch (err) {
    emit('notify', err.message, 'error')
  } finally {
    isWritingOffPos.value = false
  }
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div class="space-y-4">
    <!-- ПАНЕЛЬ КАССОВОЙ СМЕНЫ -->
    <div
      v-if="!currentShift"
      class="bg-amber-50 border border-amber-300 p-4 rounded-2xl flex flex-wrap items-center justify-between gap-3 shadow-xs"
    >
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-xl bg-amber-200 text-amber-800 flex items-center justify-center">
          <Lock class="w-5 h-5" />
        </div>
        <div>
          <div class="font-bold text-amber-900 text-sm sm:text-base">Кассовая смена закрыта</div>
          <div class="text-xs text-amber-700">
            Для фискализации чеков и продаж необходимо внести размен и открыть кассовую смену.
          </div>
        </div>
      </div>
      <button
        @click="showOpenShiftModal = true"
        class="px-5 py-2.5 bg-emerald-600 hover:bg-emerald-700 text-white font-bold rounded-xl text-sm shadow cursor-pointer transition flex items-center gap-2"
      >
        <Unlock class="w-4 h-4" />
        <span>Открыть смену</span>
      </button>
    </div>

    <div
      v-else
      class="bg-white border border-stone-200 p-4 rounded-2xl flex flex-wrap items-center justify-between gap-4 shadow-xs"
    >
      <div class="flex items-center gap-3">
        <div class="w-3 h-3 rounded-full bg-emerald-500 animate-pulse"></div>
        <div>
          <div class="flex items-center gap-2">
            <span class="font-bold text-stone-900 text-sm sm:text-base">Кассовая смена №{{ currentShift.shiftId }}</span>
            <span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800">ОТКРЫТА</span>
          </div>
          <div class="text-xs text-stone-500 mt-0.5">
            Кассир: <span class="font-medium text-stone-700">{{ currentShift.cashierName }}</span> •
            Открыта: <span class="font-medium text-stone-700">{{ formatDate(currentShift.openedAt) }}</span> •
            Размен: <span class="font-medium text-stone-700">{{ currentShift.initialCash.toFixed(2) }} ₽</span>
          </div>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <button
          @click="openXReport"
          class="px-3.5 py-2 bg-stone-100 hover:bg-stone-200 text-stone-700 font-semibold rounded-xl text-xs sm:text-sm cursor-pointer transition flex items-center gap-1.5"
        >
          <FileText class="w-4 h-4 text-stone-600" />
          <span>X-отчёт</span>
        </button>
        <button
          @click="openCloseShiftModal"
          class="px-3.5 py-2 bg-rose-50 hover:bg-rose-100 text-rose-700 border border-rose-200 font-semibold rounded-xl text-xs sm:text-sm cursor-pointer transition flex items-center gap-1.5"
        >
          <Lock class="w-4 h-4 text-rose-600" />
          <span>Закрыть смену (Z-отчёт)</span>
        </button>
      </div>
    </div>

    <!-- ОСНОВНАЯ ЧАСТЬ КАССЫ: ВИТРИНА И КОРЗИНА -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Витрина готовой выпечки -->
      <div class="lg:col-span-2 space-y-4">
        <div class="flex justify-between items-center">
          <h2 class="text-lg font-bold">Витрина готовой продукции</h2>
          <button
            @click="openWriteOffModal()"
            class="flex items-center gap-1.5 px-3 py-1.5 bg-rose-50 hover:bg-rose-100 text-rose-700 border border-rose-200 rounded-xl text-xs font-semibold cursor-pointer transition shadow-xs"
          >
            <PackageX class="w-4 h-4 text-rose-600" />
            <span>Списать с витрины</span>
          </button>
        </div>

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
              <div
                class="mt-2 inline-block px-2.5 py-0.5 rounded-full text-xs font-medium"
                :class="p.currentQuantity > 0 ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700'"
              >
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
              Очистить
            </button>
          </div>

          <!-- Список позиций в чеке -->
          <div v-if="cart.length" class="space-y-3 mb-6 max-h-64 overflow-y-auto pr-1">
            <div
              v-for="item in cart"
              :key="item.productId"
              class="flex items-center justify-between text-sm py-1 border-b border-stone-100"
            >
              <div class="flex-1 mr-2">
                <div class="font-medium truncate">{{ item.name }}</div>
                <div class="text-xs text-stone-500">{{ item.price }} ₽ / шт.</div>
              </div>
              <div class="flex items-center gap-2">
                <button
                  @click="changeCartQty(item, -1)"
                  class="w-6 h-6 rounded-lg bg-stone-100 hover:bg-stone-200 flex items-center justify-center cursor-pointer"
                >
                  <Minus class="w-3.5 h-3.5"/>
                </button>
                <span class="w-6 text-center font-bold text-sm">{{ item.quantity }}</span>
                <button
                  @click="changeCartQty(item, 1)"
                  class="w-6 h-6 rounded-lg bg-stone-100 hover:bg-stone-200 flex items-center justify-center cursor-pointer"
                >
                  <Plus class="w-3.5 h-3.5"/>
                </button>
              </div>
              <div class="w-16 text-right font-bold">{{ item.price * item.quantity }} ₽</div>
            </div>
          </div>
          <div v-else class="text-center py-10 text-stone-400 text-sm">
            Корзина пуста. Выберите выпечку слева для добавления в чек.
          </div>

          <!-- Настройки скидки и оплаты -->
          <div v-if="cart.length" class="space-y-4 pt-2 border-t border-stone-100">
            <div>
              <label class="block text-xs font-semibold text-stone-600 mb-1">
                Скидка по акции / карте: {{ discountRate }}%
              </label>
              <input
                v-model.number="discountRate"
                type="range"
                min="0"
                max="50"
                step="5"
                class="w-full accent-amber-600 cursor-pointer"
              />
            </div>

            <div>
              <label class="block text-xs font-semibold text-stone-600 mb-1">Способ оплаты</label>
              <div class="grid grid-cols-2 gap-2">
                <button
                  @click="paymentType = 'CASH'"
                  :class="[
                    'py-2 px-3 rounded-xl border text-xs font-bold flex items-center justify-center gap-2 transition cursor-pointer',
                    paymentType === 'CASH'
                      ? 'border-amber-600 bg-amber-50 text-amber-800'
                      : 'border-stone-200 hover:bg-stone-50 text-stone-700'
                  ]"
                >
                  <Banknote class="w-4 h-4"/> Наличные
                </button>
                <button
                  @click="paymentType = 'CARD'"
                  :class="[
                    'py-2 px-3 rounded-xl border text-xs font-bold flex items-center justify-center gap-2 transition cursor-pointer',
                    paymentType === 'CARD'
                      ? 'border-amber-600 bg-amber-50 text-amber-800'
                      : 'border-stone-200 hover:bg-stone-50 text-stone-700'
                  ]"
                >
                  <CreditCard class="w-4 h-4"/> Картой
                </button>
              </div>
            </div>

            <div v-if="paymentType === 'CASH'" class="p-3 bg-stone-50 rounded-xl space-y-2 border border-stone-200">
              <div class="flex justify-between items-center text-xs">
                <span class="text-stone-600 font-medium">Внесено клиентом (₽):</span>
                <input
                  v-model.number="tenderedCash"
                  type="number"
                  step="10"
                  class="w-24 px-2 py-1 rounded border border-stone-300 text-right font-bold bg-white"
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
            :disabled="!cart.length || !currentShift"
            :class="[
              'w-full py-3.5 font-bold rounded-xl shadow transition cursor-pointer',
              !currentShift
                ? 'bg-stone-200 text-stone-400 cursor-not-allowed'
                : !cart.length
                ? 'bg-stone-300 text-white cursor-not-allowed'
                : 'bg-amber-600 hover:bg-amber-700 text-white'
            ]"
          >
            {{ !currentShift ? 'Смена закрыта (откройте смену)' : 'Пробить фискальный чек' }}
          </button>
        </div>
      </div>
    </div>

    <!-- МОДАЛЬНОЕ ОКНО 1: ОТКРЫТИЕ КАССОВОЙ СМЕНЫ -->
    <div
      v-if="showOpenShiftModal"
      class="fixed inset-0 bg-stone-900/60 backdrop-blur-xs flex items-center justify-center p-4 z-50"
    >
      <div class="bg-white rounded-2xl max-w-md w-full p-6 shadow-2xl border border-stone-200">
        <div class="flex justify-between items-center mb-4 pb-3 border-b border-stone-100">
          <div class="flex items-center gap-2 text-emerald-700 font-bold text-lg">
            <Unlock class="w-5 h-5" />
            <span>Открытие кассовой смены</span>
          </div>
          <button @click="showOpenShiftModal = false" class="text-stone-400 hover:text-stone-600 p-1">
            <X class="w-5 h-5" />
          </button>
        </div>

        <form @submit.prevent="handleOpenShift" class="space-y-4">
          <div class="bg-stone-50 p-3.5 rounded-xl border border-stone-200 text-xs text-stone-600 space-y-1">
            <div>Касса: <b>Фискальный регистратор №1</b></div>
            <div>Ответственный кассир: <b>{{ currentUser.fullName }}</b></div>
          </div>

          <div>
            <label class="block text-xs font-semibold text-stone-700 uppercase mb-1">
              Сумма начального размена (₽)
            </label>
            <input
              v-model.number="openShiftCash"
              type="number"
              min="0"
              step="100"
              required
              class="w-full px-4 py-2.5 rounded-xl border border-stone-300 text-lg font-bold"
            />
            <div class="flex gap-2 mt-2">
              <button
                type="button"
                @click="openShiftCash = 1000"
                class="px-2.5 py-1 bg-stone-100 hover:bg-stone-200 text-stone-700 rounded-lg text-xs font-semibold cursor-pointer"
              >
                1 000 ₽
              </button>
              <button
                type="button"
                @click="openShiftCash = 3000"
                class="px-2.5 py-1 bg-stone-100 hover:bg-stone-200 text-stone-700 rounded-lg text-xs font-semibold cursor-pointer"
              >
                3 000 ₽
              </button>
              <button
                type="button"
                @click="openShiftCash = 5000"
                class="px-2.5 py-1 bg-stone-100 hover:bg-stone-200 text-stone-700 rounded-lg text-xs font-semibold cursor-pointer"
              >
                5 000 ₽
              </button>
            </div>
          </div>

          <div class="flex gap-2 pt-2">
            <button
              type="button"
              @click="showOpenShiftModal = false"
              class="flex-1 py-2.5 bg-stone-100 hover:bg-stone-200 text-stone-700 font-semibold rounded-xl text-sm cursor-pointer"
            >
              Отмена
            </button>
            <button
              type="submit"
              :disabled="isOpenShiftLoading"
              class="flex-1 py-2.5 bg-emerald-600 hover:bg-emerald-700 text-white font-bold rounded-xl text-sm shadow cursor-pointer transition flex items-center justify-center gap-2"
            >
              <span>{{ isOpenShiftLoading ? 'Открытие...' : 'Открыть смену' }}</span>
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- МОДАЛЬНОЕ ОКНО 2: X-ОТЧЁТ БЕЗ ГАШЕНИЯ -->
    <div
      v-if="showXReportModal"
      class="fixed inset-0 bg-stone-900/60 backdrop-blur-xs flex items-center justify-center p-4 z-50"
    >
      <div class="bg-white rounded-2xl max-w-lg w-full p-6 shadow-2xl border border-stone-200">
        <div class="flex justify-between items-center mb-4 pb-3 border-b border-stone-100">
          <div class="flex items-center gap-2 text-stone-900 font-bold text-lg">
            <FileText class="w-5 h-5 text-amber-600" />
            <span>X-отчёт (Промежуточный без гашения)</span>
          </div>
          <button @click="showXReportModal = false" class="text-stone-400 hover:text-stone-600 p-1">
            <X class="w-5 h-5" />
          </button>
        </div>

        <div v-if="xReportData" class="space-y-4">
          <div class="border border-dashed border-stone-300 p-4 rounded-xl font-mono text-xs space-y-1.5 bg-stone-50">
            <div class="text-center font-bold text-sm text-stone-900 pb-1 border-b border-stone-200">
              Пекарня «Восход» • X-ОТЧЕТ
            </div>
            <div class="flex justify-between">
              <span>Смена:</span>
              <span class="font-bold">#{{ xReportData.shiftId }}</span>
            </div>
            <div class="flex justify-between">
              <span>Кассир:</span>
              <span>{{ xReportData.cashierName }}</span>
            </div>
            <div class="flex justify-between">
              <span>Открыта:</span>
              <span>{{ new Date(xReportData.openedAt).toLocaleString('ru-RU') }}</span>
            </div>
            <div class="flex justify-between">
              <span>Снятие отчёта:</span>
              <span>{{ new Date(xReportData.reportTime).toLocaleString('ru-RU') }}</span>
            </div>
            <div class="border-t border-stone-200 pt-1.5 flex justify-between">
              <span>Начальный размен:</span>
              <span>{{ xReportData.initialCash.toFixed(2) }} ₽</span>
            </div>
            <div class="flex justify-between text-emerald-800">
              <span>Выручка наличными:</span>
              <span class="font-bold">{{ xReportData.cashRevenue.toFixed(2) }} ₽</span>
            </div>
            <div class="flex justify-between text-blue-800">
              <span>Выручка безналичная:</span>
              <span class="font-bold">{{ xReportData.cardRevenue.toFixed(2) }} ₽</span>
            </div>
            <div class="border-t border-stone-200 pt-1.5 flex justify-between font-bold text-sm text-amber-900">
              <span>ОБЩАЯ ВЫРУЧКА:</span>
              <span>{{ xReportData.totalRevenue.toFixed(2) }} ₽</span>
            </div>
            <div class="flex justify-between text-stone-600">
              <span>Чеков продажи:</span>
              <span>{{ xReportData.receiptCount }} шт.</span>
            </div>
            <div class="flex justify-between text-stone-600">
              <span>Возвратов прихода:</span>
              <span>{{ xReportData.refundedCount }} шт.</span>
            </div>
            <div class="border-t-2 border-stone-300 pt-1.5 flex justify-between font-bold text-sm text-stone-900">
              <span>ОЖИДАЕМЫЙ ОСТАТОК В КАССЕ:</span>
              <span>{{ xReportData.expectedCashInDrawer.toFixed(2) }} ₽</span>
            </div>
            <div class="text-center text-stone-400 text-[10px] pt-1">
              {{ xReportData.fiscalSlip }}
            </div>
          </div>

          <div class="flex gap-2">
            <button
              @click="window.print()"
              class="flex-1 py-2.5 bg-stone-100 hover:bg-stone-200 text-stone-700 font-semibold rounded-xl text-sm cursor-pointer flex items-center justify-center gap-1.5"
            >
              <Printer class="w-4 h-4" />
              <span>Печать</span>
            </button>
            <button
              @click="showXReportModal = false"
              class="flex-1 py-2.5 bg-amber-600 hover:bg-amber-700 text-white font-bold rounded-xl text-sm shadow cursor-pointer"
            >
              Закрыть
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- МОДАЛЬНОЕ ОКНО 3: ЗАКРЫТИЕ СМЕНЫ И Z-ОТЧЁТ -->
    <div
      v-if="showCloseShiftModal"
      class="fixed inset-0 bg-stone-900/60 backdrop-blur-xs flex items-center justify-center p-4 z-50"
    >
      <div class="bg-white rounded-2xl max-w-lg w-full p-6 shadow-2xl border border-stone-200">
        <div class="flex justify-between items-center mb-4 pb-3 border-b border-stone-100">
          <div class="flex items-center gap-2 text-rose-700 font-bold text-lg">
            <Lock class="w-5 h-5" />
            <span>Закрытие кассовой смены (Z-отчёт)</span>
          </div>
          <button @click="showCloseShiftModal = false" class="text-stone-400 hover:text-stone-600 p-1">
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Если смена уже закрыта в этой модалке, показываем результат -->
        <div v-if="closeShiftResult" class="space-y-4">
          <div class="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-center space-y-2">
            <CheckCircle2 class="w-10 h-10 text-emerald-600 mx-auto" />
            <div class="font-bold text-emerald-900 text-base">Смена успешно закрыта!</div>
            <div class="text-xs text-emerald-700">{{ closeShiftResult.message }}</div>
            <div class="text-xs font-mono font-semibold text-stone-700 pt-1">
              Фискальный Z-отчёт: {{ closeShiftResult.zReportSlip }}
            </div>
          </div>
          <button
            @click="showCloseShiftModal = false"
            class="w-full py-2.5 bg-stone-800 hover:bg-stone-900 text-white font-bold rounded-xl text-sm shadow cursor-pointer"
          >
            Завершить
          </button>
        </div>

        <!-- Форма закрытия смены -->
        <form v-else @submit.prevent="handleCloseShift" class="space-y-4">
          <div v-if="xReportData" class="grid grid-cols-2 gap-3 text-xs bg-stone-50 p-3.5 rounded-xl border border-stone-200">
            <div>
              <div class="text-stone-500">Начальный размен:</div>
              <div class="font-bold text-stone-900">{{ xReportData.initialCash.toFixed(2) }} ₽</div>
            </div>
            <div>
              <div class="text-stone-500">Выручка наличными:</div>
              <div class="font-bold text-emerald-800">+{{ xReportData.cashRevenue.toFixed(2) }} ₽</div>
            </div>
            <div>
              <div class="text-stone-500">Выручка по картам:</div>
              <div class="font-bold text-blue-800">{{ xReportData.cardRevenue.toFixed(2) }} ₽</div>
            </div>
            <div>
              <div class="text-stone-500">Ожидаемо в ящике:</div>
              <div class="font-bold text-amber-900">{{ xReportData.expectedCashInDrawer.toFixed(2) }} ₽</div>
            </div>
          </div>

          <div>
            <label class="block text-xs font-semibold text-stone-700 uppercase mb-1">
              Фактическая сумма наличных в ящике (₽)
            </label>
            <input
              v-model.number="actualCashInDrawer"
              type="number"
              step="1"
              required
              class="w-full px-4 py-2.5 rounded-xl border border-stone-300 text-lg font-bold"
            />
            <!-- Индикатор расхождения -->
            <div
              v-if="xReportData"
              class="mt-1.5 px-3 py-1.5 rounded-lg text-xs font-bold flex items-center justify-between"
              :class="discrepancy === 0 ? 'bg-emerald-50 text-emerald-700' : discrepancy > 0 ? 'bg-amber-50 text-amber-800' : 'bg-rose-50 text-rose-700'"
            >
              <span>Контроль баланса:</span>
              <span>
                {{ discrepancy === 0 ? '✅ Баланс сошёлся (0 ₽)' : discrepancy > 0 ? `🟢 Излишек: +${discrepancy} ₽` : `🔴 Недостача: ${discrepancy} ₽` }}
              </span>
            </div>
          </div>

          <div>
            <div class="flex justify-between items-center mb-1">
              <label class="text-xs font-semibold text-stone-700 uppercase">
                Сумма инкассации в сейф/банк (₽)
              </label>
              <div class="flex gap-1 text-[11px]">
                <button
                  type="button"
                  @click="encashmentSum = actualCashInDrawer"
                  class="text-amber-700 hover:underline cursor-pointer"
                >
                  Всё
                </button>
                <span>•</span>
                <button
                  type="button"
                  @click="encashmentSum = Math.max(0, actualCashInDrawer - 3000)"
                  class="text-amber-700 hover:underline cursor-pointer"
                >
                  Оставить размен (3000 ₽)
                </button>
              </div>
            </div>
            <input
              v-model.number="encashmentSum"
              type="number"
              step="1"
              required
              class="w-full px-4 py-2.5 rounded-xl border border-stone-300 text-lg font-bold"
            />
          </div>

          <div class="flex gap-2 pt-2">
            <button
              type="button"
              @click="showCloseShiftModal = false"
              class="flex-1 py-2.5 bg-stone-100 hover:bg-stone-200 text-stone-700 font-semibold rounded-xl text-sm cursor-pointer"
            >
              Отмена
            </button>
            <button
              type="submit"
              :disabled="isClosingShift"
              class="flex-1 py-2.5 bg-rose-600 hover:bg-rose-700 text-white font-bold rounded-xl text-sm shadow cursor-pointer transition flex items-center justify-center gap-2"
            >
              <span>{{ isClosingShift ? 'Печать Z-отчёта...' : 'Снять Z-отчёт и закрыть' }}</span>
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- МОДАЛЬНОЕ ОКНО 4: СПИСАНИЕ ВЫПЕЧКИ -->
    <div
      v-if="showWriteOffModal"
      class="fixed inset-0 bg-stone-900/60 backdrop-blur-xs flex items-center justify-center p-4 z-50"
    >
      <div class="bg-white rounded-2xl max-w-md w-full p-6 shadow-2xl border border-stone-200">
        <div class="flex justify-between items-center mb-4 pb-3 border-b border-stone-100">
          <div class="flex items-center gap-2 text-rose-700 font-bold text-lg">
            <PackageX class="w-5 h-5" />
            <span>Акт списания выпечки</span>
          </div>
          <button @click="showWriteOffModal = false" class="text-stone-400 hover:text-stone-600 p-1">
            <X class="w-5 h-5" />
          </button>
        </div>

        <form @submit.prevent="submitPosWriteOff" class="space-y-4">
          <div>
            <label class="block text-xs font-semibold text-stone-600 uppercase mb-1">Товар на витрине</label>
            <select
              v-model="posWriteOffForm.productId"
              class="w-full px-3 py-2.5 rounded-xl border border-stone-300 text-sm bg-white"
            >
              <option v-for="p in products" :key="p.productId" :value="p.productId">
                {{ p.name }} (в наличии: {{ p.currentQuantity }} шт. • {{ p.retailPrice }} ₽)
              </option>
            </select>
            <div v-if="selectedPosWriteOffProduct" class="mt-1 text-xs text-stone-500">
              Текущий остаток: <span class="font-bold text-stone-800">{{ selectedPosWriteOffProduct.currentQuantity }} шт.</span>
            </div>
          </div>

          <div>
            <label class="block text-xs font-semibold text-stone-600 uppercase mb-1">Количество к списанию (шт.)</label>
            <input
              v-model.number="posWriteOffForm.quantity"
              type="number"
              min="1"
              :max="selectedPosWriteOffProduct?.currentQuantity || 1"
              required
              class="w-full px-4 py-2.5 rounded-xl border border-stone-300 text-sm"
            />
          </div>

          <div>
            <label class="block text-xs font-semibold text-stone-600 uppercase mb-1">Причина</label>
            <select
              v-model="posWriteOffForm.reason"
              class="w-full px-3 py-2 rounded-xl border border-stone-300 text-xs bg-white mb-2"
            >
              <option v-for="r in predefinedReasons" :key="r" :value="r">{{ r }}</option>
              <option value="">-- Другая причина --</option>
            </select>
            <input
              v-if="!predefinedReasons.includes(posWriteOffForm.reason)"
              v-model="posWriteOffForm.reason"
              type="text"
              placeholder="Опишите причину списания..."
              class="w-full px-3 py-2 rounded-xl border border-stone-300 text-sm"
            />
          </div>

          <div class="flex gap-2 pt-2">
            <button
              type="button"
              @click="showWriteOffModal = false"
              class="flex-1 py-2.5 bg-stone-100 hover:bg-stone-200 text-stone-700 font-semibold rounded-xl text-sm cursor-pointer"
            >
              Отмена
            </button>
            <button
              type="submit"
              :disabled="isWritingOffPos || !selectedPosWriteOffProduct || selectedPosWriteOffProduct.currentQuantity === 0"
              class="flex-1 py-2.5 bg-rose-600 hover:bg-rose-700 disabled:opacity-50 text-white font-bold rounded-xl cursor-pointer text-sm flex items-center justify-center gap-1.5"
            >
              <Trash2 class="w-4 h-4" />
              <span>{{ isWritingOffPos ? 'Проведение...' : 'Списать' }}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
