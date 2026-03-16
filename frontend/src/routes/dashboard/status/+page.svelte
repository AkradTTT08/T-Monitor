<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { page } from "$app/stores";
  import Chart from "chart.js/auto";

  import { API_BASE_URL } from "$lib/config";

  let logs: any[] = [];
  let isLoading = true;
  let summary = { total: 0, up: 0, down: 0 };
  let refreshInterval: any;
  let selectedProjectId = "";

  // Chart State
  let chartCanvas: HTMLCanvasElement;
  let statusChart: Chart | null = null;

  // Filter State
  let searchQuery = "";
  let statusFilter = "ALL"; // ALL, UP, DOWN

  // Modal State
  let showLogModal = false;
  let selectedLog: any = null;

  // Derived state for filtered logs
  $: filteredLogs = logs.filter((log) => {
    // 1. Check Status Filter
    if (statusFilter === "UP" && !log.is_success) return false;
    if (statusFilter === "DOWN" && log.is_success) return false;

    // 2. Check Search Query
    if (searchQuery.trim() !== "") {
      const q = searchQuery.toLowerCase();
      const apiName = (log.api?.name || `API-${log.api_id}`).toLowerCase();
      const errorMsg = (log.error_message || "").toLowerCase();
      const statusCode = (log.status_code || "").toString();

      if (
        !apiName.includes(q) &&
        !errorMsg.includes(q) &&
        !statusCode.includes(q)
      ) {
        return false;
      }
    }

    return true;
  });

  // Pagination State
  let currentPage = 1;
  const itemsPerPage = 10;
  $: totalPages = Math.ceil(filteredLogs.length / itemsPerPage);
  $: paginatedLogs = filteredLogs.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage,
  );

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

    const labels = chartData.map((log) =>
      new Intl.DateTimeFormat("en-GB", {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      }).format(new Date(log.checked_at)),
    );

    // Background colors: Green if UP, Red if DOWN
    const backgroundColors = chartData.map((log) =>
      log.is_success ? "rgba(34, 197, 94, 0.8)" : "rgba(239, 68, 68, 0.8)",
    );

    const dataPoints = chartData.map((log) => log.response_time);

    // Custom Tooltip Data
    const tooltipLabels = chartData.map((log) => ({
      name: log.api?.name || `API-${log.api_id}`,
      status: log.status_code || "ERR",
      error: log.error_message || "-",
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
        type: "bar",
        data: {
          labels: labels,
          datasets: [
            {
              label: "Response Time (ms)",
              data: dataPoints,
              backgroundColor: backgroundColors,
              borderRadius: 4,
              borderSkipped: false,
              barPercentage: 0.8,
              categoryPercentage: 0.9,
              customData: tooltipLabels, // Attach our custom tooltip object
            } as any,
          ],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: {
            legend: { display: false },
            tooltip: {
              backgroundColor: "rgba(15, 23, 42, 0.9)",
              titleFont: {
                size: 13,
                family: "'Inter', sans-serif",
                weight: "bold",
              },
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
                  const status =
                    context[0].dataset.customData[dataIndex].status;
                  return `Status Code: ${status}`;
                },
                label: (context) => {
                  return `Response: ${context.raw} ms`;
                },
                afterLabel: (context) => {
                  const dataIndex = context.dataIndex;
                  const error = context.dataset.customData[dataIndex].error;
                  if (error && error !== "-") {
                    return `\nError: ${error}`;
                  }
                  return "";
                },
              },
            },
          },
          scales: {
            y: {
              beginAtZero: true,
              grid: { color: "rgba(226, 232, 240, 0.5)" },
              border: { dash: [4, 4] },
              ticks: {
                font: { family: "'Inter', sans-serif", size: 11 },
                color: "#64748b",
                callback: function (value) {
                  // Format as seconds similar to Postman if it's over 1000
                  if (Number(value) >= 1000) {
                    return (Number(value) / 1000).toFixed(1) + " s";
                  }
                  return value + " ms";
                },
              },
              title: {
                display: true,
                text: "Response Time",
                color: "#94a3b8",
                font: {
                  size: 10,
                  weight: "bold",
                  family: "'Inter', sans-serif",
                },
              },
            },
            x: {
              grid: { display: false },
              ticks: {
                maxTicksLimit: 12,
                maxRotation: 45,
                minRotation: 0,
                font: { family: "'Inter', sans-serif", size: 10 },
                color: "#94a3b8",
              },
            },
          },
          animation: { duration: 400 },
        },
      });
    }
  }

  onMount(async () => {
    // Read project_id from URL query param
    selectedProjectId =
      $page.url.searchParams.get("project_id") ||
      localStorage.getItem("monitor_selected_project") ||
      "";
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
      const token = localStorage.getItem("monitor_token");
      let url = `${API_BASE_URL}/api/v1/logs`;
      if (selectedProjectId) {
        url += `?project_id=${selectedProjectId}`;
      }
      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        logs = await res.json();

        // Calculate basic status based on latest logs for each unique API
        const uniqueApis = new Map();
        logs.forEach((log) => {
          if (!uniqueApis.has(log.api_id)) {
            uniqueApis.set(log.api_id, log.is_success);
          }
        });

        let up = 0,
          down = 0;
        uniqueApis.forEach((isSuccess) => {
          if (isSuccess) up++;
          else down++;
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
    const rtf = new Intl.RelativeTimeFormat("en", { numeric: "auto" });
    const diff = new Date(dateString).getTime() - new Date().getTime();

    // Convert to seconds, minutes, or hours appropriately
    const diffSecs = Math.round(diff / 1000);
    if (Math.abs(diffSecs) < 60) return rtf.format(diffSecs, "second");

    const diffMins = Math.round(diffSecs / 60);
    if (Math.abs(diffMins) < 60) return rtf.format(diffMins, "minute");

    const diffHours = Math.round(diffMins / 60);
    if (Math.abs(diffHours) < 24) return rtf.format(diffHours, "hour");

    const diffDays = Math.round(diffHours / 24);
    return rtf.format(diffDays, "day");
  }

  function formatDateTime(dateString: string) {
    return new Intl.DateTimeFormat("en-GB", {
      day: "2-digit",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    }).format(new Date(dateString));
  }

  function viewLogDetails(log: any) {
    selectedLog = log;
    showLogModal = true;
  }
</script>

<div class="fade-in max-w-6xl mx-auto w-full overflow-hidden">
  <div
    class="flex flex-col md:flex-row md:items-end justify-between gap-4 mb-8"
  >
    <div>
      <h1
        class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase"
      >
        API_STATUS_CONSOLE
      </h1>
      <p class="text-cyan-500/80 mt-2 font-mono text-sm tracking-wide">
        REAL-TIME DIAGNOSTICS AND HEALTH ANALYTICS.
      </p>
    </div>
    <div
      class="flex items-center gap-2 text-xs md:text-sm font-bold text-emerald-400 bg-slate-900 border border-emerald-500/30 py-1.5 px-3 md:py-2 md:px-4 rounded-full shadow-[0_0_15px_rgba(52,211,153,0.2)] w-fit self-start md:self-auto font-mono tracking-wider"
    >
      <div
        class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse shadow-[0_0_8px_rgba(52,211,153,0.8)]"
      ></div>
      LIVE_POLLING_ACTIVE
    </div>
  </div>

  <!-- Summary Cards -->
  <div
    class="grid grid-cols-1 sm:grid-cols-3 gap-4 md:gap-6 mb-8 relative z-10"
  >
    <div
      class="bg-slate-800/40 backdrop-blur-xl rounded-3xl border border-slate-700/50 p-5 md:p-6 shadow-[0_8px_30px_rgb(0,0,0,0.5)] relative overflow-hidden group"
    >
      <div
        class="absolute inset-0 bg-gradient-to-br from-cyan-900/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"
      ></div>
      <div
        class="absolute right-0 top-0 w-24 h-24 bg-cyan-500 opacity-10 rounded-bl-full pointer-events-none group-hover:scale-110 transition-transform duration-500"
      ></div>
      <p
        class="text-cyan-500/80 font-bold text-xs md:text-sm mb-1 uppercase tracking-widest font-mono"
      >
        MONITORED_ENDPOINTS
      </p>
      <h2
        class="text-3xl md:text-4xl font-black text-cyan-50 drop-shadow-[0_0_15px_rgba(6,182,212,0.3)]"
      >
        {summary.total}
      </h2>
    </div>

    <div
      class="bg-slate-800/40 backdrop-blur-xl rounded-3xl border border-slate-700/50 p-5 md:p-6 shadow-[0_8px_30px_rgb(0,0,0,0.5)] relative overflow-hidden group"
    >
      <div
        class="absolute inset-0 bg-gradient-to-br from-emerald-900/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"
      ></div>
      <div
        class="absolute right-0 top-0 w-24 h-24 bg-emerald-500 opacity-10 rounded-bl-full pointer-events-none group-hover:scale-110 transition-transform duration-500"
      ></div>
      <p
        class="text-emerald-500/80 font-bold text-xs md:text-sm mb-1 uppercase tracking-widest font-mono"
      >
        OPERATIONAL [UP]
      </p>
      <h2
        class="text-3xl md:text-4xl font-black text-emerald-400 drop-shadow-[0_0_15px_rgba(52,211,153,0.3)]"
      >
        {summary.up}
      </h2>
    </div>

    <div
      class="bg-slate-800/40 backdrop-blur-xl rounded-3xl border border-slate-700/50 p-5 md:p-6 shadow-[0_8px_30px_rgb(0,0,0,0.5)] relative overflow-hidden group"
    >
      <div
        class="absolute inset-0 bg-gradient-to-br from-red-900/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"
      ></div>
      <div
        class="absolute right-0 top-0 w-24 h-24 bg-red-500 opacity-10 rounded-bl-full pointer-events-none group-hover:scale-110 transition-transform duration-500"
      ></div>
      <p
        class="text-red-500/80 font-bold text-xs md:text-sm mb-1 uppercase tracking-widest font-mono"
      >
        FAILING [DOWN]
      </p>
      <h2
        class="text-3xl md:text-4xl font-black text-red-500 drop-shadow-[0_0_15px_rgba(239,68,68,0.3)]"
      >
        {summary.down}
      </h2>
    </div>
  </div>

  <!-- Performance Chart -->
  <div
    class="bg-slate-800/40 backdrop-blur-xl p-5 md:p-6 rounded-3xl border border-slate-700/50 shadow-[0_8px_30px_rgb(0,0,0,0.5)] mb-8 relative"
  >
    <div class="flex items-center justify-between mb-6">
      <h3
        class="text-sm font-bold text-cyan-50 font-mono tracking-widest flex items-center gap-2"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="text-cyan-400"
          ><path d="M3 3v18h18" /><path d="m19 9-5 5-4-4-3 3" /></svg
        >
        RECENT_ACTIVITY_LOG
      </h3>
      <span
        class="text-[10px] font-bold text-cyan-400 bg-slate-900 border border-slate-700 py-1.5 px-3 rounded-md uppercase tracking-widest font-mono"
        >LATEST 50 EVENT_CYCLES</span
      >
    </div>

    <div class="h-48 w-full relative">
      {#if isLoading && logs.length === 0}
        <div class="absolute inset-0 flex items-center justify-center">
          <svg
            class="animate-spin h-6 w-6 text-cyan-500"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            ><circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle><path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path></svg
          >
        </div>
      {/if}
      <canvas
        bind:this={chartCanvas}
        class:opacity-0={isLoading && logs.length === 0}
      ></canvas>
    </div>
  </div>

  <!-- Filters & Search -->
  <div
    class="bg-slate-900/60 backdrop-blur-md p-4 rounded-t-3xl border-x border-t border-slate-700/50 flex flex-col xl:flex-row gap-4 items-center justify-between mt-4 relative z-10"
  >
    <div class="relative w-full xl:max-w-lg shrink-0">
      <div
        class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none"
      >
        <svg
          class="h-5 w-5 text-slate-500"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
          ><path
            fill-rule="evenodd"
            d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z"
            clip-rule="evenodd"
          /></svg
        >
      </div>
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="SEARCH BY ENDPOINT_NAME, HTTP_STATUS, ERR_DETAIL..."
        class="block w-full pl-11 pr-10 py-3 border border-slate-700/80 rounded-xl leading-5 bg-slate-800 text-cyan-50 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/30 focus:border-cyan-500 sm:text-sm transition-all duration-200 font-mono tracking-wide"
      />
      {#if searchQuery}
        <button
          class="absolute inset-y-0 right-0 pr-3 flex items-center text-slate-400 hover:text-slate-600 transition-colors"
          on:click={() => (searchQuery = "")}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-4 w-4 bg-slate-200 rounded-full p-0.5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            ><path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M6 18L18 6M6 6l12 12"
            /></svg
          >
        </button>
      {/if}
    </div>

    <div
      class="flex gap-2 w-full xl:w-auto overflow-x-auto pb-1 xl:pb-0 hide-scrollbar shrink-0"
    >
      <button
        on:click={() => (statusFilter = "ALL")}
        class="px-6 py-3 text-xs tracking-widest font-mono font-bold rounded-xl border transition-all whitespace-nowrap {statusFilter ===
        'ALL'
          ? 'bg-cyan-900 text-cyan-50 border-cyan-500/50 shadow-[0_0_15px_rgba(6,182,212,0.3)]'
          : 'bg-slate-800/80 text-slate-400 border-slate-700 hover:bg-slate-700 hover:text-cyan-300'}"
        >ALL_LOGS</button
      >
      <button
        on:click={() => (statusFilter = "UP")}
        class="px-6 py-3 text-xs tracking-widest font-mono font-bold rounded-xl border transition-all whitespace-nowrap {statusFilter ===
        'UP'
          ? 'bg-emerald-900/80 text-emerald-400 border-emerald-500/50 shadow-[0_0_15px_rgba(52,211,153,0.3)]'
          : 'bg-slate-800/80 text-slate-400 border-slate-700 hover:bg-slate-700 hover:text-emerald-300'} flex items-center gap-2"
        ><div
          class="w-1.5 h-1.5 rounded-full bg-emerald-500 shadow-[0_0_5px_rgba(52,211,153,0.8)]"
        ></div>
        UP</button
      >
      <button
        on:click={() => (statusFilter = "DOWN")}
        class="px-6 py-3 text-xs tracking-widest font-mono font-bold rounded-xl border transition-all whitespace-nowrap {statusFilter ===
        'DOWN'
          ? 'bg-red-900/80 text-red-400 border-red-500/50 shadow-[0_0_15px_rgba(239,68,68,0.3)]'
          : 'bg-slate-800/80 text-slate-400 border-slate-700 hover:bg-slate-700 hover:text-red-300'} flex items-center gap-2"
        ><div
          class="w-1.5 h-1.5 rounded-full bg-red-500 shadow-[0_0_5px_rgba(239,68,68,0.8)]"
        ></div>
        DOWN</button
      >
    </div>
  </div>

  <!-- Log Table -->
  <div
    class="bg-slate-900/60 backdrop-blur-md rounded-b-3xl border border-t-0 border-slate-700/50 shadow-[0_8px_30px_rgb(0,0,0,0.5)] overflow-hidden w-full relative -mt-px z-0"
  >
    {#if isLoading && logs.length === 0}
      <div class="flex justify-center p-12">
        <svg
          class="animate-spin h-8 w-8 text-cyan-500"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          ><circle
            class="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="4"
          ></circle><path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          ></path></svg
        >
      </div>
    {:else if logs.length === 0}
      <div class="p-8 md:p-12 text-center">
        <div
          class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-slate-800 border border-slate-700 shadow-[0_0_15px_rgba(0,0,0,0.5)] mb-4"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="text-slate-400 h-8 w-8"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><path
              d="M21.2 8.4c.5.38.8.97.8 1.6v10a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V10a2 2 0 0 1 .8-1.6l8-6a2 2 0 0 1 2.4 0l8 6Z"
            ></path><path d="m22 10-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 10"
            ></path></svg
          >
        </div>
        <h3 class="text-lg font-bold text-cyan-50 font-mono tracking-widest">
          NO_LOGS_RECORDED
        </h3>
        <p
          class="text-slate-400 mt-2 text-sm font-mono uppercase tracking-wider"
        >
          WAITING FOR THE BACKGROUND WORKER TO EXECUTE HEALTH CHECKS.
        </p>
      </div>
    {:else if filteredLogs.length === 0}
      <div class="p-8 md:p-12 text-center py-24">
        <div
          class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-cyan-900/30 mb-4 ring-8 ring-cyan-900/10 border border-cyan-500/30 shadow-[0_0_15px_rgba(6,182,212,0.2)]"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="text-cyan-400 h-8 w-8"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            ><path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            /></svg
          >
        </div>
        <h3
          class="text-xl font-bold text-cyan-50 tracking-widest font-mono uppercase"
        >
          NO MATCHING LOGS FOUND
        </h3>
        <p
          class="text-slate-400 mt-3 text-sm max-w-sm mx-auto font-mono uppercase tracking-wider"
        >
          WE COULDN'T FIND ANY HEALTH CHECK LOGS MATCHING YOUR SEARCH TERMS OR
          FILTER CRITERIA.
        </p>
        <button
          on:click={() => {
            searchQuery = "";
            statusFilter = "ALL";
          }}
          class="mt-8 px-5 py-2.5 text-xs font-bold font-mono tracking-widest text-cyan-400 bg-cyan-950/50 border border-cyan-500/30 rounded-xl hover:bg-cyan-900/50 hover:border-cyan-400/50 hover:shadow-[0_0_15px_rgba(6,182,212,0.3)] transition-all inline-flex items-center gap-2 uppercase"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><path
              d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.3"
            /></svg
          >
          CLEAR_ALL_FILTERS
        </button>
      </div>
    {:else}
      <div class="overflow-x-auto w-full">
        <table
          class="w-full text-left border-collapse whitespace-nowrap min-w-[800px]"
        >
          <thead>
            <tr
              class="bg-slate-950/80 border-b border-slate-700/50 text-[10px] font-bold text-slate-400 uppercase tracking-widest font-mono"
            >
              <th class="p-3 md:p-4 pl-4 md:pl-6">Status</th>
              <th class="p-3 md:p-4">Endpoint_Name</th>
              <th class="p-3 md:p-4">Check_Time</th>
              <th class="p-3 md:p-4">Schedule</th>
              <th class="p-3 md:p-4">Resp_Time</th>
              <th class="p-3 md:p-4">HTTP</th>
              <th class="p-3 md:p-4 pr-4 md:pr-6">Error_Detail</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/50 text-sm">
            {#each paginatedLogs as log}
              <tr
                class="hover:bg-slate-800/50 transition-colors cursor-pointer active:bg-slate-800"
                on:click={() => viewLogDetails(log)}
              >
                <td class="p-3 md:p-4 pl-4 md:pl-6">
                  {#if log.is_success}
                    <div
                      class="flex items-center gap-2 text-emerald-400 font-bold font-mono text-xs tracking-widest"
                    >
                      <div
                        class="w-1.5 h-1.5 rounded-full bg-emerald-400 shadow-[0_0_5px_rgba(52,211,153,0.8)]"
                      ></div>
                      UP
                    </div>
                  {:else}
                    <div
                      class="flex items-center gap-2 text-red-500 font-bold font-mono text-xs tracking-widest"
                    >
                      <div
                        class="w-1.5 h-1.5 rounded-full bg-red-500 shadow-[0_0_5px_rgba(239,68,68,0.8)]"
                      ></div>
                      DOWN
                    </div>
                  {/if}
                </td>
                <td
                  class="p-3 md:p-4 text-cyan-50 font-bold font-mono tracking-wide truncate max-w-[200px]"
                  title={log.api?.name || "Unknown API"}
                >
                  {log.api?.name || `API-${log.api_id}`}
                </td>
                <td class="p-3 md:p-4 flex flex-col justify-center">
                  <span class="font-bold text-slate-300 font-mono text-xs"
                    >{formatDateTime(log.checked_at)}</span
                  >
                  <span
                    class="text-[10px] font-bold text-slate-500 font-mono tracking-wider uppercase mt-0.5"
                    >{formatRelativeTime(log.checked_at)}</span
                  >
                </td>
                <td class="p-3 md:p-4">
                  <span
                    class="text-xs font-bold font-mono uppercase tracking-wider {log
                      .api?.interval
                      ? 'text-cyan-400'
                      : 'text-slate-600'}"
                  >
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
                  <span
                    class="font-mono text-xs font-bold {log.response_time > 1000
                      ? 'text-amber-400'
                      : 'text-slate-300'}"
                  >
                    {log.response_time}ms
                  </span>
                </td>
                <td class="p-3 md:p-4">
                  <span
                    class="inline-flex items-center justify-center px-2 py-0.5 border rounded text-[10px] font-bold font-mono tracking-wider {log.status_code >=
                      200 && log.status_code < 300
                      ? 'bg-emerald-950/50 text-emerald-400 border-emerald-500/30'
                      : log.status_code > 0
                        ? 'bg-red-950/50 text-red-400 border-red-500/30'
                        : 'bg-slate-800 text-slate-400 border-slate-700'}"
                  >
                    {log.status_code || "ERR"}
                  </span>
                </td>
                <td
                  class="p-3 md:p-4 pr-4 md:pr-6 truncate max-w-[150px] md:max-w-xs text-slate-400 font-mono text-xs"
                  title={log.error_message}
                >
                  {log.error_message || "-"}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>

        <!-- Pagination Controls -->
        {#if totalPages > 1}
          <div
            class="flex items-center justify-between border-t border-slate-700/50 px-4 py-3 bg-slate-900/50 sm:px-6 rounded-b-3xl"
          >
            <div class="flex flex-1 justify-between sm:hidden">
              <button
                on:click={() => currentPage > 1 && currentPage--}
                disabled={currentPage === 1}
                class="relative inline-flex items-center rounded-md border border-slate-700 bg-slate-800 px-4 py-2 text-xs font-bold font-mono tracking-widest uppercase text-slate-300 hover:bg-slate-700 hover:text-cyan-50 disabled:opacity-50 transition-colors"
                >Previous</button
              >
              <button
                on:click={() => currentPage < totalPages && currentPage++}
                disabled={currentPage === totalPages}
                class="relative ml-3 inline-flex items-center rounded-md border border-slate-700 bg-slate-800 px-4 py-2 text-xs font-bold font-mono tracking-widest uppercase text-slate-300 hover:bg-slate-700 hover:text-cyan-50 disabled:opacity-50 transition-colors"
                >Next</button
              >
            </div>
            <div
              class="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between"
            >
              <div>
                <p
                  class="text-xs font-mono tracking-wider text-slate-400 uppercase"
                >
                  SHOWING
                  <span class="font-bold text-cyan-400"
                    >{(currentPage - 1) * itemsPerPage + 1}</span
                  >
                  TO
                  <span class="font-bold text-cyan-400"
                    >{Math.min(currentPage * itemsPerPage, logs.length)}</span
                  >
                  OF
                  <span class="font-bold text-cyan-400">{logs.length}</span>
                  RESULTS
                </p>
              </div>
              <div>
                <nav
                  class="isolate inline-flex -space-x-px rounded-md shadow-sm"
                  aria-label="Pagination"
                >
                  <button
                    on:click={() => currentPage > 1 && currentPage--}
                    disabled={currentPage === 1}
                    class="relative inline-flex items-center rounded-l-md px-2 py-2 text-slate-500 ring-1 ring-inset ring-slate-700 bg-slate-800 hover:bg-slate-700 hover:text-cyan-400 focus:z-20 focus:outline-offset-0 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    <span class="sr-only">Previous</span>
                    <svg
                      class="h-5 w-5"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                      aria-hidden="true"
                    >
                      <path
                        fill-rule="evenodd"
                        d="M12.79 5.23a.75.75 0 01-.02 1.06L8.832 10l3.938 3.71a.75.75 0 11-1.04 1.08l-4.5-4.25a.75.75 0 010-1.08l4.5-4.25a.75.75 0 011.06.02z"
                        clip-rule="evenodd"
                      />
                    </svg>
                  </button>

                  {#each Array(totalPages) as _, i}
                    <!-- Show max 5 page numbers -->
                    {#if i + 1 === 1 || i + 1 === totalPages || (i + 1 >= currentPage - 1 && i + 1 <= currentPage + 1)}
                      <button
                        on:click={() => (currentPage = i + 1)}
                        class="relative inline-flex items-center px-4 py-2 text-sm font-bold font-mono {currentPage ===
                        i + 1
                          ? 'z-10 bg-cyan-900/80 text-cyan-400 ring-1 ring-inset ring-cyan-500/50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-500 shadow-[0_0_10px_rgba(6,182,212,0.2)]'
                          : 'text-slate-400 ring-1 ring-inset ring-slate-700 bg-slate-800 hover:bg-slate-700 hover:text-cyan-50 focus:z-20 focus:outline-offset-0 transition-colors'}"
                      >
                        {i + 1}
                      </button>
                    {:else if (i + 1 === currentPage - 2 && currentPage > 3) || (i + 1 === currentPage + 2 && currentPage < totalPages - 2)}
                      <span
                        class="relative inline-flex items-center px-4 py-2 text-sm font-bold font-mono text-slate-500 ring-1 ring-inset ring-slate-700 bg-slate-800 focus:outline-offset-0"
                      >
                        ...
                      </span>
                    {/if}
                  {/each}

                  <button
                    on:click={() => currentPage < totalPages && currentPage++}
                    disabled={currentPage === totalPages}
                    class="relative inline-flex items-center rounded-r-md px-2 py-2 text-slate-500 ring-1 ring-inset ring-slate-700 bg-slate-800 hover:bg-slate-700 hover:text-cyan-400 focus:z-20 focus:outline-offset-0 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    <span class="sr-only">Next</span>
                    <svg
                      class="h-5 w-5"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                      aria-hidden="true"
                    >
                      <path
                        fill-rule="evenodd"
                        d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                        clip-rule="evenodd"
                      />
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

<!-- Log Details Modal -->
{#if showLogModal && selectedLog}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6">
    <div
      class="absolute inset-0 bg-slate-950/80 backdrop-blur-md transition-opacity"
      on:click={() => (showLogModal = false)}
    ></div>

    <div
      class="relative bg-slate-900 border border-slate-700/50 rounded-2xl shadow-[0_8px_30px_rgb(0,0,0,0.8)] w-full max-w-2xl max-h-[90vh] flex flex-col transform transition-all overflow-hidden animate-in fade-in zoom-in-95 duration-200"
    >
      <!-- Modal Header -->
      <div
        class="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-950/50"
      >
        <div class="flex items-center gap-3">
          <div
            class="p-2 rounded-lg border shadow-inner {selectedLog.is_success
              ? 'bg-emerald-950/50 text-emerald-400 border-emerald-500/30'
              : 'bg-red-950/50 text-red-500 border-red-500/30'}"
          >
            {#if selectedLog.is_success}
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline
                  points="22 4 12 14.01 9 11.01"
                ></polyline></svg
              >
            {:else}
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><circle cx="12" cy="12" r="10"></circle><line
                  x1="12"
                  y1="8"
                  x2="12"
                  y2="12"
                ></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg
              >
            {/if}
          </div>
          <div>
            <h3
              class="text-lg font-bold text-cyan-50 font-mono tracking-widest uppercase leading-tight"
            >
              RESPONSE_DETAILS
            </h3>
            <p
              class="text-xs font-bold text-slate-500 font-mono tracking-wider mt-1"
            >
              {formatDateTime(selectedLog.checked_at)}
            </p>
          </div>
        </div>
        <button
          on:click={() => (showLogModal = false)}
          class="text-slate-500 hover:text-cyan-400 hover:bg-slate-800 p-2 rounded-full transition-colors"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><line x1="18" y1="6" x2="6" y2="18"></line><line
              x1="6"
              y1="6"
              x2="18"
              y2="18"
            ></line></svg
          >
        </button>
      </div>

      <!-- Modal Body (Scrollable) -->
      <div class="overflow-y-auto p-6 flex-1 bg-slate-900/50">
        <div class="grid grid-cols-2 gap-4 mb-6">
          <div
            class="bg-slate-800/80 rounded-xl p-4 border border-slate-700/50 shadow-inner"
          >
            <span
              class="block text-[10px] font-bold text-slate-500 font-mono tracking-widest uppercase mb-1.5"
              >HTTP_STATUS</span
            >
            <div class="flex items-center gap-2">
              <span
                class="inline-flex items-center justify-center px-2 py-0.5 border rounded text-xs font-bold font-mono tracking-wider {selectedLog.status_code >=
                  200 && selectedLog.status_code < 300
                  ? 'bg-emerald-950/50 text-emerald-400 border-emerald-500/30'
                  : selectedLog.status_code > 0
                    ? 'bg-red-950/50 text-red-400 border-red-500/30'
                    : 'bg-slate-800 text-slate-400 border-slate-700'}"
              >
                {selectedLog.status_code || "ERR"}
              </span>
              <span
                class="text-xs font-bold font-mono tracking-widest uppercase {selectedLog.is_success
                  ? 'text-emerald-400'
                  : 'text-red-400'}"
                >{selectedLog.is_success ? "OK" : "ERROR"}</span
              >
            </div>
          </div>
          <div
            class="bg-slate-800/80 rounded-xl p-4 border border-slate-700/50 shadow-inner"
          >
            <span
              class="block text-[10px] font-bold text-slate-500 font-mono tracking-widest uppercase mb-1.5"
              >RESPONSE_TIME</span
            >
            <div class="flex items-baseline gap-1">
              <span
                class="text-2xl font-black font-mono {selectedLog.response_time >
                1000
                  ? 'text-amber-400 drop-shadow-[0_0_8px_rgba(251,191,36,0.5)]'
                  : 'text-cyan-400 drop-shadow-[0_0_8px_rgba(34,211,238,0.5)]'}"
                >{selectedLog.response_time}</span
              >
              <span
                class="text-xs font-bold text-slate-500 font-mono tracking-widest"
                >ms</span
              >
            </div>
          </div>
        </div>

        <div class="mb-6">
          <h4
            class="text-xs font-bold text-cyan-50 font-mono tracking-widest uppercase mb-3 border-b border-slate-800 pb-2"
          >
            ENDPOINT_INFORMATION
          </h4>
          <div
            class="bg-slate-950/50 rounded-lg border border-slate-700/80 overflow-hidden text-sm shadow-inner"
          >
            <div class="grid grid-cols-[100px_1fr] border-b border-slate-800">
              <div
                class="p-3 bg-slate-900 text-slate-500 text-xs font-bold font-mono tracking-wider uppercase border-r border-slate-800"
              >
                NAME
              </div>
              <div
                class="p-3 font-bold font-mono text-cyan-50 text-xs tracking-wide"
              >
                {selectedLog.api?.name || `API-${selectedLog.api_id}`}
              </div>
            </div>
            {#if selectedLog.api?.url}
              <div class="grid grid-cols-[100px_1fr] border-b border-slate-800">
                <div
                  class="p-3 bg-slate-900 text-slate-500 text-xs font-bold font-mono tracking-wider uppercase border-r border-slate-800"
                >
                  METHOD
                </div>
                <div class="p-3">
                  <span
                    class="px-2 py-0.5 rounded border border-blue-500/30 text-[10px] font-bold font-mono bg-blue-950/50 text-blue-400 tracking-widest"
                    >{selectedLog.api?.method || "GET"}</span
                  >
                </div>
              </div>
              <div class="grid grid-cols-[100px_1fr]">
                <div
                  class="p-3 bg-slate-900 text-slate-500 text-xs font-bold font-mono tracking-wider uppercase border-r border-slate-800"
                >
                  URL
                </div>
                <div
                  class="p-3 font-mono text-xs text-slate-400 break-all bg-slate-900/50"
                >
                  {selectedLog.api?.url}
                </div>
              </div>
            {/if}
            {#if selectedLog.api?.headers && selectedLog.api?.headers !== "{}" && selectedLog.api?.headers !== "[]"}
              <div class="grid grid-cols-[100px_1fr] border-t border-slate-800">
                <div
                  class="p-3 bg-slate-900 text-slate-500 text-xs font-bold font-mono tracking-wider uppercase border-r border-slate-800"
                >
                  HEADERS
                </div>
                <div class="p-3 bg-slate-900/50">
                  <pre
                    class="font-mono text-[10px] text-green-400 whitespace-pre-wrap break-all">{selectedLog
                      .api.headers}</pre>
                </div>
              </div>
            {/if}
            {#if selectedLog.api?.body}
              <div class="grid grid-cols-[100px_1fr] border-t border-slate-800">
                <div
                  class="p-3 bg-slate-900 text-slate-500 text-xs font-bold font-mono tracking-wider uppercase border-r border-slate-800"
                >
                  BODY
                </div>
                <div class="p-3 bg-slate-900/50">
                  <pre
                    class="font-mono text-[10px] text-blue-400 whitespace-pre-wrap break-all">{selectedLog
                      .api.body}</pre>
                </div>
              </div>
            {/if}
          </div>
        </div>

        {#if selectedLog.error_message || selectedLog.response_body}
          <div>
            <h4
              class="text-xs font-bold text-cyan-50 font-mono tracking-widest uppercase mb-3 flex items-center gap-2 border-b border-slate-800 pb-2"
            >
              {#if selectedLog.is_success}
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  class="text-emerald-500"
                  ><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline
                    points="22 4 12 14.01 9 11.01"
                  ></polyline></svg
                >
              {:else}
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  class="text-red-500"
                  ><circle cx="12" cy="12" r="10"></circle><line
                    x1="12"
                    y1="8"
                    x2="12"
                    y2="12"
                  ></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg
                >
              {/if}
              {selectedLog.is_success
                ? "RESPONSE_BODY"
                : "ERROR_DETAILS / RESPONSE_BODY"}
            </h4>
            <div
              class="bg-slate-950 rounded-xl p-4 overflow-hidden border border-slate-800 shadow-inner"
            >
              <pre
                class="{selectedLog.is_success
                  ? 'text-emerald-400'
                  : 'text-red-400'} font-mono text-[10px] whitespace-pre-wrap leading-relaxed overflow-x-auto"><code
                  >{selectedLog.response_body ||
                    selectedLog.error_message}</code
                ></pre>
            </div>
          </div>
        {/if}
      </div>

      <!-- Modal Footer -->
      <div
        class="px-6 py-4 border-t border-slate-100 bg-slate-50/80 flex justify-end"
      >
        <button
          on:click={() => (showLogModal = false)}
          class="px-5 py-2.5 bg-slate-200 hover:bg-slate-300 text-slate-700 font-bold rounded-xl transition-colors shadow-sm focus:outline-none focus:ring-2 focus:ring-slate-400 focus:ring-offset-2"
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Hide scrollbar for Chrome, Safari and Opera */
  .hide-scrollbar::-webkit-scrollbar {
    display: none;
  }
  /* Hide scrollbar for IE, Edge and Firefox */
  .hide-scrollbar {
    -ms-overflow-style: none; /* IE and Edge */
    scrollbar-width: none; /* Firefox */
  }
</style>
