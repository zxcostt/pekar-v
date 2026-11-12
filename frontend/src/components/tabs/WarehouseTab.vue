<script setup>
import { ref, watch } from 'vue'
import { api } from '../../api'

const props = defineProps({
  ingredients: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['supply-success', 'writeoff-success', 'notify'])

const supplyForm = ref({ ingredientId: 1, amount: 10 })
const writeOffForm = ref({ ingredientId: 1, amount: 1, reason: 'Истечение срока годности / порча' })

watch(() => props.ingredients, (newList) => {
  if (newList.length) {
    if (!supplyForm.value.ingredientId) supplyForm.value.ingredientId = newList[0].ingredientId
    if (!writeOffForm.value.ingredientId) writeOffForm.value.ingredientId = newList[0].ingredientId
  }
}, { immediate: true })

async function handleSupply() {
  try {
    await api.supplyIngredient(Number(supplyForm.value.ingredientId), Number(supplyForm.value.amount))
    emit('notify', 'Поставка сырья успешно проведена, остаток увеличен')
    emit('supply-success')
  } catch (err) {
    emit('notify', err.message, 'error')
  }
}

async function handleWriteOff() {
  try {
    await api.writeOffIngredient(
      Number(writeOffForm.value.ingredientId),
      Number(writeOffForm.value.amount),
      writeOffForm.value.reason
    )
    emit('notify', 'Акт списания сырья проведён')
    emit('writeoff-success')
  } catch (err) {
    emit('notify', err.message, 'error')
  }
}
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
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
</template>
