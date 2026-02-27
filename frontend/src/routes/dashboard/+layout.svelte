<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  
  let user: any = null;
  let isMobileMenuOpen = false;
  
  $: currentPath = $page.url.pathname;

  // Session Management State
  let lastActivityTimestamp = Date.now();
  let sessionInterval: any;
  const IDLE_TIMEOUT_MS = 60 * 60 * 1000; // 1 hour
  const TOKEN_REFRESH_THRESHOLD_MS = 30 * 60 * 1000; // 30 minutes
  let lastTokenRefresh = Date.now();

  function updateActivity() {
    lastActivityTimestamp = Date.now();
  }

  onMount(() => {
    const token = localStorage.getItem('monitor_token');
    const userData = localStorage.getItem('monitor_user');
    
    if (!token || !userData) {
      window.location.href = '/';
      return;
    }
    
    user = JSON.parse(userData);

    // Global Fetch Interceptor to catch 401/403 anywhere in the dashboard
    const originalFetch = window.fetch;
    window.fetch = async (...args) => {
      const response = await originalFetch(...args);
      if (response.status === 401 || response.status === 403) {
        console.warn("Global fetch caught unauthorized response. Logging out.");
        handleLogout();
      }
      return response;
    };

    // Setup Activity Listeners
    window.addEventListener('mousemove', updateActivity);
    window.addEventListener('keydown', updateActivity);
    window.addEventListener('click', updateActivity);
    window.addEventListener('scroll', updateActivity);

    // Setup Session Manager Interval (Runs every 5 minutes)
    sessionInterval = setInterval(checkSessionStatus, 5 * 60 * 1000);

    return () => {
      window.fetch = originalFetch; // Restore original fetch
      window.removeEventListener('mousemove', updateActivity);
      window.removeEventListener('keydown', updateActivity);
      window.removeEventListener('click', updateActivity);
      window.removeEventListener('scroll', updateActivity);
      if (sessionInterval) clearInterval(sessionInterval);
    };
  });

  async function checkSessionStatus() {
    const now = Date.now();
    
    // Check Idle Timeout (1 Hour)
    if (now - lastActivityTimestamp > IDLE_TIMEOUT_MS) {
      console.warn("Session expired due to inactivity.");
      handleLogout();
      return;
    }

    // Check Token Keep-Alive (Refresh every 30 mins if active)
    if (now - lastTokenRefresh > TOKEN_REFRESH_THRESHOLD_MS) {
      await performTokenRefresh();
    }
  }

  async function performTokenRefresh() {
    try {
      const token = localStorage.getItem('monitor_token');
      if (!token) return;

      const res = await fetch('http://localhost:5273/api/v1/auth/refresh', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });

      if (res.ok) {
        const data = await res.json();
        if (data.token) {
          localStorage.setItem('monitor_token', data.token);
          lastTokenRefresh = Date.now();
          console.log("Session token refreshed successfully.");
        }
      } else if (res.status === 401 || res.status === 403) {
        console.warn("Token refresh rejected by backend. Forcing logout.");
        handleLogout();
      }
    } catch (err) {
      console.error("Failed to refresh token:", err);
    }
  }

  function handleLogout() {
    localStorage.removeItem('monitor_token');
    localStorage.removeItem('monitor_user');
    window.location.href = '/';
  }
</script>

{#if user}
<div class="h-screen w-full bg-slate-50 flex overflow-hidden font-sans text-slate-800">
  
  <!-- Mobile Menu Button -->
  <button 
    class="md:hidden fixed top-4 right-4 z-50 p-2 rounded-lg bg-white shadow-md border border-slate-200 text-slate-600"
    on:click={() => isMobileMenuOpen = !isMobileMenuOpen}
  >
    {#if isMobileMenuOpen}
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="3" y1="12" x2="21" y2="12"></line><line x1="3" y1="6" x2="21" y2="6"></line><line x1="3" y1="18" x2="21" y2="18"></line></svg>
    {/if}
  </button>

  <!-- Sidebar Overlay for Mobile -->
  {#if isMobileMenuOpen}
    <div 
      class="fixed inset-0 bg-slate-900/50 backdrop-blur-sm z-30 md:hidden"
      on:click={() => isMobileMenuOpen = false}
    ></div>
  {/if}

  <!-- Sidebar Area (Single-column layout) -->
  <div class="fixed md:static inset-y-0 left-0 flex h-full bg-[#fcfcfc] z-40 transition-transform duration-300 transform {isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'} border-r border-slate-200">
    
    <aside class="w-[260px] h-full flex flex-col shrink-0 relative z-10 px-4 py-6">
      <!-- Top Profile/Org block -->
      <button class="w-full flex items-center justify-between p-2 rounded-xl hover:bg-slate-100 transition-colors mb-6 text-left group">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-slate-900 flex shrink-0 items-center justify-center text-white shadow-sm">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.2 7.8l-7.7 7.7-4-4-5.7 5.7"/><path d="M15 7h6v6"/></svg>
          </div>
          <div class="flex flex-col overflow-hidden">
            <span class="font-bold text-slate-900 text-sm leading-tight truncate">Pantera Capital</span>
            <span class="text-xs text-slate-500 truncate">Workspace • Main</span>
          </div>
        </div>
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-slate-400 group-hover:text-slate-600"><polyline points="9 18 15 12 9 6"></polyline></svg>
      </button>

      <!-- Main Navigation -->
      <div class="flex-1 overflow-y-auto space-y-1 pr-1">
        <a href="/dashboard" on:click={() => isMobileMenuOpen = false} class="w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-sm font-medium transition-colors {currentPath === '/dashboard' ? 'bg-[#ecff82] text-slate-900' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}">
          <div class="flex items-center gap-3">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" class="{currentPath === '/dashboard' ? 'text-slate-900' : 'text-slate-500'}" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="9" rx="1"/><rect x="14" y="3" width="7" height="5" rx="1"/><rect x="14" y="12" width="7" height="9" rx="1"/><rect x="3" y="16" width="7" height="5" rx="1"/></svg>
            Project APIs
          </div>
        </a>

        <!-- Open APIs Link -->
        <a href="/dashboard/apis" on:click={() => isMobileMenuOpen = false} class="w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-sm font-medium transition-colors {currentPath === '/dashboard/apis' ? 'bg-[#ecff82] text-slate-900' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}">
          <div class="flex items-center gap-3">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" class="{currentPath === '/dashboard/apis' ? 'text-slate-900' : 'text-slate-500'}" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v20"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
            Open APIs
          </div>
          <span class="bg-slate-900 text-white text-[10px] font-bold px-2 py-0.5 rounded-md">8</span>
        </a>

        <!-- Status Live Link -->
        <a href="/dashboard/status" on:click={() => isMobileMenuOpen = false} class="w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-sm font-medium transition-colors {currentPath === '/dashboard/status' ? 'bg-[#ecff82] text-slate-900' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}">
          <div class="flex items-center gap-3">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" class="{currentPath === '/dashboard/status' ? 'text-slate-900' : 'text-slate-500'}" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>
            Status Live
          </div>
        </a>

        <!-- Notification Channels Link -->
        <a href={`/dashboard/projects/${$page.params.id || '1'}/notifications`} on:click={() => isMobileMenuOpen = false} class="w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-sm font-medium transition-colors {currentPath.includes('notifications') ? 'bg-amber-100 text-amber-900' : 'text-slate-600 hover:bg-amber-50 hover:text-amber-700'}">
          <div class="flex items-center gap-3">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" class="{currentPath.includes('notifications') ? 'text-amber-700' : 'text-amber-500'}" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>
            Alerts & Channels
          </div>
        </a>

        {#if user.role === 'admin'}
        <div class="pt-4 pb-1">
          <div class="px-3 flex items-center justify-between text-xs font-semibold text-slate-400 mb-1">
            <span>ADMINISTRATION</span>
          </div>
          <!-- Manage Users Link -->
          <a href="/dashboard/users" on:click={() => isMobileMenuOpen = false} class="w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-sm font-medium transition-colors {currentPath === '/dashboard/users' ? 'bg-[#ecff82] text-slate-900' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}">
            <div class="flex items-center gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" class="{currentPath === '/dashboard/users' ? 'text-slate-900' : 'text-slate-500'}" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
              Users & Roles
            </div>
          </a>
        </div>
        {/if}
      </div>

      <!-- Bottom User Section -->
      <div class="mt-auto pt-4 border-t border-slate-100 flex flex-col gap-4">
        <!-- Help Center -->
        <a href="#" class="w-full flex items-center gap-3 px-3 py-2 rounded-xl text-sm font-medium transition-colors text-slate-600 hover:bg-slate-100 hover:text-slate-900">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>
          Help center
        </a>

        <!-- User Info -->
        <div class="w-full flex flex-col pt-2">
          <button on:click={handleLogout} class="w-full flex items-center justify-between p-2 rounded-xl hover:bg-slate-100 transition-colors text-left group">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-full bg-slate-900 flex shrink-0 items-center justify-center text-white font-bold text-sm shadow-sm uppercase relative">
                {user?.email?.charAt(0) || 'U'}
                <span class="absolute bottom-0 right-0 w-3 h-3 bg-white text-blue-600 rounded-full flex items-center justify-center text-[8px] font-bold border border-slate-200">S</span>
              </div>
              <div class="flex flex-col overflow-hidden">
                <span class="font-bold text-slate-900 text-sm leading-tight truncate capitalize">{user?.email?.split('@')[0] || 'User'}</span>
                <span class="text-xs text-slate-500 truncate">{user?.email || 'user@example.com'}</span>
              </div>
            </div>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-slate-400 group-hover:text-red-500 transition-colors"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" x2="9" y1="12" y2="12"/></svg>
          </button>
          
          <!-- Progress Bar (Static mock) -->
          <div class="px-2 mt-4">
            <div class="flex items-center justify-between gap-3 text-[11px] font-bold text-slate-800 mb-1.5">
              <div class="flex-1 h-1.5 bg-slate-200 rounded-full overflow-hidden">
                <div class="h-full bg-emerald-500 rounded-full w-[70%]"></div>
              </div>
              <span>70%</span>
            </div>
            <span class="text-[11px] font-medium text-slate-500">Complete your profile</span>
          </div>
        </div>
      </div>
    </aside>
  </div>

  <!-- Main Content Area -->
  <main class="flex-1 h-screen overflow-y-auto relative w-full pt-16 md:pt-0">
    <!-- Header decoration -->
    <div class="fixed top-0 left-0 w-full h-64 bg-gradient-to-b from-blue-50/80 to-transparent pointer-events-none -z-10"></div>
    
    <div class="min-h-full p-6 md:p-8 max-w-7xl mx-auto w-full">
      <slot />
    </div>
  </main>
  
</div>
{/if}
