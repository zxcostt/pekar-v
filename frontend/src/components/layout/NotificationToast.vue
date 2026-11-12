<script setup>
import { AlertCircle, CheckCircle2 } from 'lucide-vue-next'

defineProps({
  notification: {
    type: Object,
    default: () => ({ message: '', type: 'success' })
  }
})
</script>

<template>
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
</template>
