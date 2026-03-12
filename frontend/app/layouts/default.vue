<template>
  <div class="min-h-screen bg-background font-sans flex flex-col">
    <header class="bg-white/90 backdrop-blur-sm border-b border-gray-100 shadow-sm sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16 items-center">
          <div class="flex items-center space-x-2">
            <h1 class="text-xl font-bold bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent">Go Support Ticket</h1>
          </div>
          <nav>
            <NuxtLink to="/" class="text-text hover:text-primary transition-colors font-medium px-3 py-2">Dashboard</NuxtLink>
            <NuxtLink to="/approvals" class="text-text hover:text-primary transition-colors font-medium px-3 py-2">Approvals</NuxtLink>
            <button class="ml-4 bg-primary text-white px-4 py-2 rounded-lg font-medium hover:bg-secondary transition-colors shadow-sm">
              New Ticket
            </button>
          </nav>
        </div>
      </div>
    </header>

    <main class="flex-grow w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <slot />
    </main>
  </div>
</template>
