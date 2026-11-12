<script setup>
defineProps({
  shiftReport: {
    type: Object,
    default: null
  }
})

function exportShiftReportPdf() {
  window.print()
}
</script>

<template>
  <div v-if="shiftReport" class="space-y-6">
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
  <div v-else class="text-center py-12 text-stone-500">
    Загрузка сменного отчёта...
  </div>
</template>
