<script setup>
import { ref, computed } from 'vue'
import { api } from '../../api'
import { Minus, Plus, Banknote, CreditCard } from 'lucide-vue-next'

const props = defineProps({
  products: {
    type: Array,
    default: () => []
  },
  currentUser: {
    type: Object,
    required: true
  },
  lastPrintedReceipt: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['checkout-success', 'notify'])

// Состояние Кассы (UC 8..14)
const cart = ref([])
const discountRate = ref(0)
const paymentType = ref('CASH')
const tenderedCash = ref(500)
const cardPan = ref('**** **** **** 4242')

function addToCart(product) {
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
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
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
</template>
