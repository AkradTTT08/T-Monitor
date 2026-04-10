<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { page } from "$app/stores";
  import Chart from "chart.js/auto";
  import { API_BASE_URL } from "$lib/config";

  let isLoading = true;
  let selectedProjectId = "";
  let selectedPeriod = "24h";
  let refreshInterval: any;

  // Data
  let uptimeData: any = null;
  let latencyData: any = null;
  let incidentData: any = null;

  // Charts
  let latencyChartCanvas: HTMLCanvasElement;
  let latencyChart: Chart | null = null;
  let uptimeBarCanvas: HTMLCanvasElement;
  let uptimeBarChart: Chart | null = null;

  onMount(async () => {
    selectedProjectId =
      $page.url.searchParams.get("project_id") ||
      localStorage.getItem("monitor_selected_project") ||
      "";
    await fetchAllData();
    refreshInterval = setInterval(fetchAllData, 30000);
  });

  onDestroy(() => {
    if (refreshInterval) clearInterval(refreshInterval);
    if (latencyChart) latencyChart.destroy();
    if (uptimeBarChart) uptimeBarChart.destroy();
  });

  async function fetchAllData() {
    if (!selectedProjectId) return;
    isLoading = true;
    const token = localStorage.getItem("monitor_token");
    const headers = { Authorization: `Bearer ${token}` };

    try {
      const [uptimeRes, latencyRes, incidentRes] = await Promise.all([
        fetch(`${API_BASE_URL}/api/v1/analytics/uptime?project_id=${selectedProjectId}&period=${selectedPeriod}`, { headers }),
        fetch(`${API_BASE_URL}/api/v1/analytics/latency-trend?project_id=${selectedProjectId}&period=${selectedPeriod}`, { headers }),
        fetch(`${API_BASE_URL}/api/v1/analytics/incidents?project_id=${selectedProjectId}&limit=30`, { headers }),
      ]);

      if (uptimeRes.ok) uptimeData = await uptimeRes.json();
      if (latencyRes.ok) latencyData = await latencyRes.json();
      if (incidentRes.ok) incidentData = await incidentRes.json();
    } catch (err) {
      console.error("Failed to fetch analytics data:", err);
    } finally {
      isLoading = false;
    }
  }

  async function changePeriod(period: string) {
    selectedPeriod = period;
    await fetchAllData();
  }

  // Reactively update charts
  $: if (latencyChartCanvas && latencyData?.data_points) {
    renderLatencyChart();
  }
  $: if (uptimeBarCanvas && uptimeData?.apis) {
    renderUptimeBarChart();
  }

  $: avgLatency = uptimeData?.apis?.length 
    ? Math.round(uptimeData.apis.reduce((a: any, b: any) => a + b.avg_latency, 0) / uptimeData.apis.length) 
    : 0;

  function renderLatencyChart() {
    if (!latencyChartCanvas || !latencyData?.data_points) return;

    const points = latencyData.data_points;
    const labels = points.map((p: any) => {
      const ts = p.timestamp;
      if (selectedPeriod === "30d" && ts.length <= 10) return ts.slice(5); // MM-DD
      if (selectedPeriod === "7d") return ts.slice(5, 10) + " " + ts.slice(11) + ":00"; // MM-DD HH:00
      return ts.slice(11) + ":00"; // HH:00 for 24h
    });

    try {
      if (latencyChart) {
        latencyChart.destroy();
      }

      latencyChart = new Chart(latencyChartCanvas, {
        type: "line",
        data: {
          labels,
          datasets: [
            {
              label: "Avg Latency (ms)",
              data: points.map((p: any) => p.avg_latency),
              borderColor: "rgba(6, 182, 212, 0.8)",
              backgroundColor: "rgba(6, 182, 212, 0.1)",
              fill: true,
              tension: 0.4,
              borderWidth: 2,
              pointRadius: 2,
              pointHoverRadius: 5,
            },
            {
              label: "Max Latency (ms)",
              data: points.map((p: any) => p.max_latency),
              borderColor: "rgba(239, 68, 68, 0.5)",
              backgroundColor: "transparent",
              borderDash: [5, 5],
              tension: 0.4,
              borderWidth: 1.5,
              pointRadius: 0,
            },
            {
              label: "Success Rate (%)",
              data: points.map((p: any) => p.success_rate),
              borderColor: "rgba(34, 197, 94, 0.8)",
              backgroundColor: "transparent",
              tension: 0.4,
              borderWidth: 2,
              pointRadius: 0,
              yAxisID: "y1",
            },
          ],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          interaction: { mode: "index", intersect: false },
          plugins: {
            legend: {
              labels: { color: "#94a3b8", font: { size: 11, family: "'Inter', sans-serif" } },
            },
            tooltip: {
              backgroundColor: "rgba(15, 23, 42, 0.95)",
              titleFont: { size: 12, family: "'Inter', sans-serif" },
              bodyFont: { size: 11, family: "'Inter', sans-serif" },
              cornerRadius: 8,
              padding: 10,
            },
          },
          scales: {
            y: {
              beginAtZero: true,
              grid: { color: "rgba(226, 232, 240, 0.05)" },
              ticks: { color: "#64748b", font: { size: 10 }, callback: (v) => v + "ms" },
              title: { display: true, text: "Latency", color: "#64748b", font: { size: 10 } },
            },
            y1: {
              position: "right",
              min: 0, max: 100,
              grid: { display: false },
              ticks: { color: "#22c55e", font: { size: 10 }, callback: (v) => v + "%" },
              title: { display: true, text: "Success Rate", color: "#22c55e", font: { size: 10 } },
            },
            x: {
              grid: { display: false },
              ticks: { color: "#64748b", font: { size: 9 }, maxRotation: 45 },
            },
          },
        },
      });
    } catch (err) {
      console.error("Error creating latency chart:", err);
    }
  }

  function renderUptimeBarChart() {
    if (!uptimeBarCanvas || !uptimeData?.apis) return;

    const apis = uptimeData.apis.slice(0, 15);
    const labels = apis.map((a: any) => a.name.length > 18 ? a.name.slice(0, 18) + "…" : a.name);
    const data = apis.map((a: any) => a.uptime_percent);
    const colors = data.map((v: number) => v >= 99 ? "rgba(34, 197, 94, 0.7)" : v >= 95 ? "rgba(234, 179, 8, 0.7)" : "rgba(239, 68, 68, 0.7)");

    try {
      if (uptimeBarChart) {
        uptimeBarChart.destroy();
      }

      uptimeBarChart = new Chart(uptimeBarCanvas, {
        type: "bar",
        data: {
          labels,
          datasets: [{
            label: "Uptime %",
            data,
            backgroundColor: colors,
            borderRadius: 6,
            barPercentage: 0.7,
          }],
        },
        options: {
          indexAxis: "y",
          responsive: true,
          maintainAspectRatio: false,
          plugins: {
            legend: { display: false },
            tooltip: {
              backgroundColor: "rgba(15, 23, 42, 0.95)",
              cornerRadius: 8,
              callbacks: { label: (ctx) => `${ctx.raw}% uptime` },
            },
          },
          scales: {
            x: { min: 0, max: 100, grid: { color: "rgba(226, 232, 240, 0.05)" }, ticks: { color: "#64748b", callback: (v) => v + "%" } },
            y: { grid: { display: false }, ticks: { color: "#94a3b8", font: { size: 11 } } },
          },
        },
      });
    } catch (err) {
      console.error("Error creating uptime bar chart:", err);
    }
  }

  function getUptimeColor(pct: number): string {
    if (pct >= 99) return "text-emerald-400";
    if (pct >= 95) return "text-amber-400";
    return "text-red-400";
  }

  function getUptimeBg(pct: number): string {
    if (pct >= 99) return "bg-emerald-500";
    if (pct >= 95) return "bg-amber-500";
    return "bg-red-500";
  }

  function formatTimeAgo(dateStr: string): string {
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "Just now";
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    return `${Math.floor(hrs / 24)}d ago`;
  }
</script>

<div class="fade-in max-w-7xl mx-auto w-full overflow-hidden p-4 md:p-6">
  <!-- Header -->
  <div class="flex flex-col md:flex-row md:items-end justify-between gap-4 mb-8">
    <div>
      <h1 class="text-2xl md:text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-indigo-400 via-purple-400 to-cyan-400 tracking-tight font-mono uppercase">
        Analytics Dashboard
      </h1>
      <p class="text-indigo-400/60 mt-1 font-mono text-xs tracking-wide uppercase">
        Uptime, Performance & Incident Intelligence
      </p>
    </div>

    <!-- Period Selector -->
    <div class="flex items-center gap-2 bg-slate-900/60 backdrop-blur-sm border border-slate-800 rounded-2xl p-1.5">
      {#each ["24h", "7d", "30d"] as period}
        <button
          onclick={() => changePeriod(period)}
          class="px-4 py-2 text-xs font-bold uppercase tracking-wider rounded-xl transition-all duration-300
            {selectedPeriod === period
              ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-500/30'
              : 'text-slate-500 hover:text-indigo-400 hover:bg-slate-800/60'}"
        >
          {period}
        </button>
      {/each}
    </div>
  </div>

  {#if isLoading}
    <div class="flex flex-col items-center justify-center py-24 gap-4">
      <div class="relative">
        <div class="w-12 h-12 border-4 border-slate-800 rounded-full"></div>
        <div class="w-12 h-12 border-4 border-indigo-500 border-t-transparent rounded-full animate-spin absolute inset-0"></div>
      </div>
      <p class="text-slate-500 text-xs font-mono uppercase tracking-widest">Loading Analytics...</p>
    </div>
  {:else}
    <!-- Overview Cards -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      <!-- Uptime Card -->
      <div class="relative bg-slate-800/30 backdrop-blur-xl rounded-2xl border border-slate-700/40 p-5 overflow-hidden group hover:border-indigo-500/30 transition-all duration-500">
        <div class="absolute top-0 right-0 w-24 h-24 bg-gradient-to-bl from-indigo-500/10 to-transparent rounded-bl-[50px] group-hover:from-indigo-500/20 transition-all"></div>
        <p class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-3">Overall Uptime</p>
        <div class="flex items-end gap-2">
          <span class="text-3xl font-black {getUptimeColor(uptimeData?.overall_uptime || 0)}">{uptimeData?.overall_uptime || 0}%</span>
        </div>
        <div class="mt-3 w-full bg-slate-900 rounded-full h-1.5 overflow-hidden">
          <div class="{getUptimeBg(uptimeData?.overall_uptime || 0)} h-full rounded-full transition-all duration-1000" style="width: {uptimeData?.overall_uptime || 0}%"></div>
        </div>
      </div>

      <!-- Avg Latency Card -->
      <div class="relative bg-slate-800/30 backdrop-blur-xl rounded-2xl border border-slate-700/40 p-5 overflow-hidden group hover:border-cyan-500/30 transition-all duration-500">
        <div class="absolute top-0 right-0 w-24 h-24 bg-gradient-to-bl from-cyan-500/10 to-transparent rounded-bl-[50px] group-hover:from-cyan-500/20 transition-all"></div>
        <p class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-3">Avg Response</p>
        <div class="flex items-end gap-1">
          <span class="text-3xl font-black text-cyan-400">{avgLatency}</span>
          <span class="text-sm text-cyan-500/60 font-bold mb-1">ms</span>
        </div>
      </div>

      <!-- Total Checks Card -->
      <div class="relative bg-slate-800/30 backdrop-blur-xl rounded-2xl border border-slate-700/40 p-5 overflow-hidden group hover:border-emerald-500/30 transition-all duration-500">
        <div class="absolute top-0 right-0 w-24 h-24 bg-gradient-to-bl from-emerald-500/10 to-transparent rounded-bl-[50px] group-hover:from-emerald-500/20 transition-all"></div>
        <p class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-3">Total Checks</p>
        <span class="text-3xl font-black text-emerald-400">{(uptimeData?.total_checks || 0).toLocaleString()}</span>
      </div>

      <!-- Incidents Card -->
      <div class="relative bg-slate-800/30 backdrop-blur-xl rounded-2xl border border-slate-700/40 p-5 overflow-hidden group hover:border-red-500/30 transition-all duration-500">
        <div class="absolute top-0 right-0 w-24 h-24 bg-gradient-to-bl from-red-500/10 to-transparent rounded-bl-[50px] group-hover:from-red-500/20 transition-all"></div>
        <p class="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-3">Failures</p>
        <span class="text-3xl font-black text-red-400">{(uptimeData?.total_failures || 0).toLocaleString()}</span>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="grid grid-cols-1 lg:grid-cols-5 gap-6 mb-8">
      <!-- Latency Trend Chart (3/5) -->
      <div class="lg:col-span-3 bg-slate-800/30 backdrop-blur-xl rounded-2xl border border-slate-700/40 p-5">
        <h3 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-4 flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-cyan-500"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
          Latency & Success Rate Trend
        </h3>
        <div class="h-[280px]">
          <canvas bind:this={latencyChartCanvas}></canvas>
        </div>
      </div>

      <!-- Uptime Bar Chart (2/5) -->
      <div class="lg:col-span-2 bg-slate-800/30 backdrop-blur-xl rounded-2xl border border-slate-700/40 p-5">
        <h3 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-4 flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-emerald-500"><path d="M12 20V10"/><path d="M18 20V4"/><path d="M6 20v-4"/></svg>
          API Uptime Ranking
        </h3>
        <div style="height: {Math.max(200, (uptimeData?.apis?.length || 3) * 32)}px">
          <canvas bind:this={uptimeBarCanvas}></canvas>
        </div>
      </div>
    </div>

    <!-- API Uptime Table -->
    <div class="bg-slate-800/30 backdrop-blur-xl rounded-2xl border border-slate-700/40 p-5 mb-8">
      <h3 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-4 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-500"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
        Per-API Health Report ({selectedPeriod})
      </h3>
      <div class="overflow-x-auto">
        <table class="w-full text-left">
          <thead>
            <tr class="border-b border-slate-700/50">
              <th class="py-3 px-3 text-[10px] font-bold text-slate-500 uppercase tracking-widest">API</th>
              <th class="py-3 px-3 text-[10px] font-bold text-slate-500 uppercase tracking-widest text-center">Uptime</th>
              <th class="py-3 px-3 text-[10px] font-bold text-slate-500 uppercase tracking-widest text-center">Avg</th>
              <th class="py-3 px-3 text-[10px] font-bold text-slate-500 uppercase tracking-widest text-center">Max</th>
              <th class="py-3 px-3 text-[10px] font-bold text-slate-500 uppercase tracking-widest text-center">Checks</th>
              <th class="py-3 px-3 text-[10px] font-bold text-slate-500 uppercase tracking-widest text-center">Fails</th>
              <th class="py-3 px-3 text-[10px] font-bold text-slate-500 uppercase tracking-widest text-right">Last Check</th>
            </tr>
          </thead>
          <tbody>
            {#if uptimeData?.apis}
              {#each uptimeData.apis as api}
                <tr class="border-b border-slate-800/50 hover:bg-slate-800/30 transition-colors group">
                  <td class="py-3 px-3">
                    <div class="flex items-center gap-2">
                      <span class="text-[10px] font-bold px-1.5 py-0.5 rounded bg-slate-900 text-slate-400 border border-slate-800 uppercase">{api.method}</span>
                      <span class="text-sm font-semibold text-cyan-50 truncate max-w-[200px]">{api.name}</span>
                    </div>
                  </td>
                  <td class="py-3 px-3 text-center">
                    <div class="flex flex-col items-center gap-1">
                      <span class="text-sm font-black {getUptimeColor(api.uptime_percent)}">{api.uptime_percent}%</span>
                      <div class="w-14 bg-slate-900 rounded-full h-1 overflow-hidden">
                        <div class="{getUptimeBg(api.uptime_percent)} h-full rounded-full" style="width: {api.uptime_percent}%"></div>
                      </div>
                    </div>
                  </td>
                  <td class="py-3 px-3 text-center text-xs font-mono text-cyan-400">{api.avg_latency}ms</td>
                  <td class="py-3 px-3 text-center text-xs font-mono text-slate-500">{api.max_latency}ms</td>
                  <td class="py-3 px-3 text-center text-xs font-mono text-slate-400">{api.total_checks}</td>
                  <td class="py-3 px-3 text-center">
                    {#if api.fail_count > 0}
                      <span class="text-xs font-bold text-red-400 bg-red-950/40 px-2 py-0.5 rounded-md border border-red-500/20">{api.fail_count}</span>
                    {:else}
                      <span class="text-xs text-emerald-500">0</span>
                    {/if}
                  </td>
                  <td class="py-3 px-3 text-right text-[10px] text-slate-500 font-mono">
                    {api.last_checked ? formatTimeAgo(api.last_checked) : "N/A"}
                  </td>
                </tr>
              {/each}
            {:else}
              <tr>
                <td colspan="7" class="py-12 text-center text-slate-600 text-sm">No API data available</td>
              </tr>
            {/if}
          </tbody>
        </table>
      </div>
    </div>

    <!-- Incident Timeline -->
    <div class="bg-slate-800/30 backdrop-blur-xl rounded-2xl border border-slate-700/40 p-5">
      <h3 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-6 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-red-400"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
        Recent Incident Timeline
      </h3>

      {#if incidentData?.incidents?.length > 0}
        <div class="relative ml-4">
          <!-- Vertical Line -->
          <div class="absolute left-0 top-0 bottom-0 w-px bg-gradient-to-b from-red-500/50 via-amber-500/30 to-slate-800"></div>

          <div class="space-y-4">
            {#each incidentData.incidents as incident, i}
              <div class="relative pl-8 group animate-fade-in" style="animation-delay: {i * 60}ms">
                <!-- Dot -->
                <div class="absolute left-0 top-2 -translate-x-1/2 w-3 h-3 rounded-full bg-red-500 border-2 border-slate-900 shadow-[0_0_8px_rgba(239,68,68,0.6)] group-hover:scale-125 transition-transform"></div>

                <!-- Card -->
                <div class="bg-slate-900/50 border border-slate-800 rounded-xl p-4 hover:border-red-500/30 hover:shadow-[0_0_15px_rgba(239,68,68,0.1)] transition-all duration-300">
                  <div class="flex items-start justify-between gap-3 mb-2">
                    <div class="flex items-center gap-2 min-w-0">
                      <span class="text-[9px] font-bold px-1.5 py-0.5 rounded bg-slate-800 text-slate-500 border border-slate-700 uppercase shrink-0">{incident.api_method}</span>
                      <span class="text-sm font-bold text-cyan-50 truncate">{incident.api_name}</span>
                    </div>
                    <div class="flex items-center gap-2 shrink-0">
                      <span class="text-[10px] font-bold bg-red-950/50 text-red-400 px-2 py-0.5 rounded-md border border-red-500/20">
                        {incident.status_code || "ERR"}
                      </span>
                      <span class="text-[10px] text-slate-600 font-mono">{formatTimeAgo(incident.checked_at)}</span>
                    </div>
                  </div>
                  <p class="text-xs text-slate-500 line-clamp-2 leading-relaxed">{incident.error_message || "Unknown error"}</p>
                  {#if incident.response_time > 0}
                    <span class="inline-block mt-2 text-[10px] font-mono text-amber-500/60">{incident.response_time}ms</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {:else}
        <div class="text-center py-12">
          <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-emerald-500/30 mx-auto mb-3"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
          <p class="text-slate-600 text-sm font-medium">No incidents recorded</p>
          <p class="text-slate-700 text-xs mt-1">All systems operating normally ✨</p>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .fade-in {
    animation: fadeIn 0.5s ease-out;
  }
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
  }
  .animate-fade-in {
    animation: fadeIn 0.4s ease-out both;
  }
  .line-clamp-2 {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
</style>
