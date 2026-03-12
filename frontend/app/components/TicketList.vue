<template>
  <div class="w-full">
    <div v-if="isLoading" class="p-8 text-center text-gray-500">
      Loading tickets...
    </div>
    
    <div v-else-if="error" class="p-8 text-center text-red-500">
      {{ error.message }}
    </div>

    <div v-else-if="tickets.length === 0" class="p-8 text-center text-gray-500">
      No tickets found.
    </div>
    
    <div v-else>
      <!-- Table Header -->
      <div class="grid grid-cols-12 gap-4 px-6 py-3 border-b border-gray-100 bg-gray-50/50 text-xs font-semibold text-gray-500 uppercase tracking-wider">
        <div class="col-span-1">ID</div>
        <div class="col-span-4">Subject</div>
        <div class="col-span-2">Sender</div>
        <div class="col-span-2">Status</div>
        <div class="col-span-2">Assignee</div>
        <div class="col-span-1 text-right">Actions</div>
      </div>

      <!-- Table Body -->
      <div 
        v-for="ticket in tickets" 
        :key="ticket.id" 
        class="grid grid-cols-12 gap-4 px-6 py-4 items-center border-b border-gray-50 hover:bg-gray-50/50 transition-colors cursor-pointer group"
      >
        <div class="col-span-1 font-mono text-xs text-gray-500">#{{ ticket.id.substring(0, 6) }}</div>
        
        <div class="col-span-4">
          <NuxtLink :to="`/ticket/${ticket.id}`" class="text-sm font-medium text-text group-hover:text-primary transition-colors block truncate">
            {{ ticket.subject }}
          </NuxtLink>
        </div>
        
        <div class="col-span-2">
          <div class="text-sm text-gray-700 truncate">{{ ticket.sender_email }}</div>
        </div>
        
        <div class="col-span-2">
          <span 
            class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
            :class="{
              'bg-yellow-100 text-yellow-800': ticket.status === 'TICKET_STATUS_OPEN',
              'bg-blue-100 text-blue-800': ticket.status === 'TICKET_STATUS_IN_PROGRESS',
              'bg-green-100 text-green-800': ticket.status === 'TICKET_STATUS_RESOLVED',
              'bg-gray-100 text-gray-800': ticket.status === 'TICKET_STATUS_MERGED',
            }"
          >
            {{ formatStatus(ticket.status) }}
          </span>
        </div>
        
        <div class="col-span-2">
          <div v-if="ticket.assignee_id" class="flex items-center space-x-2">
            <div class="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center text-primary text-xs font-bold">
              {{ ticket.assignee_id.substring(0, 1).toUpperCase() }}
            </div>
            <span class="text-sm text-gray-700 truncate">{{ ticket.assignee_id }}</span>
          </div>
          <span v-else class="text-sm text-gray-400 italic">Unassigned</span>
        </div>
        
        <div class="col-span-1 text-right">
          <NuxtLink :to="`/ticket/${ticket.id}`" class="text-primary hover:text-secondary text-sm font-medium">
            View
          </NuxtLink>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useApi } from '~/composables/useApi'

// Using the useApi composite injected globally or imported locally
const { useApiQuery } = useApi()

// Fetch tickets
const { data, isLoading, error } = useApiQuery(['tickets'], '/api/v1/tickets')

const tickets = computed(() => {
  return data.value?.tickets || []
})

const formatStatus = (status: string) => {
  if (!status) return 'Unknown';
  return status.replace('TICKET_STATUS_', '').replace(/_/g, ' ')
}
</script>
