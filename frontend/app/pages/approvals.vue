<template>
  <div>
    <h2 class="text-2xl font-bold text-text mb-6">Approvals Dashboard</h2>

    <div v-if="isLoading" class="p-8 text-center text-gray-500">
      Loading approvals...
    </div>

    <div v-else-if="error" class="p-8 text-center text-red-500">
      {{ error.message }}
    </div>

    <div v-else-if="approvals.length === 0" class="p-8 text-center text-gray-500">
      No pending approvals.
    </div>

    <div v-else class="grid grid-cols-1 gap-6">
      <div v-for="approval in approvals" :key="approval.id" class="glass-panel p-6">
        <div class="flex justify-between items-start mb-4">
          <div>
            <h3 class="text-lg font-semibold text-text">Approval Request #{{ approval.id.substring(0, 8) }}</h3>
            <p class="text-sm text-gray-500 mt-1">Ticket ID: <NuxtLink :to="`/ticket/${approval.ticket_id}`" class="text-primary hover:underline">{{ approval.ticket_id }}</NuxtLink></p>
          </div>
          <span 
            class="px-3 py-1 rounded-full text-sm font-medium"
            :class="{
              'bg-yellow-100 text-yellow-800': approval.status === 'APPROVAL_STATUS_PENDING',
              'bg-green-100 text-green-800': approval.status === 'APPROVAL_STATUS_APPROVED',
              'bg-red-100 text-red-800': approval.status === 'APPROVAL_STATUS_REJECTED',
            }"
          >
            {{ formatStatus(approval.status) }}
          </span>
        </div>

        <div class="bg-gray-50 rounded-lg p-4 mb-4 font-mono text-sm overflow-x-auto border border-gray-100 text-gray-700">
          <strong>Proposed Action:</strong> {{ approval.action_id }}
        </div>

        <div v-if="approval.status === 'APPROVAL_STATUS_PENDING'" class="flex justify-end space-x-3">
          <button 
            @click="decideApproval(approval.id, 'reject')" 
            class="px-4 py-2 border border-red-200 text-red-600 rounded-lg font-medium hover:bg-red-50 transition-colors"
            :disabled="isDeciding"
          >
            Reject
          </button>
          <button 
            @click="decideApproval(approval.id, 'approve')" 
            class="px-4 py-2 bg-green-500 text-white rounded-lg font-medium hover:bg-green-600 transition-colors shadow-sm"
            :disabled="isDeciding"
          >
            Approve
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useApi } from '~/composables/useApi'

const { useApiQuery, useApiMutation } = useApi()

// Fetch open approvals
const { data, isLoading, error, refetch } = useApiQuery(
  ['approvals'], 
  '/api/v1/approvals'
)

const approvals = computed(() => data.value?.approvals || [])

// Mutation for approving/rejecting
const { mutate: decideMutation, isPending: isDeciding } = useApiMutation(
  '', 'POST'
)

const decideApproval = (id: string, action: 'approve' | 'reject') => {
  const endpoint = `/api/v1/approvals/${id}/${action}`
  
  decideMutation(
    { approver_id: 'manager-1', comment: `Manager ${action}d` },
    {
      onSuccess: () => refetch(),
      onError: (err) => alert('Failed to process approval: ' + err.message)
    }
  )
}

const formatStatus = (status: string) => {
  if (!status) return 'Unknown'
  return status.replace('APPROVAL_STATUS_', '').replace(/_/g, ' ')
}
</script>
