<script setup>
import { ref } from 'vue'
import { Printer, Calendar, ShieldCheck, CheckCircle2, Lock, Unlock, FileSpreadsheet } from 'lucide-vue-next'

const props = defineProps({
  shiftReport: {
    type: Object,
    default: null
  },
  currentShift: {
    type: Object,
    default: null
  },
  shiftsHistory: {
    type: Array,
    default: () => []
  }
})

function exportShiftReportPdf() {
  window.print()
}

function formatDateTime(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<template>
  <div class="space-y-6">
    <!-- СВОДНЫЙ СМЕННЫЙ ОТЧЁТ (ТЕКУЩАЯ / ПОСЛЕДНЯЯ СМЕНА) -->
    <div v-if="shiftReport" class="bg-white p-8 rounded-2xl border border-stone-200 shadow-sm space-y-6">
      <div class="flex flex-wrap justify-between items-start gap-4">
        <div>
          <div class="flex items-center gap-3">
            <h2 class="text-xl font-bold">Сводный сменный отчёт (UC 3)</h2>
            <span
              v-if="currentShift"
              class="px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800 flex items-center gap-1.5"
            >
              <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
              Смена №{{ currentShift.shiftId }} (ОТКРЫТА)
            </span>
            <span
              v-else
              class="px-2.5 py-1 rounded-full text-xs font-semibold bg-stone-100 text-stone-600 flex items-center gap-1.5"
            >
              <Lock class="w-3.5 h-3.5" />
              Касса закрыта
            </span>
          </div>
          <p class="text-sm text-stone-500 mt-1">
            Пекарня «Восход» •
            <template v-if="currentShift">
              Ответственный кассир: <b>{{ currentShift.cashierName }}</b> • Открыта: {{ formatDateTime(currentShift.openedAt) }}
            </template>
            <template v-else>
              Смена №{{ shiftReport.shiftId }} • Открыта: {{ formatDateTime(shiftReport.openedAt) }}
            </template>
          </p>
        </div>

        <button
          @click="exportShiftReportPdf"
          class="px-5 py-2.5 bg-amber-600 hover:bg-amber-700 text-white font-semibold rounded-xl shadow cursor-pointer transition flex items-center gap-2"
        >
          <Printer class="w-4 h-4" />
          <span>Печать сменного отчёта</span>
        </button>
      </div>

      <!-- Ключевые показатели смены -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
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
        <div class="p-4 rounded-xl bg-rose-50 border border-rose-200">
          <div class="text-xs text-rose-800 uppercase font-semibold">Списано выпечки</div>
          <div class="text-2xl font-bold text-rose-900 mt-1">{{ shiftReport.writtenOffCount || 0 }} шт.</div>
          <div class="text-xs text-rose-600 font-medium mt-1">Потери: {{ (shiftReport.writtenOffTotal || 0).toFixed(2) }} ₽</div>
        </div>
      </div>
    </div>

    <!-- ТАБЛИЦА: ЖУРНАЛ И АРХИВ КАССОВЫХ СМЕН (Z-ОТЧЁТЫ И ИНКАССАЦИЯ) -->
    <div class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm space-y-4">
      <div class="flex items-center justify-between pb-3 border-b border-stone-100">
        <div class="flex items-center gap-2">
          <FileSpreadsheet class="w-5 h-5 text-amber-600" />
          <h3 class="font-bold text-lg">Журнал фискальных кассовых смен (Z-отчёты)</h3>
        </div>
        <span class="text-xs font-semibold px-2.5 py-1 bg-stone-100 text-stone-700 rounded-lg">
          Всего смен: {{ shiftsHistory.length }}
        </span>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead class="text-xs uppercase bg-stone-50 text-stone-500">
            <tr>
              <th class="p-3">№ Смены</th>
              <th class="p-3">Статус</th>
              <th class="p-3">Кассир</th>
              <th class="p-3">Открыта</th>
              <th class="p-3">Закрыта</th>
              <th class="p-3">Размен</th>
              <th class="p-3">Выручка (Нал / Безнал)</th>
              <th class="p-3">Итого</th>
              <th class="p-3">Инкассация</th>
              <th class="p-3">Фискальный Z-отчёт</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-stone-200">
            <tr
              v-for="s in shiftsHistory"
              :key="s.shiftId"
              :class="s.status === 'OPEN' ? 'bg-emerald-50/40 hover:bg-emerald-50/70' : 'hover:bg-stone-50'"
              class="transition"
            >
              <td class="p-3 font-bold font-mono">#{{ s.shiftId }}</td>
              <td class="p-3">
                <span
                  :class="[
                    'px-2.5 py-0.5 rounded-full text-xs font-semibold',
                    s.status === 'OPEN' ? 'bg-emerald-100 text-emerald-800' : 'bg-stone-200 text-stone-700'
                  ]"
                >
                  {{ s.status === 'OPEN' ? 'ОТКРЫТА' : 'ЗАКРЫТА' }}
                </span>
              </td>
              <td class="p-3 font-medium text-stone-900">{{ s.cashierName }}</td>
              <td class="p-3 text-xs text-stone-500">{{ formatDateTime(s.openedAt) }}</td>
              <td class="p-3 text-xs text-stone-500">{{ formatDateTime(s.closedAt) }}</td>
              <td class="p-3 text-xs font-semibold text-stone-700">{{ s.initialCash.toFixed(2) }} ₽</td>
              <td class="p-3 text-xs">
                <span class="text-emerald-700 font-medium">{{ s.cashRevenue.toFixed(2) }} ₽</span> /
                <span class="text-blue-700 font-medium">{{ s.cardRevenue.toFixed(2) }} ₽</span>
              </td>
              <td class="p-3 font-bold text-amber-900">
                {{ (s.cashRevenue + s.cardRevenue).toFixed(2) }} ₽
              </td>
              <td class="p-3 text-xs font-bold text-emerald-800">
                {{ s.encashmentAmount ? s.encashmentAmount.toFixed(2) + ' ₽' : '-' }}
              </td>
              <td class="p-3 font-mono text-xs text-stone-600">
                {{ s.zReportDoc || (s.status === 'OPEN' ? 'В процессе смены' : '-') }}
              </td>
            </tr>
            <tr v-if="shiftsHistory.length === 0">
              <td colspan="10" class="p-8 text-center text-stone-400">
                Архив кассовых смен пуст. Откройте первую смену на кассе.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
