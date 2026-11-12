<script setup>
import { ref, watch, computed } from 'vue'
import { api } from '../../api'
import { ChefHat, PackageX, PackageCheck, AlertCircle, Trash2 } from 'lucide-vue-next'

const props = defineProps({
  products: {
    type: Array,
    default: () => []
  },
  productionOrders: {
    type: Array,
    default: () => []
  },
  productWriteOffs: {
    type: Array,
    default: () => []
  },
  currentUser: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['produce-success', 'writeoff-success', 'notify'])

// Вкладка активного журнала справа
const activeJournal = ref('orders') // 'orders' | 'writeoffs'

// Форма выпуска продукции (UC 6 + UC 7)
const produceForm = ref({ productId: 1, producedUnits: 10 })
const isProducing = ref(false)

// Форма списания готовой продукции (Акт списания выпечки)
const writeOffForm = ref({
  productId: 1,
  quantity: 1,
  reason: 'Истечение срока годности (вечерний остаток)'
})
const isWritingOff = ref(false)

const predefinedReasons = [
  'Истечение срока годности (вечерний остаток)',
  'Брак выпечки (подгорело / пересушено)',
  'Потеря товарного вида / деформация',
  'Дегустация / витринный образец'
]

// Если список товаров загружен и productId не задан, выставляем первый
watch(() => props.products, (newProducts) => {
  if (newProducts.length) {
    if (!produceForm.value.productId || !newProducts.some(p => p.productId === produceForm.value.productId)) {
      produceForm.value.productId = newProducts[0].productId
    }
    if (!writeOffForm.value.productId || !newProducts.some(p => p.productId === writeOffForm.value.productId)) {
      writeOffForm.value.productId = newProducts[0].productId
    }
  }
}, { immediate: true })

const selectedWriteOffProduct = computed(() => {
  return props.products.find(p => p.productId === writeOffForm.value.productId)
})

async function handleProduce() {
  if (produceForm.value.producedUnits <= 0) {
    return emit('notify', 'Количество единиц должно быть больше 0', 'error')
  }
  isProducing.value = true
  try {
    const res = await api.produceProduct(
      Number(produceForm.value.productId),
      props.currentUser.id,
      Number(produceForm.value.producedUnits)
    )
    emit('notify', res.message)
    emit('produce-success', res)
  } catch (err) {
    emit('notify', err.message, 'error')
  } finally {
    isProducing.value = false
  }
}

async function handleWriteOff() {
  const prod = selectedWriteOffProduct.value
  if (!prod) {
    return emit('notify', 'Выберите изделие для списания', 'error')
  }
  if (writeOffForm.value.quantity <= 0) {
    return emit('notify', 'Количество к списанию должно быть больше 0', 'error')
  }
  if (writeOffForm.value.quantity > prod.currentQuantity) {
    return emit('notify', `Нельзя списать ${writeOffForm.value.quantity} шт.: на витрине всего ${prod.currentQuantity} шт.`, 'error')
  }
  if (!writeOffForm.value.reason.trim()) {
    return emit('notify', 'Укажите причину списания', 'error')
  }

  isWritingOff.value = true
  try {
    const res = await api.writeOffProduct(
      Number(writeOffForm.value.productId),
      props.currentUser.id,
      Number(writeOffForm.value.quantity),
      writeOffForm.value.reason.trim()
    )
    emit('notify', res.message || 'Акт списания выпечки успешно проведён')
    emit('writeoff-success', res)
    activeJournal.value = 'writeoffs'
    writeOffForm.value.quantity = 1
  } catch (err) {
    emit('notify', err.message, 'error')
  } finally {
    isWritingOff.value = false
  }
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- ЛЕВАЯ КОЛОНКА: ОПЕРАТИВНЫЕ ДОКУМЕНТЫ (ВЫПУСК И СПИСАНИЕ) -->
    <div class="space-y-6">
      <!-- 1. Акт выпуска выпечки (UC 6 + UC 7) -->
      <div class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
        <div class="flex items-center gap-2 mb-2 text-amber-800">
          <ChefHat class="w-5 h-5 text-amber-600" />
          <h2 class="text-lg font-bold">Выпуск продукции (Цех)</h2>
        </div>
        <p class="text-xs text-stone-500 mb-4">
          Списывает сырьё со склада по техкарте и добавляет выпечку на баланс витрины.
        </p>
        <form @submit.prevent="handleProduce" class="space-y-4">
          <div>
            <label class="block text-xs font-semibold uppercase text-stone-600 mb-1">Готовое изделие</label>
            <select
              v-model="produceForm.productId"
              class="w-full px-3 py-2.5 rounded-xl border border-stone-300 bg-white text-sm"
            >
              <option v-for="p in products" :key="p.productId" :value="p.productId">
                {{ p.name }} (на витрине: {{ p.currentQuantity }} шт.)
              </option>
            </select>
          </div>
          <div>
            <label class="block text-xs font-semibold uppercase text-stone-600 mb-1">Объём выпуска (шт.)</label>
            <input
              v-model.number="produceForm.producedUnits"
              type="number"
              min="1"
              required
              class="w-full px-4 py-2.5 rounded-xl border border-stone-300 text-sm"
            />
          </div>
          <button
            type="submit"
            :disabled="isProducing"
            class="w-full py-2.5 bg-amber-600 hover:bg-amber-700 disabled:opacity-50 text-white font-bold rounded-xl cursor-pointer transition flex items-center justify-center gap-2"
          >
            <PackageCheck class="w-4 h-4" />
            <span>{{ isProducing ? 'Проведение...' : 'Подтвердить выпуск' }}</span>
          </button>
        </form>
      </div>

      <!-- 2. Акт списания готовой продукции с витрины -->
      <div class="bg-white p-6 rounded-2xl border border-rose-200 shadow-sm">
        <div class="flex items-center gap-2 mb-2 text-rose-800">
          <PackageX class="w-5 h-5 text-rose-600" />
          <h2 class="text-lg font-bold">Списание с витрины</h2>
        </div>
        <p class="text-xs text-stone-500 mb-4">
          Списывает черствую, бракованную или поврежденную выпечку с витрины с фиксацией акта списания.
        </p>
        <form @submit.prevent="handleWriteOff" class="space-y-4">
          <div>
            <label class="block text-xs font-semibold uppercase text-stone-600 mb-1">Изделие на витрине</label>
            <select
              v-model="writeOffForm.productId"
              class="w-full px-3 py-2.5 rounded-xl border border-stone-300 bg-white text-sm"
            >
              <option v-for="p in products" :key="p.productId" :value="p.productId">
                {{ p.name }} — доступно: {{ p.currentQuantity }} шт. ({{ p.retailPrice }} ₽)
              </option>
            </select>
            <div v-if="selectedWriteOffProduct" class="mt-1 text-xs" :class="selectedWriteOffProduct.currentQuantity > 0 ? 'text-stone-500' : 'text-rose-600 font-semibold'">
              Остаток на витрине: {{ selectedWriteOffProduct.currentQuantity }} шт.
            </div>
          </div>

          <div>
            <label class="block text-xs font-semibold uppercase text-stone-600 mb-1">Количество к списанию (шт.)</label>
            <input
              v-model.number="writeOffForm.quantity"
              type="number"
              min="1"
              :max="selectedWriteOffProduct?.currentQuantity || 1"
              required
              class="w-full px-4 py-2.5 rounded-xl border border-stone-300 text-sm"
            />
          </div>

          <div>
            <label class="block text-xs font-semibold uppercase text-stone-600 mb-1">Причина списания</label>
            <select
              v-model="writeOffForm.reason"
              class="w-full px-3 py-2 rounded-xl border border-stone-300 bg-white text-xs mb-2"
            >
              <option v-for="r in predefinedReasons" :key="r" :value="r">
                {{ r }}
              </option>
              <option value="">-- Указать другую причину --</option>
            </select>
            <input
              v-if="!predefinedReasons.includes(writeOffForm.reason)"
              v-model="writeOffForm.reason"
              type="text"
              placeholder="Введите причину списания..."
              class="w-full px-3 py-2 rounded-xl border border-stone-300 text-sm"
            />
          </div>

          <button
            type="submit"
            :disabled="isWritingOff || !selectedWriteOffProduct || selectedWriteOffProduct.currentQuantity === 0"
            class="w-full py-2.5 bg-rose-600 hover:bg-rose-700 disabled:opacity-50 text-white font-bold rounded-xl cursor-pointer transition flex items-center justify-center gap-2"
          >
            <Trash2 class="w-4 h-4" />
            <span>{{ isWritingOff ? 'Списание...' : 'Оформить списание выпечки' }}</span>
          </button>
        </form>
      </div>
    </div>

    <!-- ПРАВАЯ КОЛОНКА: ЖУРНАЛЫ ВЫПУСКА И СПИСАНИЙ -->
    <div class="lg:col-span-2 bg-white p-6 rounded-2xl border border-stone-200 shadow-sm flex flex-col">
      <!-- Переключатель журналов -->
      <div class="flex items-center justify-between pb-4 mb-4 border-b border-stone-200">
        <div class="flex gap-2">
          <button
            @click="activeJournal = 'orders'"
            :class="[
              'px-4 py-2 rounded-xl text-sm font-semibold transition cursor-pointer flex items-center gap-2',
              activeJournal === 'orders'
                ? 'bg-amber-100 text-amber-900 border border-amber-300'
                : 'bg-stone-100 hover:bg-stone-200 text-stone-600'
            ]"
          >
            <ChefHat class="w-4 h-4" />
            <span>Журнал выпуска</span>
            <span class="px-2 py-0.5 text-xs bg-amber-200 text-amber-900 rounded-full font-bold">
              {{ productionOrders.length }}
            </span>
          </button>

          <button
            @click="activeJournal = 'writeoffs'"
            :class="[
              'px-4 py-2 rounded-xl text-sm font-semibold transition cursor-pointer flex items-center gap-2',
              activeJournal === 'writeoffs'
                ? 'bg-rose-100 text-rose-900 border border-rose-300'
                : 'bg-stone-100 hover:bg-stone-200 text-stone-600'
            ]"
          >
            <PackageX class="w-4 h-4" />
            <span>Журнал списаний витрины</span>
            <span class="px-2 py-0.5 text-xs bg-rose-200 text-rose-900 rounded-full font-bold">
              {{ productWriteOffs.length }}
            </span>
          </button>
        </div>
      </div>

      <!-- 1. ТАБЛИЦА ВЫПУСКА ВЫПЕЧКИ -->
      <div v-if="activeJournal === 'orders'" class="overflow-x-auto flex-1">
        <table class="w-full text-sm text-left">
          <thead class="text-xs uppercase bg-stone-50 text-stone-500">
            <tr>
              <th class="p-3">№ Наряда</th>
              <th class="p-3">Время</th>
              <th class="p-3">Техкарта</th>
              <th class="p-3">Пекарь</th>
              <th class="p-3">Выпущено</th>
              <th class="p-3">Статус</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-stone-200">
            <tr v-for="o in productionOrders" :key="o.orderId" class="hover:bg-stone-50 transition">
              <td class="p-3 font-bold font-mono">#{{ o.orderId }}</td>
              <td class="p-3 text-xs text-stone-500">{{ formatDate(o.orderDate) }}</td>
              <td class="p-3 font-medium">{{ o.recipeTitle }}</td>
              <td class="p-3 text-stone-600">{{ o.bakerName }}</td>
              <td class="p-3 font-bold text-amber-900">{{ o.producedUnits }} шт.</td>
              <td class="p-3">
                <span class="px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-700">
                  {{ o.orderStatus }}
                </span>
              </td>
            </tr>
            <tr v-if="productionOrders.length === 0">
              <td colspan="6" class="p-8 text-center text-stone-400">
                Журнал нарядов пуст. Оформите первый выпуск выпечки.
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 2. ТАБЛИЦА СПИСАНИЙ ГОТОВОЙ ВЫПЕЧКИ -->
      <div v-else class="overflow-x-auto flex-1">
        <table class="w-full text-sm text-left">
          <thead class="text-xs uppercase bg-stone-50 text-stone-500">
            <tr>
              <th class="p-3">№ Акта</th>
              <th class="p-3">Время</th>
              <th class="p-3">Изделие</th>
              <th class="p-3">Списано</th>
              <th class="p-3">Сумма потерь</th>
              <th class="p-3">Причина</th>
              <th class="p-3">Сотрудник</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-stone-200">
            <tr v-for="w in productWriteOffs" :key="w.writeOffId" class="hover:bg-rose-50/40 transition">
              <td class="p-3 font-bold font-mono text-rose-800">#АКТ-{{ w.writeOffId }}</td>
              <td class="p-3 text-xs text-stone-500">{{ formatDate(w.writeOffDate) }}</td>
              <td class="p-3 font-medium text-stone-900">{{ w.productName }}</td>
              <td class="p-3 font-bold text-rose-700">{{ w.quantity }} шт.</td>
              <td class="p-3 text-xs font-semibold text-stone-700">
                {{ (w.quantity * (w.retailPrice || 0)).toFixed(2) }} ₽
              </td>
              <td class="p-3 text-xs text-stone-600">
                <span class="px-2 py-0.5 bg-stone-100 rounded text-stone-700">
                  {{ w.reason }}
                </span>
              </td>
              <td class="p-3 text-xs text-stone-500">{{ w.userName }}</td>
            </tr>
            <tr v-if="productWriteOffs.length === 0">
              <td colspan="7" class="p-8 text-center text-stone-400">
                Списаний готовой продукции не зафиксировано.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
