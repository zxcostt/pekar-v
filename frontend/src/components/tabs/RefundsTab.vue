<script setup>
import { ref } from 'vue'
import { api } from '../../api'

defineProps({
  receipts: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['refund-success', 'notify'])

const refundForm = ref({
  fiscalDocNumber: '',
  reason: 'Возврат по обращению покупателя'
})

async function handleRefund(fiscalDoc) {
  const docToRefund = fiscalDoc || refundForm.value.fiscalDocNumber
  if (!docToRefund) {
    return emit('notify', 'Укажите фискальный номер чека', 'error')
  }

  try {
    const res = await api.refund(docToRefund, refundForm.value.reason)
    refundForm.value.fiscalDocNumber = ''
    emit('notify', `${res.message} (${res.refundFiscalSlip})`)
    emit('refund-success', res)
  } catch (err) {
    emit('notify', err.message, 'error')
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Оформление возврата -->
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

    <!-- Таблица чеков -->
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
</template>
