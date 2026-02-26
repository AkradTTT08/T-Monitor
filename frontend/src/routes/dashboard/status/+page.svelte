<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  
  let logs: any[] = [];
  let isLoading = true;
  let summary = { total: 0, up: 0, down: 0 };
  let refreshInterval: any;

  onMount(async () => {
    await fetchLogs();
    // Real-time aspect: Poll every 10 seconds for new health checks
    refreshInterval = setInterval(fetchLogs, 10000);
  });

  onDestroy(() => {
    if (refreshInterval) clearInterval(refreshInterval);
  });

  async function fetchLogs() {
    try {
      const token = localStorage.getItem('monitor_token');
      const res = await fetch('http://localhost:5273/api/v1/logs', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      
      if (res.ok) {
        logs = await res.json();
        
        // Calculate basic status based on latest logs for each unique API
        const uniqueApis = new Map();
        logs.forEach(log => {
          if (!uniqueApis.has(log.api_id)) {
            uniqueApis.set(log.api_id, log.is_success);
          }
        });
        
        let up = 0, down = 0;
        uniqueApis.forEach((isSuccess) => {
          if (isSuccess) up++; else down++;
        });
        
        summary = { total: uniqueApis.size, up, down };
      }
    } catch (err) {
      console.error(err);
    } finally {
      isLoading = false;
    }
  }

  function formatRelativeTime(dateString: string) {
    const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
    const diff = new Date(dateString).getTime() - new Date().getTime();
    
    // Convert to seconds, minutes, or hours appropriately
    const diffSecs = Math.round(diff / 1000);
    if (Math.abs(diffSecs) < 60) return rtf.format(diffSecs, 'second');
    
    const diffMins = Math.round(diffSecs / 60);
    if (Math.abs(diffMins) < 60) return rtf.format(diffMins, 'minute');
    
    const diffHours = Math.round(diffMins / 60);
    if (Math.abs(diffHours) < 24) return rtf.format(diffHours, 'hour');
    
    const diffDays = Math.round(diffHours / 24);
    return rtf.format(diffDays, 'day');
  }
</script>

<div class="fade-in max-w-6xl mx-auto w-full overflow-hidden">
  <div class="flex flex-col md:flex-row md:items-end justify-between gap-4 mb-4 md:mb-8">
    <div>
      <h1 class="text-2xl md:text-3xl font-bold text-slate-900 tracking-tight">API Status Console</h1>
      <p class="text-sm md:text-base text-slate-500 mt-2">Real-time health check logs and analytics.</p>
    </div>
    <div class="flex items-center gap-2 text-xs md:text-sm font-medium text-slate-500 bg-white border border-slate-200 py-1.5 px-3 md:py-2 md:px-4 rounded-full shadow-sm w-fit self-start md:self-auto">
      <div class="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div>
      Live Polling Active
    </div>
  </div>

  <!-- Summary Cards -->
  <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 md:gap-6 mb-8">
    <div class="bg-white rounded-2xl border border-slate-200 p-5 md:p-6 shadow-sm relative overflow-hidden">
      <div class="absolute right-0 top-0 w-24 h-24 bg-blue-500 opacity-5 rounded-bl-full pointer-events-none"></div>
      <p class="text-slate-500 font-medium text-xs md:text-sm mb-1 uppercase tracking-wider">Monitored Endpoints</p>
      <h2 class="text-3xl md:text-4xl font-black text-slate-800">{summary.total}</h2>
    </div>
    
    <div class="bg-white rounded-2xl border border-slate-200 p-5 md:p-6 shadow-sm relative overflow-hidden">
      <div class="absolute right-0 top-0 w-24 h-24 bg-green-500 opacity-5 rounded-bl-full pointer-events-none"></div>
      <p class="text-slate-500 font-medium text-xs md:text-sm mb-1 uppercase tracking-wider">Operational (UP)</p>
      <h2 class="text-3xl md:text-4xl font-black text-green-500">{summary.up}</h2>
    </div>
    
    <div class="bg-white rounded-2xl border border-slate-200 p-5 md:p-6 shadow-sm relative overflow-hidden">
      <div class="absolute right-0 top-0 w-24 h-24 bg-red-500 opacity-5 rounded-bl-full pointer-events-none"></div>
      <p class="text-slate-500 font-medium text-xs md:text-sm mb-1 uppercase tracking-wider">Failing (DOWN)</p>
      <h2 class="text-3xl md:text-4xl font-black text-red-500">{summary.down}</h2>
    </div>
  </div>

  <!-- Log Table -->
  <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden w-full">
    {#if isLoading && logs.length === 0}
      <div class="flex justify-center p-12">
        <svg class="animate-spin h-8 w-8 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
      </div>
    {:else if logs.length === 0}
      <div class="p-8 md:p-12 text-center">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-slate-50 mb-4">
          <svg xmlns="http://www.w3.org/2000/svg" class="text-slate-400 h-8 w-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.2 8.4c.5.38.8.97.8 1.6v10a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V10a2 2 0 0 1 .8-1.6l8-6a2 2 0 0 1 2.4 0l8 6Z"></path><path d="m22 10-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 10"></path></svg>
        </div>
        <h3 class="text-lg font-bold text-slate-800">No Logs Recorded</h3>
        <p class="text-slate-500 mt-1 text-sm">Waiting for the background worker to execute health checks.</p>
      </div>
    {:else}
      <div class="overflow-x-auto w-full">
        <table class="w-full text-left border-collapse whitespace-nowrap min-w-[700px]">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200 text-xs font-bold text-slate-500 uppercase tracking-wider">
              <th class="p-3 md:p-4 pl-4 md:pl-6">Status</th>
              <th class="p-3 md:p-4">API ID</th>
              <th class="p-3 md:p-4">Check Time</th>
              <th class="p-3 md:p-4">Response Time</th>
              <th class="p-3 md:p-4">Status Code</th>
              <th class="p-3 md:p-4 pr-4 md:pr-6">Error Detail</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 text-sm">
            {#each logs as log}
              <tr class="hover:bg-slate-50 transition-colors">
                <td class="p-3 md:p-4 pl-4 md:pl-6">
                  {#if log.is_success}
                    <div class="flex items-center gap-2 text-green-600 font-bold">
                      <div class="w-2 h-2 rounded-full bg-green-500"></div> UP
                    </div>
                  {:else}
                     <div class="flex items-center gap-2 text-red-600 font-bold">
                      <div class="w-2 h-2 rounded-full bg-red-500"></div> DOWN
                    </div>
                  {/if}
                </td>
                <td class="p-3 md:p-4 font-mono font-medium text-slate-700">API-{log.api_id}</td>
                <td class="p-3 md:p-4 text-slate-500">{formatRelativeTime(log.checked_at)}</td>
                <td class="p-3 md:p-4">
                  <span class="font-mono {log.response_time > 1000 ? 'text-orange-500 font-semibold' : 'text-slate-600'}">
                    {log.response_time}ms
                  </span>
                </td>
                <td class="p-3 md:p-4">
                  <span class="inline-flex items-center justify-center px-2 py-1 rounded text-xs font-bold {log.status_code >= 200 && log.status_code < 300 ? 'bg-green-100 text-green-700' : log.status_code > 0 ? 'bg-red-100 text-red-700' : 'bg-slate-200 text-slate-700'}">
                    {log.status_code || 'ERR'}
                  </span>
                </td>
                <td class="p-3 md:p-4 pr-4 md:pr-6 truncate max-w-[150px] md:max-w-xs text-slate-500" title={log.error_message}>
                  {log.error_message || '-'}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
