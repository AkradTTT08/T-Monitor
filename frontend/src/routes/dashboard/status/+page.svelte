<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { page } from "$app/stores";
  import Chart from "chart.js/auto";

  import { API_BASE_URL } from "$lib/config";
  import { systemAlert, systemToast } from "$lib/swal-design";
  import logoUrl from "../../../image/SVG/Logo-T-monitor.svg";

  let logs: any[] = [];
  let isLoading = true;
  let summary = { total: 0, up: 0, down: 0 };
  let refreshInterval: any;
  let selectedProjectId = "";
  let selectedProject: any = null;

  // Chart State
  let chartCanvas: HTMLCanvasElement;
  let statusChart: Chart | null = null;

  // Filter State
  let searchQuery = "";
  let statusFilter = "ALL"; // ALL, UP, DOWN

  // Modal State
  let showLogModal = false;
  let selectedLog: any = null;

  // Report State
  let startDate = "";
  let endDate = "";
  let isGeneratingReport = false;

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
    const chartData = [...filteredLogs].slice(0, 50).reverse();

    const labels = chartData.map((log) =>
      new Intl.DateTimeFormat("en-GB", {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      }).format(new Date(log.checked_at)),
    );

    const backgroundColors = chartData.map((log) =>
      log.is_success ? "rgba(34, 197, 94, 0.8)" : "rgba(239, 68, 68, 0.8)",
    );

    const dataPoints = chartData.map((log) => log.response_time);

    const tooltipLabels = chartData.map((log) => ({
      name: log.api?.name || `API-${log.api_id}`,
      status: log.status_code || "ERR",
      error: log.error_message || "-",
    }));

    if (statusChart) {
      statusChart.data.labels = labels;
      statusChart.data.datasets[0].data = dataPoints;
      statusChart.data.datasets[0].backgroundColor = backgroundColors;
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
              customData: tooltipLabels,
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
                  const status = context[0].dataset.customData[dataIndex].status;
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
              grid: { color: "rgba(226, 232, 240, 0.1)" },
              border: { dash: [4, 4] },
              ticks: {
                font: { family: "'Inter', sans-serif", size: 11 },
                color: "#64748b",
                callback: function (value) {
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
    selectedProjectId =
      $page.url.searchParams.get("project_id") ||
      localStorage.getItem("monitor_selected_project") ||
      "";
    await fetchLogs();
    refreshInterval = setInterval(fetchLogs, 10000);

    if (selectedProjectId) {
      fetchProjectDetails();
    }
  });

  async function fetchProjectDetails() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects/${selectedProjectId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        selectedProject = await res.json();
      }
    } catch (err) {
      console.error("Failed to fetch project details", err);
    }
  }

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

  // --- PDF Report Generation --- //
  async function generateReport() {
    if (!startDate || !endDate) {
      systemAlert.fire({
        icon: "warning",
        title: "Date Range Required",
        text: "Please select both start and end dates for the report.",
      });
      return;
    }

    isGeneratingReport = true;
    try {
      const token = localStorage.getItem("monitor_token");
      let url = `${API_BASE_URL}/api/v1/logs?project_id=${selectedProjectId}&start_date=${startDate}&end_date=${endDate}`;
      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (!res.ok) throw new Error("Failed to fetch report data");
      const reportLogs = await res.json();

      if (reportLogs.length === 0) {
        systemAlert.fire({
          icon: "info",
          title: "No Data Found",
          text: "There are no logs in the selected date range.",
        });
        return;
      }

      const totalChecks = reportLogs.length;
      const successCount = reportLogs.filter((l: any) => l.is_success).length;
      const uptimePercent = ((successCount / totalChecks) * 100).toFixed(2);
      const avgResponse = Math.round(
        reportLogs.reduce((acc: number, l: any) => acc + l.response_time, 0) /
          totalChecks,
      );

      const failureCounts: Record<string, number> = {};
      reportLogs
        .filter((l: any) => !l.is_success)
        .forEach((l: any) => {
          const apiName = l.api?.name || `API-${l.api_id}`;
          failureCounts[apiName] = (failureCounts[apiName] || 0) + 1;
        });
      const topFails = Object.entries(failureCounts)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 3);

      let recommendations = [];
      if (Number(uptimePercent) < 95) recommendations.push("⚠️ Uptime below threshold. Check stability.");
      else recommendations.push("✅ System active and stable.");
      if (avgResponse > 1000) recommendations.push("🐢 High latency detected. Optimize endpoints.");
      if (topFails.length > 0) recommendations.push(`🔴 Focus on failures: ${topFails.map(f => f[0]).join(", ")}`);

      // @ts-ignore
      const { jsPDF } = window.jspdf;
      // @ts-ignore
      const html2canvas = window.html2canvas;

      const pdf = new jsPDF("p", "mm", "a4");
      const reportContent = document.getElementById("pdf-report-export-container");
      if (reportContent) {
        reportContent.style.display = "block";
        const metricsContainer = reportContent.querySelector("#report-metrics");
        if (metricsContainer) {
          metricsContainer.innerHTML = `
            <div class="report-card"><div class="label">UPTIME</div><div class="value">${uptimePercent}%</div></div>
            <div class="report-card"><div class="label">AVG_RESP</div><div class="value">${avgResponse}ms</div></div>
            <div class="report-card"><div class="label">LOGS</div><div class="value">${totalChecks}</div></div>
          `;
        }
        const recList = reportContent.querySelector("#report-recommendations");
        if (recList) {
          recList.innerHTML = recommendations.map(r => `<li class="rec-item">${r}</li>`).join("");
        }

        // Wait for images and chart to settle
        await new Promise(resolve => setTimeout(resolve, 500));
        
        const canvas = await html2canvas(reportContent, {
          scale: 2,
          useCORS: true,
          backgroundColor: "#030712",
          logging: false
        });
        pdf.addImage(canvas.toDataURL("image/png"), "PNG", 0, 0, 210, (canvas.height * 210) / canvas.width);
        pdf.save(`T-Monitor-Report-${startDate}.pdf`);
        reportContent.style.display = "none";
        systemToast.fire({ icon: "success", title: "Report Generated" });
      }
    } catch (err) {
      console.error(err);
      systemAlert.fire({ icon: "error", title: "Export Failed" });
    } finally {
      isGeneratingReport = false;
    }
  }
</script>

<svelte:head>
  <script src="https://unpkg.com/jspdf@latest/dist/jspdf.umd.min.js"></script>
  <script src="https://unpkg.com/html2canvas@1.4.1/dist/html2canvas.min.js"></script>
</svelte:head>

<div class="fade-in max-w-6xl mx-auto w-full overflow-hidden p-6">
  <div class="flex flex-col md:flex-row md:items-end justify-between gap-4 mb-8">
    <div>
      <h1 class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase">API_STATUS_CONSOLE</h1>
      <p class="text-cyan-500/80 mt-2 font-mono text-sm tracking-wide">REAL-TIME DIAGNOSTICS AND HEALTH ANALYTICS.</p>
    </div>

    <div class="flex flex-wrap items-center gap-3">
      <div class="flex flex-col gap-1">
        <label class="text-[10px] font-bold text-slate-500 font-mono uppercase tracking-widest">Period_Start</label>
        <input type="date" bind:value={startDate} on:click={(e) => e.currentTarget.showPicker?.()} class="bg-slate-900 border border-slate-700 rounded-xl px-3 py-1.5 text-xs text-cyan-400 font-mono [color-scheme:dark]"/>
      </div>
      <div class="flex flex-col gap-1">
        <label class="text-[10px] font-bold text-slate-500 font-mono uppercase tracking-widest">Period_End</label>
        <input type="date" bind:value={endDate} on:click={(e) => e.currentTarget.showPicker?.()} class="bg-slate-900 border border-slate-700 rounded-xl px-3 py-1.5 text-xs text-cyan-400 font-mono [color-scheme:dark]"/>
      </div>
      <button on:click={generateReport} disabled={isGeneratingReport} class="mt-auto h-[38px] px-6 rounded-xl font-mono font-bold text-xs uppercase tracking-widest bg-gradient-to-r from-cyan-600/20 to-blue-600/20 text-cyan-400 border border-cyan-500/30">
        {isGeneratingReport ? "EXPORTING..." : "EXPORT_PDF_REPORT"}
      </button>
    </div>
  </div>

  <!-- Summary Cards -->
  <div class="grid grid-cols-1 sm:grid-cols-3 gap-6 mb-8">
    <div class="bg-slate-800/40 rounded-3xl border border-slate-700/50 p-6 shadow-xl">
      <p class="text-cyan-500/80 font-bold text-xs mb-1 uppercase tracking-widest font-mono">MONITORED_ENDPOINTS</p>
      <h2 class="text-3xl font-black text-cyan-50">{summary.total}</h2>
    </div>
    <div class="bg-slate-800/40 rounded-3xl border border-slate-700/50 p-6 shadow-xl">
      <p class="text-emerald-500/80 font-bold text-xs mb-1 uppercase tracking-widest font-mono">OPERATIONAL [UP]</p>
      <h2 class="text-3xl font-black text-emerald-400">{summary.up}</h2>
    </div>
    <div class="bg-slate-800/40 rounded-3xl border border-slate-700/50 p-6 shadow-xl">
      <p class="text-red-500/80 font-bold text-xs mb-1 uppercase tracking-widest font-mono">FAILING [DOWN]</p>
      <h2 class="text-3xl font-black text-red-500">{summary.down}</h2>
    </div>
  </div>

  <!-- Performance Chart -->
  <div class="bg-slate-800/40 p-6 rounded-3xl border border-slate-700/50 shadow-xl mb-8">
    <h3 class="text-sm font-bold text-cyan-50 font-mono tracking-widest mb-6 uppercase flex items-center gap-2">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-cyan-400"><path d="M3 3v18h18"/><path d="m19 9-5 5-4-4-3 3"/></svg>
      PERFORMANCE_HISTORY
    </h3>
    <div class="h-48 w-full"><canvas bind:this={chartCanvas}></canvas></div>
  </div>

  <!-- Filters -->
  <div class="bg-slate-900/60 p-4 rounded-t-3xl border border-slate-700/50 flex flex-col md:flex-row gap-4 items-center justify-between">
    <input type="text" bind:value={searchQuery} placeholder="SEARCH ENDPOINTS..." class="w-full md:max-w-md px-4 py-3 bg-slate-800 border border-slate-700 rounded-xl text-sm text-cyan-50 font-mono"/>
    <div class="flex gap-2">
      <button on:click={() => statusFilter = "ALL"} class="px-4 py-2 text-xs font-mono font-bold rounded-lg border {statusFilter === 'ALL' ? 'bg-cyan-900 border-cyan-500 text-cyan-400' : 'bg-slate-800 border-slate-700 text-slate-400'}">ALL</button>
      <button on:click={() => statusFilter = "UP"} class="px-4 py-2 text-xs font-mono font-bold rounded-lg border {statusFilter === 'UP' ? 'bg-emerald-900 border-emerald-500 text-emerald-400' : 'bg-slate-800 border-slate-700 text-slate-400'}">UP</button>
      <button on:click={() => statusFilter = "DOWN"} class="px-4 py-2 text-xs font-mono font-bold rounded-lg border {statusFilter === 'DOWN' ? 'bg-red-900 border-red-500 text-red-400' : 'bg-slate-800 border-slate-700 text-slate-400'}">DOWN</button>
    </div>
  </div>

  <!-- Table -->
  <div class="bg-slate-900/60 rounded-b-3xl border border-t-0 border-slate-700/50 overflow-hidden">
    <table class="w-full text-left whitespace-nowrap">
      <thead>
        <tr class="bg-slate-950/80 border-b border-slate-700/50 text-[10px] font-bold text-slate-400 uppercase font-mono">
          <th class="p-4">STATUS</th>
          <th class="p-4">ENDPOINT</th>
          <th class="p-4">CHECK_TIME</th>
          <th class="p-4">LATENCY</th>
          <th class="p-4">CODE</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-800/50 text-sm">
        {#each paginatedLogs as log}
          <tr class="hover:bg-slate-800/50 cursor-pointer" on:click={() => viewLogDetails(log)}>
            <td class="p-4">
              <span class="font-bold font-mono text-xs {log.is_success ? 'text-emerald-400' : 'text-red-400'}">
                {log.is_success ? "● UP" : "● DOWN"}
              </span>
            </td>
            <td class="p-4 text-cyan-50 font-mono font-bold">{log.api?.name || `API-${log.api_id}`}</td>
            <td class="p-4 font-mono text-xs text-slate-300">{formatDateTime(log.checked_at)}</td>
            <td class="p-4 font-mono text-xs text-slate-300">{log.response_time}ms</td>
            <td class="p-4"><span class="px-2 py-0.5 rounded border {log.is_success ? 'border-emerald-500/30 text-emerald-400' : 'border-red-500/30 text-red-400'}">{log.status_code || "ERR"}</span></td>
          </tr>
        {/each}
      </tbody>
    </table>
    
    <!-- Paginator -->
    {#if totalPages > 1}
      <div class="p-4 flex justify-between items-center bg-slate-950/50 border-t border-slate-800">
        <button on:click={() => currentPage--} disabled={currentPage === 1} class="px-3 py-1 bg-slate-800 rounded-lg text-xs disabled:opacity-50 font-mono">PREV</button>
        <span class="text-xs font-mono text-slate-500">PAGE {currentPage} OF {totalPages}</span>
        <button on:click={() => currentPage++} disabled={currentPage === totalPages} class="px-3 py-1 bg-slate-800 rounded-lg text-xs disabled:opacity-50 font-mono">NEXT</button>
      </div>
    {/if}
  </div>
</div>

<!-- Modal -->
{#if showLogModal && selectedLog}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <div class="absolute inset-0 bg-slate-950/80 backdrop-blur-sm" on:click={() => showLogModal = false}></div>
    <div class="relative bg-slate-900 border border-slate-700 rounded-2xl p-6 w-full max-w-lg shadow-2xl">
      <h3 class="text-lg font-bold text-cyan-400 font-mono mb-4">LOG_DETAILS</h3>
      <div class="space-y-4">
        <div class="bg-slate-950 p-4 rounded-xl border border-slate-800 font-mono text-xs">
          <pre class="text-cyan-50/70 whitespace-pre-wrap">{JSON.stringify(selectedLog, null, 2)}</pre>
        </div>
      </div>
      <button on:click={() => showLogModal = false} class="mt-6 w-full py-2 bg-slate-800 hover:bg-slate-700 rounded-xl font-bold text-slate-300 transition-colors">CLOSE</button>
    </div>
  </div>
{/if}

<!-- Report Template -->
<div id="pdf-report-export-container" style="display: none; position: absolute; left: -9999px; width: 210mm; padding: 20mm; background-color: #030712; color: #f8fafc; font-family: sans-serif;">
  <div style="border-bottom: 2px solid #06b6d4; padding-bottom: 5mm; margin-bottom: 10mm; display: flex; justify-content: space-between; align-items: center;">
    <div style="display: flex; align-items: center; gap: 4mm;">
      <div style="width: 12mm; height: 12mm; display: flex; align-items: center; justify-content: center; background: #0f172a; border-radius: 3mm; border: 1px solid #1e293b;">
        <img src={logoUrl} alt="Logo" width="40" height="40" style="width: 8mm; height: 8mm; object-fit: contain;"/>
      </div>
      <span style="font-size: 24px; font-weight: 900; color: #06b6d4; letter-spacing: -0.5px;">T-Monitor</span>
    </div>
    <div style="text-align: right; font-size: 14px; color: #64748b;">HEALTH_AUDIT_REPORT</div>
  </div>
  <div style="margin-bottom: 8mm; font-size: 12px; color: #64748b;">
    PROJECT: <span style="color: #fff;">{selectedProject?.name || "N/A"}</span> | PERIOD: <span style="color: #fff;">{startDate} - {endDate}</span>
  </div>
  <div id="report-metrics" style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 5mm; margin-bottom: 10mm;"></div>
  <div style="font-size: 14px; font-weight: bold; color: #06b6d4; margin-bottom: 4mm;">DIAGNOSTIC_RECOMMENDATIONS</div>
  <ul id="report-recommendations" style="list-style: none; padding: 0; color: #94a3b8; font-size: 12px;"></ul>
</div>

<style>
  .fade-in { animation: fadeIn 0.5s ease-out; }
  @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
  canvas { width: 100% !important; height: 100% !important; }
</style>
