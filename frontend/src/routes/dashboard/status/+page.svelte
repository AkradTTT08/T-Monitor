<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Chart from 'chart.js/auto';
  
  let logs: any[] = [];
  let isLoading = true;
  let summary = { total: 0, up: 0, down: 0 };
  let refreshInterval: any;

  // Chart State
  let chartCanvas: HTMLCanvasElement;
  let statusChart: Chart | null = null;

  // Filter State
  let searchQuery = '';
  let statusFilter = 'ALL'; // ALL, UP, DOWN
  
  // Derived state for filtered logs
  $: filteredLogs = logs.filter(log => {
    // 1. Check Status Filter
    if (statusFilter === 'UP' && !log.is_success) return false;
    if (statusFilter === 'DOWN' && log.is_success) return false;
    
    // 2. Check Search Query
    if (searchQuery.trim() !== '') {
      const q = searchQuery.toLowerCase();
      const apiName = (log.api?.name || `API-${log.api_id}`).toLowerCase();
      const errorMsg = (log.error_message || '').toLowerCase();
      const statusCode = (log.status_code || '').toString();
      
      if (!apiName.includes(q) && !errorMsg.includes(q) && !statusCode.includes(q)) {
        return false;
      }
    }
    
    return true;
  });

  // Pagination State
  let currentPage = 1;
  const itemsPerPage = 10;
  $: totalPages = Math.ceil(filteredLogs.length / itemsPerPage);
  $: paginatedLogs = filteredLogs.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage);

  // Reset page when filters change
  $: if (searchQuery !== undefined || statusFilter !== undefined) {
    currentPage = 1;
  }

  // --- Chart.js Rendering Logic --- //
  $: if (chartCanvas && filteredLogs.length >= 0) {
    updateChart();
  }

  function updateChart() {
    if (!chartCanvas) return;
    
    // We want the chart to flow chronologically (oldest left, newest right)
    // The logs from DB are newest-first, so limit to latest 50 and reverse.
    const chartData = [...filteredLogs].slice(0, 50).reverse();
    
    const labels = chartData.map(log => 
      new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(new Date(log.checked_at))
    );
    
    // Background colors: Green if UP, Red if DOWN
    const backgroundColors = chartData.map(log => 
      log.is_success ? 'rgba(34, 197, 94, 0.8)' : 'rgba(239, 68, 68, 0.8)'
    );

    const dataPoints = chartData.map(log => log.response_time);

    // Custom Tooltip Data
    const tooltipLabels = chartData.map(log => ({
      name: log.api?.name || `API-${log.api_id}`,
      status: log.status_code || 'ERR',
      error: log.error_message || '-'
    }));

    if (statusChart) {
      statusChart.data.labels = labels;
      statusChart.data.datasets[0].data = dataPoints;
      statusChart.data.datasets[0].backgroundColor = backgroundColors;
      // Store custom data for tooltips
      (statusChart.data.datasets[0] as any).customData = tooltipLabels;
      statusChart.update();
    } else {
      statusChart = new Chart(chartCanvas, {
        type: 'bar',
        data: {
          labels: labels,
          datasets: [{
            label: 'Response Time (ms)',
            data: dataPoints,
            backgroundColor: backgroundColors,
            borderRadius: 4,
            borderSkipped: false,
            barPercentage: 0.8,
            categoryPercentage: 0.9,
            customData: tooltipLabels // Attach our custom tooltip object
          } as any]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: {
            legend: { display: false },
            tooltip: {
              backgroundColor: 'rgba(15, 23, 42, 0.9)',
              titleFont: { size: 13, family: "'Inter', sans-serif", weight: 'bold' },
              bodyFont: { size: 12, family: "'Inter', sans-serif" },
              padding: 12,
              cornerRadius: 8,
              callbacks: {
                title: (context) => {
                  const dataIndex = context[0].dataIndex;
                  const name = context[0].dataset.customData[dataIndex].name;
                  return `${name} • ${context[0].label}`;
                },
                afterTitle: (context) => {
                  const dataIndex = context[0].dataIndex;
                  const status = context[0].dataset.customData[dataIndex].status;
                  return `Status Code: ${status}`;
                },
                label: (context) => {
                  return `Response: ${context.raw} ms`;
                },
                afterLabel: (context) => {
                  const dataIndex = context.dataIndex;
                  const error = context.dataset.customData[dataIndex].error;
                  if (error && error !== '-') {
                    return `\nError: ${error}`;
                  }
                  return '';
                }
              }
            }
          },
          scales: {
            y: {
              beginAtZero: true,
              grid: { color: 'rgba(226, 232, 240, 0.5)' },
              border: { dash: [4, 4] },
              ticks: {
                font: { family: "'Inter', sans-serif", size: 11 },
                color: '#64748b',
                callback: function(value) {
                  // Format as seconds similar to Postman if it's over 1000
                  if (Number(value) >= 1000) {
                    return (Number(value) / 1000).toFixed(1) + ' s';
                  }
                  return value + ' ms';
                }
              },
              title: {
                display: true,
                text: 'Response Time',
                color: '#94a3b8',
                font: { size: 10, weight: 'bold', family: "'Inter', sans-serif" }
              }
            },
            x: {
              grid: { display: false },
              ticks: {
                maxTicksLimit: 12,
                maxRotation: 45,
                minRotation: 0,
                font: { family: "'Inter', sans-serif", size: 10 },
                color: '#94a3b8'
              }
            }
          },
          animation: { duration: 400 }
        }
      });
    }
  }

  onMount(async () => {
    await fetchLogs();
    // Real-time aspect: Poll every 10 seconds for new health checks
    refreshInterval = setInterval(fetchLogs, 10000);
  });

  onDestroy(() => {
    if (refreshInterval) clearInterval(refreshInterval);
    if (statusChart) statusChart.destroy();
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

  function formatDateTime(dateString: string) {
    return new Intl.DateTimeFormat('en-GB', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
    }).format(new Date(dateString));
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

  <!-- Performance Chart -->
  <div class="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm mb-6">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-sm font-semibold text-slate-800 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-blue-500"><path d="M3 3v18h18"/><path d="m19 9-5 5-4-4-3 3"/></svg>
        Recent Activity (Run Summary)
      </h3>
      <span class="text-xs font-medium text-slate-500 bg-slate-100 py-1 px-2.5 rounded-md border border-slate-200">Latest 50 Hits</span>
    </div>
    
    <div class="h-48 w-full relative">
      {#if isLoading && logs.length === 0}
        <div class="absolute inset-0 flex items-center justify-center">
          <svg class="animate-spin h-6 w-6 text-slate-300" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
        </div>
      {/if}
      <canvas bind:this={chartCanvas} class:opacity-0={isLoading && logs.length === 0}></canvas>
    </div>
  </div>

  <!-- Filters & Search -->
  <div class="bg-white p-4 rounded-t-2xl border-x border-t border-slate-200 flex flex-col md:flex-row gap-4 items-center justify-between mt-4">
    <div class="relative w-full md:max-w-md lg:max-w-lg shrink-0">
      <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
        <svg class="h-5 w-5 text-slate-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z" clip-rule="evenodd" /></svg>
      </div>
      <input type="text" bind:value={searchQuery} placeholder="Search by endpoint name, status code, error detail..." class="block w-full pl-10 pr-10 py-2.5 border border-slate-200 rounded-xl leading-5 bg-slate-50 placeholder-slate-400 focus:outline-none focus:bg-white focus:ring-2 focus:ring-blue-500/30 focus:border-blue-500 sm:text-sm transition-all duration-200" />
      {#if searchQuery}
        <button class="absolute inset-y-0 right-0 pr-3 flex items-center text-slate-400 hover:text-slate-600 transition-colors" on:click={() => searchQuery = ''}>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 bg-slate-200 rounded-full p-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      {/if}
    </div>
    
    <div class="flex gap-2 w-full md:w-auto overflow-x-auto pb-1 md:pb-0 hide-scrollbar shrink-0">
      <button on:click={() => statusFilter = 'ALL'} class="px-5 py-2.5 text-sm font-semibold rounded-xl border transition-all whitespace-nowrap {statusFilter === 'ALL' ? 'bg-slate-800 text-white border-slate-800 shadow-md shadow-slate-800/20' : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50 hover:border-slate-300'}">All Logs</button>
      <button on:click={() => statusFilter = 'UP'} class="px-5 py-2.5 text-sm font-semibold rounded-xl border transition-all whitespace-nowrap {statusFilter === 'UP' ? 'bg-emerald-50 text-emerald-700 border-emerald-300 shadow-sm shadow-emerald-500/10' : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50 hover:border-slate-300'} flex items-center gap-2"><div class="w-1.5 h-1.5 rounded-full bg-emerald-500"></div> Up</button>
      <button on:click={() => statusFilter = 'DOWN'} class="px-5 py-2.5 text-sm font-semibold rounded-xl border transition-all whitespace-nowrap {statusFilter === 'DOWN' ? 'bg-rose-50 text-rose-700 border-rose-300 shadow-sm shadow-rose-500/10' : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50 hover:border-slate-300'} flex items-center gap-2"><div class="w-1.5 h-1.5 rounded-full bg-rose-500"></div> Down</button>
    </div>
  </div>

  <!-- Log Table -->
  <div class="bg-white rounded-b-2xl border border-t-0 border-slate-200 shadow-sm overflow-hidden w-full relative -mt-px">
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
    {:else if filteredLogs.length === 0}
      <div class="p-8 md:p-12 text-center py-24">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-blue-50 mb-4 ring-8 ring-blue-50/50">
          <svg xmlns="http://www.w3.org/2000/svg" class="text-blue-500 h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
        </div>
        <h3 class="text-xl font-bold text-slate-800 tracking-tight">No matching logs found</h3>
        <p class="text-slate-500 mt-2 text-sm max-w-sm mx-auto">We couldn't find any health check logs matching your search terms or filter criteria.</p>
        <button on:click={() => { searchQuery = ''; statusFilter = 'ALL'; }} class="mt-6 px-5 py-2.5 text-sm font-semibold text-blue-700 bg-blue-50 rounded-xl hover:bg-blue-100 transition-colors shadow-sm inline-flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.3"/></svg>
          Clear all filters
        </button>
      </div>
    {:else}
      <div class="overflow-x-auto w-full">
        <table class="w-full text-left border-collapse whitespace-nowrap min-w-[800px]">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200 text-xs font-bold text-slate-500 uppercase tracking-wider">
              <th class="p-3 md:p-4 pl-4 md:pl-6">Status</th>
              <th class="p-3 md:p-4">Endpoint Name</th>
              <th class="p-3 md:p-4">Check Time</th>
              <th class="p-3 md:p-4">Schedule</th>
              <th class="p-3 md:p-4">Response Time</th>
              <th class="p-3 md:p-4">Status Code</th>
              <th class="p-3 md:p-4 pr-4 md:pr-6">Error Detail</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 text-sm">
            {#each paginatedLogs as log}
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
                <td class="p-3 md:p-4 text-slate-800 font-semibold truncate max-w-[200px]" title={log.api?.name || 'Unknown API'}>
                  {log.api?.name || `API-${log.api_id}`}
                </td>
                <td class="p-3 md:p-4 text-slate-500 flex flex-col justify-center">
                  <span class="font-medium text-slate-700">{formatDateTime(log.checked_at)}</span>
                  <span class="text-xs text-slate-400">{formatRelativeTime(log.checked_at)}</span>
                </td>
                <td class="p-3 md:p-4">
                  <span class="text-sm font-medium {log.api?.interval ? 'text-blue-600' : 'text-slate-400'}">
                    {#if log.api?.interval}
                      {#if log.api.interval < 60}
                        Every {log.api.interval} sec
                      {:else if log.api.interval < 3600}
                        Every {Math.round(log.api.interval / 60)} min
                      {:else}
                        Every {Math.round(log.api.interval / 3600)} hr
                      {/if}
                    {:else}
                      -
                    {/if}
                  </span>
                </td>
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
        
        <!-- Pagination Controls -->
        {#if totalPages > 1}
          <div class="flex items-center justify-between border-t border-slate-200 px-4 py-3 bg-white sm:px-6 rounded-b-2xl">
            <div class="flex flex-1 justify-between sm:hidden">
              <button on:click={() => currentPage > 1 && currentPage--} disabled={currentPage === 1} class="relative inline-flex items-center rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50">Previous</button>
              <button on:click={() => currentPage < totalPages && currentPage++} disabled={currentPage === totalPages} class="relative ml-3 inline-flex items-center rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50">Next</button>
            </div>
            <div class="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
              <div>
                <p class="text-sm text-slate-700">
                  Showing
                  <span class="font-medium">{(currentPage - 1) * itemsPerPage + 1}</span>
                  to
                  <span class="font-medium">{Math.min(currentPage * itemsPerPage, logs.length)}</span>
                  of
                  <span class="font-medium">{logs.length}</span>
                  results
                </p>
              </div>
              <div>
                <nav class="isolate inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
                  <button 
                    on:click={() => currentPage > 1 && currentPage--}
                    disabled={currentPage === 1}
                    class="relative inline-flex items-center rounded-l-md px-2 py-2 text-slate-400 ring-1 ring-inset ring-slate-300 hover:bg-slate-50 focus:z-20 focus:outline-offset-0 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <span class="sr-only">Previous</span>
                    <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                      <path fill-rule="evenodd" d="M12.79 5.23a.75.75 0 01-.02 1.06L8.832 10l3.938 3.71a.75.75 0 11-1.04 1.08l-4.5-4.25a.75.75 0 010-1.08l4.5-4.25a.75.75 0 011.06.02z" clip-rule="evenodd" />
                    </svg>
                  </button>
                  
                  {#each Array(totalPages) as _, i}
                    <!-- Show max 5 page numbers -->
                    {#if i + 1 === 1 || i + 1 === totalPages || (i + 1 >= currentPage - 1 && i + 1 <= currentPage + 1)}
                      <button 
                        on:click={() => currentPage = i + 1}
                        class="relative inline-flex items-center px-4 py-2 text-sm font-semibold {currentPage === i + 1 ? 'z-10 bg-blue-600 text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600' : 'text-slate-900 ring-1 ring-inset ring-slate-300 hover:bg-slate-50 focus:z-20 focus:outline-offset-0'}"
                      >
                        {i + 1}
                      </button>
                    {:else if (i + 1 === currentPage - 2 && currentPage > 3) || (i + 1 === currentPage + 2 && currentPage < totalPages - 2)}
                      <span class="relative inline-flex items-center px-4 py-2 text-sm font-semibold text-slate-700 ring-1 ring-inset ring-slate-300 focus:outline-offset-0">
                        ...
                      </span>
                    {/if}
                  {/each}
                  
                  <button 
                    on:click={() => currentPage < totalPages && currentPage++}
                    disabled={currentPage === totalPages}
                    class="relative inline-flex items-center rounded-r-md px-2 py-2 text-slate-400 ring-1 ring-inset ring-slate-300 hover:bg-slate-50 focus:z-20 focus:outline-offset-0 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <span class="sr-only">Next</span>
                    <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                      <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
                    </svg>
                  </button>
                </nav>
              </div>
            </div>
          </div>
        {/if}
        
      </div>
    {/if}
  </div>
</div>
<style>
  /* Hide scrollbar for Chrome, Safari and Opera */
  .hide-scrollbar::-webkit-scrollbar {
    display: none;
  }
  /* Hide scrollbar for IE, Edge and Firefox */
  .hide-scrollbar {
    -ms-overflow-style: none;  /* IE and Edge */
    scrollbar-width: none;  /* Firefox */
  }
</style>
