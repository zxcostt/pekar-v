<script setup>
import { ref } from 'vue'
import { api } from '../../api'

const props = defineProps({
  recipes: {
    type: Array,
    default: () => []
  },
  ingredients: {
    type: Array,
    default: () => []
  },
  currentUser: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['create-recipe-success', 'notify'])

const newRecipe = ref({
  recipeTitle: '',
  productName: '',
  retailPrice: 85,
  yieldQuantity: 10,
  items: [{ ingredientId: 1, requiredQuantity: 0.5 }]
})

function addRecipeRow() {
  const defaultIngId = props.ingredients[0]?.ingredientId || 1
  newRecipe.value.items.push({ ingredientId: defaultIngId, requiredQuantity: 0.1 })
}

async function handleCreateRecipe() {
  try {
    await api.createRecipe({
      recipeTitle: newRecipe.value.recipeTitle,
      productName: newRecipe.value.productName ? newRecipe.value.productName.trim() : undefined,
      retailPrice: Number(newRecipe.value.retailPrice) || 80,
      yieldQuantity: Number(newRecipe.value.yieldQuantity),
      items: newRecipe.value.items.map(i => ({
        ingredientId: Number(i.ingredientId),
        requiredQuantity: Number(i.requiredQuantity)
      }))
    })
    emit('notify', 'Техкарта сохранена, изделие добавлено в ассортимент цеха и на витрину!')
    newRecipe.value = {
      recipeTitle: '',
      productName: '',
      retailPrice: 85,
      yieldQuantity: 10,
      items: [{ ingredientId: props.ingredients[0]?.ingredientId || 1, requiredQuantity: 0.5 }]
    }
    emit('create-recipe-success')
  } catch (err) {
    emit('notify', err.message, 'error')
  }
}
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Форма создания ТК (только для управляющего) -->
    <div v-if="currentUser.role === 'MANAGER'" class="bg-white p-6 rounded-2xl border border-stone-200 shadow-sm">
      <h3 class="font-bold text-lg mb-3">Добавить техкарту (UC 2)</h3>
      <form @submit.prevent="handleCreateRecipe" class="space-y-3">
        <div>
          <label class="block text-xs font-medium text-stone-600 mb-1">Название техкарты</label>
          <input
            v-model="newRecipe.recipeTitle"
            type="text"
            required
            placeholder="Например: ТК №2: Багет французский"
            class="w-full px-3 py-2 rounded-xl border border-stone-300"
          />
        </div>
        <div>
          <label class="block text-xs font-medium text-stone-600 mb-1">Наименование изделия для витрины</label>
          <input
            v-model="newRecipe.productName"
            type="text"
            placeholder="Например: Багет французский (по умолчанию из ТК)"
            class="w-full px-3 py-2 rounded-xl border border-stone-300"
          />
        </div>
        <div class="grid grid-cols-2 gap-2">
          <div>
            <label class="block text-xs font-medium text-stone-600 mb-1">Выход партии (шт.)</label>
            <input
              v-model.number="newRecipe.yieldQuantity"
              type="number"
              min="1"
              class="w-full px-3 py-2 rounded-xl border border-stone-300"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-stone-600 mb-1">Розничная цена (₽ / шт.)</label>
            <input
              v-model.number="newRecipe.retailPrice"
              type="number"
              min="1"
              step="1"
              class="w-full px-3 py-2 rounded-xl border border-stone-300"
            />
          </div>
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

    <!-- Список техкарт -->
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
</template>
