<script lang="ts">
  import { onMount, onDestroy, tick } from "svelte";
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
  let reportChart: any = null;

  // Filter State
  let searchQuery = "";
  let statusFilter = "ALL"; // ALL, UP, DOWN

  // Modal State
  let showLogModal = false;
  let selectedLog: any = null;

  // Report State
  let startDate = new Date().toISOString().split("T")[0];
  let endDate = new Date().toISOString().split("T")[0];
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
                  const dataset = context[0].dataset as any;
                  const name = dataset.customData[dataIndex].name;
                  return `${name} • ${context[0].label}`;
                },
                afterTitle: (context) => {
                  const dataIndex = context[0].dataIndex;
                  const dataset = context[0].dataset as any;
                  const status = dataset.customData[dataIndex].status;
                  return `Status Code: ${status}`;
                },
                label: (context) => {
                  return `Response: ${context.raw} ms`;
                },
                afterLabel: (context) => {
                  const dataIndex = context.dataIndex;
                  const dataset = context.dataset as any;
                  const error = dataset.customData[dataIndex].error;
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

    // Initialize Flatpickr
    if ((window as any).flatpickr) {
      (window as any).flatpickr("#period-start", {
        dateFormat: "Y-m-d",
        defaultDate: startDate,
        onChange: (selectedDates: Date[], dateStr: string) => {
          startDate = dateStr;
        },
      });

      (window as any).flatpickr("#period-end", {
        dateFormat: "Y-m-d",
        defaultDate: endDate,
        onChange: (selectedDates: Date[], dateStr: string) => {
          endDate = dateStr;
        },
      });
    }
  });

  async function fetchProjectDetails() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/projects/${selectedProjectId}`,
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );
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
    const effectiveStart = startDate || endDate;
    const effectiveEnd = endDate || startDate;

    if (!effectiveStart) {
      systemAlert.fire({
        icon: "warning",
        title: "Date Selection Required",
        text: "Please select at least one date for the report.",
      });
      return;
    }

    isGeneratingReport = true;
    try {
      const token = localStorage.getItem("monitor_token");
      let url = `${API_BASE_URL}/api/v1/logs?project_id=${selectedProjectId}&start_date=${effectiveStart}&end_date=${effectiveEnd}`;
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
      if (Number(uptimePercent) < 95)
        recommendations.push(
          "⚠️ อัตราความพร้อมใช้งานต่ำกว่าเกณฑ์ ตรวจสอบความเสถียรของระบบ",
        );
      else recommendations.push("✅ ระบบทำงานปกติและมีความเสถียร");
      if (avgResponse > 1000)
        recommendations.push(
          "🐢 ตรวจพบความล่าช้าสูง ควรปรับปรุงประสิทธิภาพ Endpoint",
        );
      if (topFails.length > 0)
        recommendations.push(
          `🔴 ตรวจสอบข้อผิดพลาดที่: ${topFails.map((f) => f[0]).join(", ")}`,
        );

      // @ts-ignore
      const jspdfLib = window.jspdf;
      // @ts-ignore
      const html2canvas = window.html2canvas;

      if (!jspdfLib || !html2canvas) {
        throw new Error(
          "PDF layout libraries (jspdf/html2canvas) are still loading. Please try again in a few seconds.",
        );
      }

      // --- UI CLEANUP FOR CAPTURE ---
      // Hide open flatpickr calendars to avoid artifacts at bottom of screen
      const openPickers = document.querySelectorAll(".flatpickr-calendar.open");
      openPickers.forEach((p) => ((p as HTMLElement).style.display = "none"));

      await new Promise((resolve) => setTimeout(resolve, 500));

      const { jsPDF } = jspdfLib;
      const pdf = new jsPDF("p", "mm", "a4");
      const reportContent = document.getElementById(
        "pdf-report-export-container",
      );
      const errorPage = document.getElementById("report-error-page");

      if (reportContent) {
        reportContent.style.display = "block";
        const metricsContainer = reportContent.querySelector("#report-metrics");
        if (metricsContainer) {
          metricsContainer.innerHTML = `
            <div style="background: rgba(6, 182, 212, 0.05); border: 1px solid rgba(6, 182, 212, 0.1); padding: 5mm; border-radius: 4mm; text-align: center;">
              <div style="font-size: 10px; color: #64748b; font-weight: 800; margin-bottom: 2mm;">เวลาทำงานปกติ</div>
              <div style="font-size: 20px; font-weight: 900; color: #06b6d4;">${uptimePercent}%</div>
            </div>
            <div style="background: rgba(6, 182, 212, 0.05); border: 1px solid rgba(6, 182, 212, 0.1); padding: 5mm; border-radius: 4mm; text-align: center;">
              <div style="font-size: 10px; color: #64748b; font-weight: 800; margin-bottom: 2mm;">ความเร็วตอบสนองเฉลี่ย</div>
              <div style="font-size: 20px; font-weight: 900; color: #06b6d4;">${avgResponse}ms</div>
            </div>
            <div style="background: rgba(6, 182, 212, 0.05); border: 1px solid rgba(6, 182, 212, 0.1); padding: 5mm; border-radius: 4mm; text-align: center;">
              <div style="font-size: 10px; color: #64748b; font-weight: 800; margin-bottom: 2mm;">จำนวนบันทึก</div>
              <div style="font-size: 20px; font-weight: 900; color: #06b6d4;">${totalChecks}</div>
            </div>
          `;
        }
        const recList = reportContent.querySelector("#report-recommendations");
        if (recList) {
          recList.innerHTML = recommendations
            .map(
              (r) => `
            <li style="margin-bottom: 3mm; display: flex; align-items: flex-start; gap: 3mm;">
              <span style="color: #06b6d4;">•</span>
              <span>${r}</span>
            </li>
          `,
            )
            .join("");
        }

        await new Promise((resolve) => setTimeout(resolve, 300));

        const reportCanvas = document.getElementById(
          "report-chart-canvas",
        ) as HTMLCanvasElement;
        if (reportCanvas) {
          const chartData = [...reportLogs].slice(0, 50).reverse();
          const labels = chartData.map((log) =>
            new Intl.DateTimeFormat("th-TH", {
              hour: "2-digit",
              minute: "2-digit",
            }).format(new Date(log.checked_at)),
          );
          const backgroundColors = chartData.map((log) =>
            log.is_success
              ? "rgba(34, 197, 94, 0.8)"
              : "rgba(239, 68, 68, 0.8)",
          );
          const dataPoints = chartData.map((log) => log.response_time);

          if (reportChart) reportChart.destroy();

          reportChart = new Chart(reportCanvas, {
            type: "bar",
            data: {
              labels: labels,
              datasets: [
                {
                  label: "ความล่าช้า (ms)",
                  data: dataPoints,
                  backgroundColor: backgroundColors,
                  borderRadius: 4,
                },
              ],
            },
            options: {
              responsive: false, // Important for html2canvas
              animation: false, // No animation for capture
              plugins: { legend: { display: false } },
              scales: {
                y: {
                  beginAtZero: true,
                  grid: { color: "rgba(255, 255, 255, 0.05)" },
                  ticks: { color: "#64748b", font: { size: 10 } },
                },
                x: {
                  grid: { display: false },
                  ticks: {
                    color: "#64748b",
                    font: { size: 9 },
                    maxRotation: 0,
                  },
                },
              },
            },
          });
        }

        const failedLogs = reportLogs
          .filter((log: any) => !log.is_success)
          .slice(0, 110);
        const CHUNK_SIZE = 9;
        const totalErrorPages = Math.ceil(failedLogs.length / CHUNK_SIZE);

        // --- PAGE 1: DASHBOARD ---
        const canvas1 = await html2canvas(reportContent, {
          scale: 2,
          useCORS: true,
          backgroundColor: "#030712",
        });
        const imgHeight1 = (canvas1.height * 210) / canvas1.width;
        pdf.addImage(
          canvas1.toDataURL("image/png"),
          "PNG",
          0,
          0,
          210,
          Math.min(imgHeight1, 297),
        );

        // --- PAGE 2+ (MODULAR ERROR PAGES) ---
        if (errorPage) {
          errorPage.style.display = "block";

          for (let i = 0; i < totalErrorPages; i++) {
            const chunk = failedLogs.slice(
              i * CHUNK_SIZE,
              (i + 1) * CHUNK_SIZE,
            );

            // Explicitly Update Headers for each page capture to avoid reactivity issues
            const errorTableBody = errorPage.querySelector("#error-table-body");
            const pageNumSpan = errorPage.querySelector("#error-page-num");
            const projectNameHeader = errorPage.querySelector(
              "#report-error-project-name",
            );
            const dateRangeHeader = errorPage.querySelector(
              "#report-error-date-range",
            );

            if (projectNameHeader)
              projectNameHeader.textContent = selectedProject?.name || "N/A";
            if (dateRangeHeader)
              dateRangeHeader.textContent = `ช่วงเวลาตรวจสอบ: ${startDate || "N/A"} ถึง ${endDate || "N/A"}`;

            if (errorTableBody) {
              errorTableBody.innerHTML = chunk
                .map(
                  (log: any) => `
                <tr>
                  <td style="padding: 4mm 2mm; color: #64748b; border-bottom: 1px solid #1e293b; font-size: 9.5px; line-height: 1.4; vertical-align: middle;">
                    ${new Date(log.checked_at).toLocaleString("th-TH", {
                      year: "numeric",
                      month: "short",
                      day: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </td>
                  <td style="padding: 4mm 2mm; font-weight: 700; color: #e2e8f0; border-bottom: 1px solid #1e293b; font-size: 10px; word-break: break-all; vertical-align: middle; letter-spacing: 0.3px;">${log.api?.name || "Unknown"}</td>
                  <td style="padding: 4mm 2mm; text-align: center; border-bottom: 1px solid #1e293b; vertical-align: middle;">
                    <span style="display: inline-block; background: rgba(239, 68, 68, 0.15); color: #f87171; padding: 1mm 2.5mm; border-radius: 1.5mm; border: 1px solid rgba(239, 68, 68, 0.2); font-weight: 900; font-size: 8px; letter-spacing: 1px;">
                      ${log.status_code || "ERR"}
                    </span>
                  </td>
                  <td style="padding: 4mm 2mm; color: #94a3b8; border-bottom: 1px solid #1e293b; font-size: 9.5px; line-height: 1.6; word-break: break-word; vertical-align: middle;">${log.error_message || "N/A"}</td>
                </tr>
              `,
                )
                .join("");
            }

            if (pageNumSpan) {
              pageNumSpan.textContent = `หน้า ${i + 2}`;
            }

            // High delay for absolute render stability
            await new Promise((r) => setTimeout(r, 400));

            const canvasN = await html2canvas(errorPage, {
              scale: 2,
              useCORS: true,
              backgroundColor: "#030712",
            });

            const imgHeightN = (canvasN.height * 210) / canvasN.width;
            pdf.addPage();
            pdf.addImage(
              canvasN.toDataURL("image/png"),
              "PNG",
              0,
              0,
              210,
              Math.min(imgHeightN, 297),
            );
          }
          errorPage.style.display = "none";
        }

        const fileName = `T-Monitor-Report-${selectedProject?.name || "Project"}-${startDate}.pdf`;
        pdf.save(fileName);

        reportContent.style.display = "none";
        systemToast.fire({ icon: "success", title: "Report Generated" });
      }
    } catch (err: any) {
      console.error("Export error:", err);
      systemAlert.fire({
        icon: "error",
        title: "Export Failed",
        text:
          err.message || "An error occurred while generating the PDF report.",
      });
    } finally {
      isGeneratingReport = false;
    }
  }
</script>

<svelte:head>
  <script src="https://unpkg.com/jspdf@latest/dist/jspdf.umd.min.js"></script>
  <script
    src="https://unpkg.com/html2canvas@1.4.1/dist/html2canvas.min.js"
  ></script>
  <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
  <link
    rel="stylesheet"
    href="https://cdn.jsdelivr.net/npm/flatpickr/dist/flatpickr.min.css"
  />
  <script src="https://cdn.jsdelivr.net/npm/flatpickr"></script>
</svelte:head>

<div class="fade-in max-w-6xl mx-auto w-full overflow-hidden p-6">
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

    <div class="flex flex-wrap items-center gap-3">
      <div class="flex flex-col gap-1">
        <label
          for="period-start"
          class="text-[10px] font-bold text-slate-500 font-mono uppercase tracking-widest"
          >Period_Start</label
        >
        <div class="relative group">
          <input
            id="period-start"
            type="text"
            bind:value={startDate}
            placeholder="YYYY-MM-DD"
            class="bg-slate-900/50 border border-slate-700/50 hover:border-cyan-500/50 rounded-xl px-10 py-2.5 text-xs text-cyan-400 font-mono outline-none transition-all placeholder:text-slate-700 w-48"
          />
          <div
            class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-600 group-hover:text-cyan-500/70 transition-colors"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><rect x="3" y="4" width="18" height="18" rx="2" ry="2" /><line
                x1="16"
                y1="2"
                x2="16"
                y2="6"
              /><line x1="8" y1="2" x2="8" y2="6" /><line
                x1="3"
                y1="10"
                x2="21"
                y2="10"
              /></svg
            >
          </div>
        </div>
      </div>
      <div class="flex flex-col gap-1">
        <label
          for="period-end"
          class="text-[10px] font-bold text-slate-500 font-mono uppercase tracking-widest"
          >Period_End</label
        >
        <div class="relative group">
          <input
            id="period-end"
            type="text"
            bind:value={endDate}
            placeholder="YYYY-MM-DD"
            class="bg-slate-900/50 border border-slate-700/50 hover:border-cyan-500/50 rounded-xl px-10 py-2.5 text-xs text-cyan-400 font-mono outline-none transition-all placeholder:text-slate-700 w-48"
          />
          <div
            class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-600 group-hover:text-cyan-500/70 transition-colors"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><rect x="3" y="4" width="18" height="18" rx="2" ry="2" /><line
                x1="16"
                y1="2"
                x2="16"
                y2="6"
              /><line x1="8" y1="2" x2="8" y2="6" /><line
                x1="3"
                y1="10"
                x2="21"
                y2="10"
              /></svg
            >
          </div>
        </div>
      </div>
      <button
        onclick={generateReport}
        disabled={isGeneratingReport}
        class="mt-auto h-[38px] px-6 rounded-xl font-mono font-bold text-xs uppercase tracking-widest bg-gradient-to-r from-cyan-600/20 to-blue-600/20 text-cyan-400 border border-cyan-500/30"
      >
        {isGeneratingReport ? "EXPORTING..." : "EXPORT_PDF_REPORT"}
      </button>
    </div>
  </div>

  <!-- Summary Cards -->
  {#if isLoading}
    <div class="flex justify-center p-12">
      <svg
        class="animate-spin h-8 w-8 text-cyan-400"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle
          class="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          stroke-width="4"
        ></circle>
        <path
          class="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
        ></path>
      </svg>
    </div>
  {:else}
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-6 mb-8">
      <div
        class="bg-slate-800/40 rounded-3xl border border-slate-700/50 p-6 shadow-xl"
      >
        <p
          class="text-cyan-500/80 font-bold text-xs mb-1 uppercase tracking-widest font-mono"
        >
          MONITORED_ENDPOINTS
        </p>
        <h2 class="text-3xl font-black text-cyan-50">{summary.total}</h2>
      </div>
      <div
        class="bg-slate-800/40 rounded-3xl border border-slate-700/50 p-6 shadow-xl"
      >
        <p
          class="text-emerald-500/80 font-bold text-xs mb-1 uppercase tracking-widest font-mono"
        >
          OPERATIONAL [UP]
        </p>
        <h2 class="text-3xl font-black text-emerald-400">{summary.up}</h2>
      </div>
      <div
        class="bg-slate-800/40 rounded-3xl border border-slate-700/50 p-6 shadow-xl"
      >
        <p
          class="text-red-500/80 font-bold text-xs mb-1 uppercase tracking-widest font-mono"
        >
          FAILING [DOWN]
        </p>
        <h2 class="text-3xl font-black text-red-500">{summary.down}</h2>
      </div>
    </div>

    <!-- Performance Chart -->
    <div
      class="bg-slate-800/40 p-6 rounded-3xl border border-slate-700/50 shadow-xl mb-8"
    >
      <h3
        class="text-sm font-bold text-cyan-50 font-mono tracking-widest mb-6 uppercase flex items-center gap-2"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          class="text-cyan-400"
          ><path d="M3 3v18h18" /><path d="m19 9-5 5-4-4-3 3" /></svg
        >
        PERFORMANCE_HISTORY
      </h3>
      <div class="h-48 w-full"><canvas bind:this={chartCanvas}></canvas></div>
    </div>

    <!-- Filters -->
    <div
      class="bg-slate-900/60 p-4 rounded-t-3xl border border-slate-700/50 flex flex-col md:flex-row gap-4 items-center justify-between"
    >
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="SEARCH ENDPOINTS..."
        class="w-full md:max-w-md px-4 py-3 bg-slate-800 border border-slate-700 rounded-xl text-sm text-cyan-50 font-mono"
      />
      <div class="flex gap-2">
        <button
          onclick={() => (statusFilter = "ALL")}
          class="px-4 py-2 text-xs font-mono font-bold rounded-lg border {statusFilter ===
          'ALL'
            ? 'bg-cyan-900 border-cyan-500 text-cyan-400'
            : 'bg-slate-800 border-slate-700 text-slate-400'}">ALL</button
        >
        <button
          onclick={() => (statusFilter = "UP")}
          class="px-4 py-2 text-xs font-mono font-bold rounded-lg border {statusFilter ===
          'UP'
            ? 'bg-emerald-900 border-emerald-500 text-emerald-400'
            : 'bg-slate-800 border-slate-700 text-slate-400'}">UP</button
        >
        <button
          onclick={() => (statusFilter = "DOWN")}
          class="px-4 py-2 text-xs font-mono font-bold rounded-lg border {statusFilter ===
          'DOWN'
            ? 'bg-red-900 border-red-500 text-red-400'
            : 'bg-slate-800 border-slate-700 text-slate-400'}">DOWN</button
        >
      </div>
    </div>

    <!-- Table -->
    <div
      class="bg-slate-900/60 rounded-b-3xl border border-t-0 border-slate-700/50 overflow-hidden"
    >
      <table class="w-full text-left whitespace-nowrap">
        <thead>
          <tr
            class="bg-slate-950/80 border-b border-slate-700/50 text-[10px] font-bold text-slate-400 uppercase font-mono"
          >
            <th class="p-4">STATUS</th>
            <th class="p-4">ENDPOINT</th>
            <th class="p-4">CHECK_TIME</th>
            <th class="p-4">LATENCY</th>
            <th class="p-4">CODE</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/50 text-sm">
          {#each paginatedLogs as log}
            <tr
              class="hover:bg-slate-800/50 cursor-pointer"
              onclick={() => viewLogDetails(log)}
            >
              <td class="p-4">
                <span
                  class="font-bold font-mono text-xs {log.is_success
                    ? 'text-emerald-400'
                    : 'text-red-400'}"
                >
                  {log.is_success ? "● UP" : "● DOWN"}
                </span>
              </td>
              <td class="p-4 text-cyan-50 font-mono font-bold"
                >{log.api?.name || `API-${log.api_id}`}</td
              >
              <td class="p-4 font-mono text-xs text-slate-300"
                >{formatDateTime(log.checked_at)}</td
              >
              <td class="p-4 font-mono text-xs text-slate-300"
                >{log.response_time}ms</td
              >
              <td class="p-4"
                ><span
                  class="px-2 py-0.5 rounded border {log.is_success
                    ? 'border-emerald-500/30 text-emerald-400'
                    : 'border-red-500/30 text-red-400'}"
                  >{log.status_code || "ERR"}</span
                ></td
              >
            </tr>
          {/each}
        </tbody>
      </table>

      <!-- Paginator -->
      {#if totalPages > 1}
        <div
          class="p-4 flex justify-between items-center bg-slate-950/50 border-t border-slate-800"
        >
          <button
            onclick={() => currentPage--}
            disabled={currentPage === 1}
            class="px-3 py-1 bg-slate-800 rounded-lg text-xs disabled:opacity-50 font-mono tracking-widest hover:bg-slate-700 transition-colors"
            >ก่อนหน้า</button
          >
          <span
            class="text-xs font-mono text-slate-500 uppercase tracking-widest"
            >หน้า <span class="text-cyan-400">{currentPage}</span> จาก {totalPages}</span
          >
          <button
            onclick={() => currentPage++}
            disabled={currentPage === totalPages}
            class="px-3 py-1 bg-slate-800 rounded-lg text-xs disabled:opacity-50 font-mono tracking-widest hover:bg-slate-700 transition-colors"
            >ถัดไป</button
          >
        </div>
      {/if}
    </div>
  {/if}
</div>

<!-- Modal -->
{#if showLogModal && selectedLog}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <div
      role="presentation"
      class="absolute inset-0 bg-slate-950/80 backdrop-blur-sm"
      onclick={() => (showLogModal = false)}
      onkeydown={(e) => e.key === "Escape" && (showLogModal = false)}
    ></div>
    <div
      class="relative bg-slate-900 border border-slate-700 rounded-2xl p-6 w-full max-w-lg shadow-2xl"
    >
      <h3 class="text-lg font-bold text-cyan-400 font-mono mb-4">
        LOG_DETAILS
      </h3>
      <div class="space-y-4">
        <div
          class="bg-slate-950 p-4 rounded-xl border border-slate-800 font-mono text-xs"
        >
          <pre class="text-cyan-50/70 whitespace-pre-wrap">{JSON.stringify(
              selectedLog,
              null,
              2,
            )}</pre>
        </div>
      </div>
      <button
        onclick={() => (showLogModal = false)}
        class="mt-6 w-full py-2 bg-slate-800 hover:bg-slate-700 rounded-xl font-bold text-slate-300 transition-colors"
        >CLOSE</button
      >
    </div>
  </div>
{/if}

<!-- Report Template -->
<div
  id="pdf-report-export-container"
  style="display: none; position: absolute; left: -9999px; width: 210mm; min-height: 297mm; padding: 20mm; background-color: #030712; color: #f8fafc; font-family: 'Inter', sans-serif; box-sizing: border-box;"
>
  <div
    style="border-bottom: 2px solid #06b6d4; padding-bottom: 5mm; margin-bottom: 10mm; display: flex; justify-content: space-between; align-items: center;"
  >
    <div style="display: flex; align-items: center; gap: 5mm; margin-bottom: 12mm;">
      <div
        style="width: 18mm; height: 18mm; position: relative; background: #0f172a; border-radius: 4.5mm; border: 1.5px solid #1e293b; overflow: hidden; display: flex; align-items: center; justify-content: center;"
      >
        <img
          src={logoUrl}
          alt="Logo"
          style="width: 400%; height: 400%; position: absolute; bottom: -25%; right: -25%; object-fit: contain;"
        />
      </div>
      <div>
        <div
          style="font-size: 26px; font-weight: 900; color: #06b6d4; letter-spacing: -1.5px; line-height: 1;"
        >
          T-Monitor
        </div>
        <div
          style="font-size: 11px; font-weight: 700; color: #64748b; margin-top: 1.5mm; text-transform: uppercase; letter-spacing: 1.5px;"
        >
          รายงานการตรวจสอบสถานะระบบ
        </div>
      </div>
    </div>
    <div
      style="text-align: right; font-size: 14px; color: #64748b; font-weight: 800; letter-spacing: 1px;"
    >
      รายงานการตรวจสอบสถานะระบบ
    </div>
  </div>

  <div
    style="margin-bottom: 12mm; font-size: 12px; color: #64748b; font-weight: 600; letter-spacing: 0.5px;"
  >
    โครงการ: <span style="color: #fff; font-weight: 800;"
      >{selectedProject?.name || "N/A"}</span
    > <span style="margin: 0 4mm; opacity: 0.3;">|</span> ช่วงเวลา:
    <span style="color: #fff; font-weight: 800;">{startDate} ถึง {endDate}</span
    >
  </div>

  <div
    id="report-metrics"
    style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 8mm; margin-bottom: 15mm;"
  >
    <!-- Filled Dynamically -->
  </div>

  <div
    style="margin-bottom: 15mm; background: rgba(15, 23, 42, 0.4); border: 1px solid #1e293b; border-radius: 6mm; padding: 8mm;"
  >
    <div
      style="font-size: 12px; font-weight: 800; color: #06b6d4; margin-bottom: 6mm; letter-spacing: 1px; text-transform: uppercase;"
    >
      ประวัติประสิทธิภาพ [ภาพรวม]
    </div>
    <div
      style="width: 100%; height: 280px; display: flex; align-items: center; justify-content: center;"
    >
      <canvas id="report-chart-canvas" width="600" height="260"></canvas>
    </div>
  </div>

  <div
    style="padding: 8mm; background: rgba(6, 182, 212, 0.03); border: 1px solid rgba(6, 182, 212, 0.1); border-radius: 6mm;"
  >
    <div
      style="font-size: 12px; font-weight: 800; color: #06b6d4; margin-bottom: 6mm; letter-spacing: 1px; text-transform: uppercase;"
    >
      คำแนะนำในการวินิจฉัย
    </div>
    <ul
      id="report-recommendations"
      style="list-style: none; padding: 0; color: #94a3b8; font-size: 12px; line-height: 1.8;"
    >
      <!-- Filled Dynamically -->
    </ul>
  </div>

  <div
    style="position: absolute; bottom: 20mm; left: 20mm; right: 20mm; border-top: 1px solid #1e293b; padding-top: 5mm; display: flex; justify-content: space-between; align-items: center; font-size: 9px; color: #334155; font-weight: 600; text-transform: uppercase; letter-spacing: 1px;"
  >
    <span>สร้างโดยระบบอัตโนมัติ • {new Date().toLocaleString("th-TH")}</span>
    <span>ลับเฉพาะ • การตรวจสอบภายใน</span>
  </div>
</div>

<!-- SECOND PAGE: Error Logs template -->
<div
  id="report-error-page"
  style="display: none; position: absolute; left: -9999px; width: 210mm; min-height: 297mm; padding: 20mm; background-color: #030712; color: #f8fafc; font-family: 'Inter', sans-serif; box-sizing: border-box;"
>
  <div
    style="border-bottom: 2px solid #ef4444; padding-bottom: 5mm; margin-bottom: 10mm; display: flex; justify-content: space-between; align-items: center;"
  >
    <div style="display: flex; align-items: center; gap: 5mm;">
      <div
        style="width: 18mm; height: 18mm; position: relative; background: #0f172a; border-radius: 4.5mm; border: 1.5px solid #1e293b; overflow: hidden; display: flex; align-items: center; justify-content: center;"
      >
        <img
          src={logoUrl}
          alt="Logo"
          style="width: 400%; height: 400%; position: absolute; bottom: -25%; right: -25%; object-fit: contain;"
        />
      </div>
      <div>
        <div
          style="font-size: 20px; font-weight: 900; color: #06b6d4; letter-spacing: -0.5px; line-height: 1;"
        >
          T-Monitor
        </div>
        <div
          style="font-size: 11px; font-weight: 700; color: #ef4444; margin-top: 1.5mm;"
        >
          รายละเอียดข้อผิดพลาด (Error Logs)
        </div>
      </div>
    </div>
    <div style="text-align: right;">
      <div
        style="font-size: 9px; color: #64748b; font-weight: 800; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 2mm;"
      >
        โครงการ: <span id="report-error-project-name" style="color: #fff;"
          >N/A</span
        >
      </div>
      <div
        style="font-size: 9px; color: #64748b; font-weight: 800; text-transform: uppercase; letter-spacing: 1px;"
      >
        รายงานระบบ • <span id="error-page-num" style="color: #06b6d4;"
          >หน้า 2</span
        >
      </div>
    </div>
  </div>

  <div
    id="report-error-date-range"
    style="margin-bottom: 8mm; font-size: 10px; color: #64748b; font-weight: 600; padding: 3mm 4mm; background: rgba(15, 23, 42, 0.4); border-radius: 3mm; border: 1px solid #1e293b;"
  >
    ช่วงเวลาตรวจสอบ: N/A
  </div>

  <table
    style="width: 100%; border-collapse: collapse; font-size: 11px; table-layout: fixed;"
  >
    <thead>
      <tr
        style="color: #64748b; text-transform: uppercase; font-weight: 800; border-bottom: 2px solid #334155;"
      >
        <th style="padding: 3mm 2mm; text-align: left; width: 16%;"
          >เวลาที่ตรวจสอบ</th
        >
        <th style="padding: 3mm 2mm; text-align: left; width: 24%;">ENDPOINT</th
        >
        <th style="padding: 3mm 2mm; text-align: center; width: 10%;">CODE</th>
        <th style="padding: 3mm 2mm; text-align: left; width: 50%;"
          >รายละเอียดความผิดพลาด</th
        >
      </tr>
    </thead>
    <tbody id="error-table-body">
      <!-- To be filled dynamically -->
    </tbody>
  </table>

  <div
    style="position: absolute; bottom: 20mm; left: 20mm; right: 20mm; border-top: 1px solid #1e293b; padding-top: 5mm; display: flex; justify-content: space-between; align-items: center; font-size: 9px; color: #334155; font-weight: 600; text-transform: uppercase; letter-spacing: 1px;"
  >
    <span>สรุปข้อมูลผิดพลาดประจำวัน</span>
    <span>ลับเฉพาะ • เอกสารแนบท้าย</span>
  </div>
</div>

<style>
  .fade-in {
    animation: fadeIn 0.5s ease-out;
  }
  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
  canvas {
    width: 100% !important;
    height: 100% !important;
  }

  :global(.flatpickr-calendar) {
    background: rgba(15, 23, 42, 0.95) !important;
    backdrop-filter: blur(16px) !important;
    border: 1px solid rgba(51, 65, 85, 0.5) !important;
    border-radius: 20px !important;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5) !important;
    font-family: "Inter", sans-serif !important;
    margin-top: 10px !important;
  }

  :global(.flatpickr-calendar:before),
  :global(.flatpickr-calendar:after) {
    display: none !important;
  }

  :global(.flatpickr-day) {
    color: #94a3b8 !important;
    border-radius: 12px !important;
    transition: all 0.2s ease !important;
  }

  :global(.flatpickr-day.today) {
    border-color: rgba(6, 182, 212, 0.3) !important;
    color: #06b6d4 !important;
    font-weight: bold !important;
  }

  :global(.flatpickr-day:hover) {
    background: rgba(6, 182, 212, 0.1) !important;
    color: #06b6d4 !important;
  }

  :global(.flatpickr-day.selected) {
    background: #06b6d4 !important;
    color: #fff !important;
    font-weight: 800 !important;
    box-shadow: 0 0 20px rgba(6, 182, 212, 0.3) !important;
    border: none !important;
  }

  :global(.flatpickr-months .flatpickr-month) {
    color: #fff !important;
    height: 40px !important;
  }

  :global(.flatpickr-current-month .flatpickr-monthDropdown-months) {
    font-weight: 800 !important;
    text-transform: uppercase !important;
    letter-spacing: 1px !important;
    font-size: 13px !important;
  }

  :global(.flatpickr-weekday) {
    color: #64748b !important;
    font-size: 10px !important;
    font-weight: 800 !important;
    text-transform: uppercase !important;
    letter-spacing: 1px !important;
  }

  :global(.flatpickr-calendar .flatpickr-innerContainer) {
    padding: 10px !important;
  }

  :global(.flatpickr-prev-month svg),
  :global(.flatpickr-next-month svg) {
    fill: #06b6d4 !important;
    width: 14px !important;
    height: 14px !important;
  }

  :global(.flatpickr-prev-month:hover svg),
  :global(.flatpickr-next-month:hover svg) {
    fill: #fff !important;
  }
</style>
