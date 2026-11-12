<script setup>
import { ref, watch } from 'vue'
import { api } from '../../api'

const props = defineProps({
  products: {
    type: Array,
    default: () => []
  },
  productionOrders: {
    type: Array,
    default: () => []
  },
  currentUser: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['produce-success', 'notify'])

const produceForm = ref({ productId: 1, producedUnits: 10 })

// Если список товаров загружен и productId не задан, выставляем первый
watch(() => props.products, (newProducts) => {
  if (newProducts.length && !produceForm.value.productId) {
    produceForm.value.productId = newProducts[0].productId
  }
}, { immediate: true })

async function handleProduce() {
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
  }
}
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
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
</template>
