<template>
  <div v-if="isLoading" class="p-8 text-center text-gray-500">
    Loading ticket details...
  </div>
  
  <div v-else-if="error" class="p-8 text-center text-red-500">
    Failed to load ticket: {{ error.message }}
  </div>

  <div v-else-if="ticket" class="grid grid-cols-1 lg:grid-cols-3 gap-8">
    <!-- Main Content: Ticket info and Add Action form -->
    <div class="lg:col-span-2 space-y-6">
      <div class="glass-panel p-6">
        <div class="flex justify-between items-start mb-4">
          <div>
            <h2 class="text-2xl font-bold text-text">{{ ticket.subject }}</h2>
            <p class="text-gray-500 font-mono text-sm mt-1">#{{ ticket.id }}</p>
          </div>
          <span 
            class="px-3 py-1 rounded-full text-sm font-medium"
            :class="{
              'bg-yellow-100 text-yellow-800': ticket.status === 'TICKET_STATUS_OPEN',
              'bg-blue-100 text-blue-800': ticket.status === 'TICKET_STATUS_IN_PROGRESS',
              'bg-green-100 text-green-800': ticket.status === 'TICKET_STATUS_RESOLVED',
            }"
          >
            {{ formatStatus(ticket.status) }}
          </span>
        </div>

        <div class="prose max-w-none mb-6">
          <p class="text-gray-700 whitespace-pre-wrap">{{ ticket.description }}</p>
        </div>

        <div class="flex flex-wrap gap-4 text-sm text-gray-600 border-t border-gray-100 pt-4">
          <div class="flex items-center">
            <span class="font-medium mr-2">Sender:</span>
            {{ ticket.sender_email }}
          </div>
          <div class="flex items-center">
            <span class="font-medium mr-2">Assignee:</span>
            {{ ticket.assignee_id || 'Unassigned' }}
          </div>
          <div class="flex items-center">
            <span class="font-medium mr-2">Created:</span>
            {{ formatDate(ticket.created_at) }}
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="glass-panel p-6">
        <h3 class="text-lg font-semibold mb-4">Take Action</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Action Type</label>
            <select v-model="actionType" class="w-full border-gray-300 rounded-md shadow-sm focus:border-primary focus:ring-primary h-10 px-3">
              <option value="comment">Add Comment</option>
              <option value="status_change">Change Status</option>
              <option value="escalate">Escalate</option>
            </select>
          </div>
          
          <div v-if="actionType === 'comment'">
            <label class="block text-sm font-medium text-gray-700 mb-1">Comment</label>
            <textarea v-model="actionPayload" rows="3" class="w-full border-gray-300 rounded-md shadow-sm focus:border-primary focus:ring-primary p-3" placeholder="Type your comment here..."></textarea>
          </div>

          <div v-if="actionType === 'status_change'">
            <label class="block text-sm font-medium text-gray-700 mb-1">New Status</label>
            <select v-model="statusPayload" class="w-full border-gray-300 rounded-md shadow-sm focus:border-primary focus:ring-primary h-10 px-3">
              <option value="TICKET_STATUS_IN_PROGRESS">In Progress</option>
              <option value="TICKET_STATUS_RESOLVED">Resolved</option>
            </select>
          </div>

          <div class="flex justify-end">
            <button 
              @click="submitAction" 
              class="bg-primary text-white px-4 py-2 rounded-lg font-medium hover:bg-secondary transition-colors"
              :disabled="isDispatching"
            >
              {{ isDispatching ? 'Submitting...' : 'Submit' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Sidebar: Audit Timeline -->
    <div class="space-y-6">
      <div class="glass-panel p-6">
        <h3 class="text-lg font-semibold mb-4 flex items-center justify-between">
          Audit Timeline
          <span class="text-xs font-normal text-gray-500">{{ audits.length }} events</span>
        </h3>
        
        <div v-if="isLoadingAudits" class="text-center py-4 text-gray-500 text-sm">
          Loading timeline...
        </div>
        
        <div v-else-if="audits.length === 0" class="text-center py-4 text-gray-500 text-sm">
          No audit events found.
        </div>
        
        <div v-else class="space-y-6 relative before:absolute before:inset-0 before:ml-5 before:-translate-x-px md:before:mx-auto md:before:translate-x-0 before:h-full before:w-0.5 before:bg-gradient-to-b before:from-transparent before:via-slate-300 before:to-transparent">
          <div v-for="audit in audits" :key="audit.id" class="relative flex items-center justify-between md:justify-normal md:odd:flex-row-reverse group is-active">
            <div class="flex items-center justify-center w-10 h-10 rounded-full border border-white bg-slate-100 text-slate-500 shadow shrink-0 md:order-1 md:group-odd:-translate-x-1/2 md:group-even:translate-x-1/2 z-10">
              <span class="text-xs">{{ formatTimelineIcon(audit.action) }}</span>
            </div>
            <div class="w-[calc(100%-4rem)] md:w-[calc(50%-2.5rem)] bg-white p-4 rounded border border-slate-200 shadow-sm relative">
              <div class="flex items-center justify-between space-x-2 mb-1">
                <div class="font-bold text-slate-900 text-sm">{{ audit.action }}</div>
                <time class="font-mono text-xs text-indigo-500">{{ formatTime(audit.timestamp) }}</time>
              </div>
              <div class="text-sm text-slate-500" v-if="audit.details">{{ audit.details }}</div>
              <div class="text-xs mt-2 text-slate-400">By: {{ audit.actor_id }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useApi } from '~/composables/useApi'

const route = useRoute()
const ticketId = route.params.id as string

const { useApiQuery, useApiMutation } = useApi()

// Fetch ticket details
const { data: ticketData, isLoading, error, refetch: refetchTicket } = useApiQuery(
  ['ticket', ticketId], 
  `/api/v1/tickets/${ticketId}`
)

// Fetch audit logs
const { data: auditData, isLoading: isLoadingAudits, refetch: refetchAudits } = useApiQuery(
  ['audits', ticketId], 
  `/api/v1/audits?target_type=ticket&target_id=${ticketId}`
)

// Mutation for dispatching actions
const { mutate: dispatchActionMutation, isPending: isDispatching } = useApiMutation(
  '/api/v1/actions/dispatch', 
  'POST'
)

const ticket = computed(() => ticketData.value?.ticket)
const audits = computed(() => auditData.value?.logs || [])

// Form state
const actionType = ref('comment')
const actionPayload = ref('')
const statusPayload = ref('TICKET_STATUS_IN_PROGRESS')

const submitAction = () => {
  let payloadStr = ""
  
  if (actionType.value === 'comment') {
    payloadStr = JSON.stringify({ comment: actionPayload.value })
  } else if (actionType.value === 'status_change') {
    payloadStr = JSON.stringify({ status: statusPayload.value })
  }

  dispatchActionMutation(
    {
      action_schema_id: actionType.value,
      target_resource_id: ticketId,
      actor_id: 'current-user', // Mock for now
      payload: payloadStr
    },
    {
      onSuccess: () => {
        actionPayload.value = ''
        refetchTicket()
        refetchAudits()
      },
      onError: (err) => {
        alert(err.message || 'Failed to dispatch action')
      }
    }
  )
}

const formatStatus = (status: string) => {
  if (!status) return 'Unknown'
  return status.replace('TICKET_STATUS_', '').replace(/_/g, ' ')
}

const formatDate = (dateString: string) => {
  if (!dateString) return ''
  return new Date(dateString).toLocaleDateString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
  })
}

const formatTime = (dateString: string) => {
  if (!dateString) return ''
  const d = new Date(dateString)
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}

const formatTimelineIcon = (action: string) => {
  if (action.includes('COMMENT')) return '💬'
  if (action.includes('STATUS')) return '🔄'
  if (action.includes('CREATE')) return '✨'
  if (action.includes('ASSIGN')) return '👤'
  return '📝'
}
</script>
