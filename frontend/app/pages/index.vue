<template>
  <div>
    <h2 class="text-2xl font-bold text-text mb-6">Dashboard</h2>
    
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
      <!-- Status Cards -->
      <div class="glass-panel p-6 border-l-4 border-l-primary">
        <h3 class="text-gray-500 font-medium text-sm">Open Tickets</h3>
        <p class="text-3xl font-bold mt-2">12</p>
      </div>
      <div class="glass-panel p-6 border-l-4 border-l-cta">
        <h3 class="text-gray-500 font-medium text-sm">Pending Approvals</h3>
        <p class="text-3xl font-bold mt-2">3</p>
      </div>
      <div class="glass-panel p-6 border-l-4 border-l-green-500">
        <h3 class="text-gray-500 font-medium text-sm">Resolved (7d)</h3>
        <p class="text-3xl font-bold mt-2">45</p>
      </div>
    </div>

    <!-- Ticket List -->
    <div class="glass-panel overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-100 flex justify-between items-center bg-white/50">
        <h3 class="text-lg font-semibold text-text">Recent Tickets</h3>
      </div>
      <TicketList />
    </div>
  </div>
</template>

<script setup lang="ts">
import TicketList from '~/components/TicketList.vue'
</script>
