<script lang="ts">
  import { page } from "$app/stores";
  import { onMount } from "svelte";
  import Swal from "sweetalert2";
  import { systemAlert, systemToast } from "$lib/swal-design";
  import { API_BASE_URL } from "$lib/config";

  $: projectId = $page.params.id;

  let project: any = null;
  let tasks: any[] = [];
  let isLoading = true;

  // Search and Pagination
  let searchQuery = "";
  let currentPage = 1;
  const itemsPerPage = 10;

  // Selected Task
  let selectedTask: any = null;

  // Form states
  let closeForm = { reason: "", document_url: "" };
  let failForm = { description: "" };

  $: filteredTasks = tasks.filter(task => {
    const searchLow = searchQuery.toLowerCase();
    const apiName = task.api?.name?.toLowerCase() || "";
    const apiUrl = task.api?.url?.toLowerCase() || "";
    const apiMethod = task.api?.method?.toLowerCase() || "";
    const errMsg = task.error_message?.toLowerCase() || "";
    return apiName.includes(searchLow) || apiUrl.includes(searchLow) || apiMethod.includes(searchLow) || errMsg.includes(searchLow);
  });

  $: paginatedTasks = filteredTasks.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage);
  $: totalPages = Math.ceil(filteredTasks.length / itemsPerPage);

  onMount(async () => {
    if (projectId) {
      await fetchData();
    }
  });

  async function fetchData() {
    isLoading = true;
    try {
      const token = localStorage.getItem("monitor_token");
      const [projRes, tasksRes] = await Promise.all([
        fetch(`${API_BASE_URL}/api/v1/projects/${projectId}`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
        fetch(`${API_BASE_URL}/api/v1/projects/${projectId}/repair-tasks`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
      ]);

      if (projRes.ok) project = await projRes.json();
      if (tasksRes.ok) tasks = await tasksRes.json();
    } catch (err) {
      console.error("Failed to fetch data:", err);
    } finally {
      isLoading = false;
    }
  }

  async function openApprove(task: any) {
    selectedTask = task;
    const result = await systemAlert.fire({
      title: 'APPROVE TASK?',
      html: `
        <div class="text-left space-y-3">
          <p class="text-sm text-slate-400">Are you sure you want to approve this task? This will set the status to <span class="text-blue-400 font-bold uppercase">Pending</span> and assign it to you for resolution.</p>
          <div class="p-3 bg-slate-950 border border-slate-800 rounded-xl">
            <p class="text-[10px] font-black text-slate-500 uppercase">API TARGET</p>
            <p class="text-xs text-slate-200 font-mono truncate">${task.api?.name || 'Unknown'}</p>
          </div>
        </div>
      `,
      icon: 'question',
      showCancelButton: true,
      confirmButtonText: 'CONFIRM APPROVAL',
      cancelButtonText: 'CANCEL',
      confirmButtonColor: '#0891b2',
    });

    if (result.isConfirmed) {
      handleApprove();
    }
  }

  async function handleApprove() {
    if (!selectedTask) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/repair-tasks/${selectedTask.id}/approve`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        await fetchData();
        systemToast.fire({ 
          icon: 'success', 
          title: 'Task Approved', 
        });
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function openClose(task: any) {
    selectedTask = task;
    const { value: formValues } = await systemAlert.fire({
      title: 'CLOSE REPAIR TASK',
      html: `
        <div class="text-left space-y-4">
          <div class="space-y-2">
            <label class="text-[10px] font-black text-slate-500 uppercase tracking-widest px-1">Resolution Reason</label>
            <textarea id="swal-reason" class="w-full bg-slate-950 border border-slate-800 rounded-2xl p-4 text-sm text-slate-300 focus:border-emerald-500/50 outline-none transition-all h-32" placeholder="Explain how the issue was resolved..."></textarea>
          </div>
          <div class="space-y-2">
            <label class="text-[10px] font-black text-slate-500 uppercase tracking-widest px-1">Upload Documents (Multiple)</label>
            <div class="relative group">
              <input id="swal-files" type="file" multiple class="hidden">
              <label for="swal-files" class="flex items-center justify-center gap-3 w-full bg-slate-900 border-2 border-dashed border-slate-800 hover:border-emerald-500/50 rounded-2xl p-4 text-slate-400 cursor-pointer transition-all hover:bg-slate-800/50">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                <span id="file-count" class="text-xs font-bold">Select files to upload...</span>
              </label>
            </div>
          </div>
          <div class="space-y-2">
            <label class="text-[10px] font-black text-slate-500 uppercase tracking-widest px-1">Legacy Documentation URL (Optional)</label>
            <input id="swal-url" type="text" class="w-full bg-slate-950 border border-slate-800 rounded-2xl p-4 text-sm text-slate-300 focus:border-emerald-500/50 outline-none transition-all" placeholder="https://jira.com/task-123">
          </div>
        </div>
      `,
      showCancelButton: true,
      confirmButtonText: 'FINISH & CLOSE',
      cancelButtonText: 'CANCEL',
      confirmButtonColor: '#059669',
      preConfirm: async () => {
        const reason = (document.getElementById('swal-reason') as HTMLTextAreaElement).value;
        const document_url = (document.getElementById('swal-url') as HTMLInputElement).value;
        const fileInput = document.getElementById('swal-files') as HTMLInputElement;
        
        if (!reason) {
          Swal.showValidationMessage('Please enter a resolution reason');
          return false;
        }

        let documents: string[] = [];
        if (fileInput.files && fileInput.files.length > 0) {
          Swal.showLoading();
          const formData = new FormData();
          for (let i = 0; i < fileInput.files.length; i++) {
            formData.append('files', fileInput.files[i]);
          }

          try {
            const token = localStorage.getItem("monitor_token");
            const uploadRes = await fetch(`${API_BASE_URL}/api/v1/upload`, {
              method: "POST",
              headers: { Authorization: `Bearer ${token}` },
              body: formData
            });
            if (uploadRes.ok) {
              const data = await uploadRes.json();
              documents = data.urls;
            } else {
              Swal.showValidationMessage('Failed to upload files');
              return false;
            }
          } catch (err) {
            Swal.showValidationMessage('Error uploading files');
            return false;
          }
        }

        return { reason, document_url, documents };
      },
      didRender: () => {
        const fileInput = systemAlert.getPopup()?.querySelector('#swal-files') as HTMLInputElement;
        const fileCount = systemAlert.getPopup()?.querySelector('#file-count') as HTMLElement;
        if (fileInput && fileCount) {
          fileInput.onchange = () => {
            const count = fileInput.files?.length || 0;
            fileCount.innerText = count > 0 ? `${count} file(s) selected` : 'Select files to upload...';
          };
        }
      }
    });

    if (formValues) {
      closeForm = formValues;
      handleClose();
    }
  }

  async function handleClose() {
    if (!selectedTask || !closeForm.reason) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/repair-tasks/${selectedTask.id}/close`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify(closeForm),
      });

      if (res.ok) {
        await fetchData();
        systemToast.fire({ 
          icon: 'success', 
          title: 'Task Closed', 
        });
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function openFail(task: any) {
    selectedTask = task;
    const { value: formValues } = await systemAlert.fire({
      title: 'MARK AS FAILED',
      html: `
        <div class="text-left space-y-4">
          <div class="space-y-2">
            <label class="text-[10px] font-black text-slate-500 uppercase tracking-widest px-1">Failure Description</label>
            <textarea id="swal-desc" class="w-full bg-slate-950 border border-slate-800 rounded-2xl p-4 text-sm text-slate-300 focus:border-rose-500/50 outline-none transition-all h-32" placeholder="Explain why this task could not be resolved..."></textarea>
          </div>
        </div>
      `,
      showCancelButton: true,
      confirmButtonText: 'CONFIRM FAILURE',
      cancelButtonText: 'CANCEL',
      confirmButtonColor: '#e11d48',
      preConfirm: () => {
        const description = (document.getElementById('swal-desc') as HTMLTextAreaElement).value;
        if (!description) {
          systemAlert.showValidationMessage('Please enter a failure description');
          return false;
        }
        return { description };
      },
    });

    if (formValues) {
      failForm = formValues;
      handleFail();
    }
  }

  async function handleFail() {
    if (!selectedTask || !failForm.description) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/repair-tasks/${selectedTask.id}/fail`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify(failForm),
      });

      if (res.ok) {
        await fetchData();
        systemToast.fire({ 
          icon: 'error', 
          title: 'Task Marked Failed', 
        });
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function viewDetails(task: any) {
    if (task.status === 'open' || task.status === 'pending') return;

    let html = '';
    if (task.status === 'closed') {
      let docsHtml = '';
      if (task.documents) {
        try {
          const docs = JSON.parse(task.documents);
          docsHtml = `
            <div class="mt-4 space-y-2">
              <p class="text-[10px] font-black text-slate-500 uppercase tracking-widest">ATTACHED DOCUMENTS</p>
              <div class="flex flex-wrap gap-2">
                ${docs.map((doc: string) => `
                  <a href="${API_BASE_URL}${doc}" target="_blank" class="flex items-center gap-2 px-3 py-2 bg-slate-900 border border-slate-800 rounded-xl hover:border-emerald-500/50 transition-all group">
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
                    <span class="text-xs text-slate-300 font-medium truncate max-w-[150px]">${doc.split('/').pop()}</span>
                  </a>
                `).join('')}
              </div>
            </div>
          `;
        } catch (e) {}
      } else if (task.document_url) {
        docsHtml = `
          <div class="mt-4 space-y-2">
            <p class="text-[10px] font-black text-slate-500 uppercase tracking-widest">EXTERNAL DOCUMENTATION</p>
            <a href="${task.document_url}" target="_blank" class="text-xs text-emerald-400 hover:underline font-mono break-all">${task.document_url}</a>
          </div>
        `;
      }

      html = `
        <div class="text-left space-y-4">
          <div class="space-y-1">
            <p class="text-[10px] font-black text-slate-500 uppercase tracking-widest px-1">RESOLUTION REASON</p>
            <div class="bg-slate-950 border border-slate-800 rounded-2xl p-4 text-sm text-slate-300 leading-relaxed">
              ${task.reason || 'No reason provided.'}
            </div>
          </div>
          ${docsHtml}
          <div class="pt-2 border-t border-slate-800/50">
             <p class="text-[10px] font-black text-slate-600 uppercase tracking-tighter">CLOSED BY SYSTEM AT ${new Date(task.closed_at).toLocaleString()}</p>
          </div>
        </div>
      `;
    } else if (task.status === 'failed') {
      html = `
        <div class="text-left space-y-4">
          <div class="space-y-1">
            <p class="text-[10px] font-black text-slate-500 uppercase tracking-widest px-1">FAILURE DESCRIPTION</p>
            <div class="bg-slate-950 border border-slate-800 rounded-2xl p-4 text-sm text-rose-300/80 leading-relaxed italic">
              "${task.description || 'No description provided.'}"
            </div>
          </div>
        </div>
      `;
    }

    await systemAlert.fire({
      title: task.status === 'closed' ? 'TASK RESOLUTION DETAIL' : 'TASK FAILURE DETAIL',
      html,
      showConfirmButton: true,
      confirmButtonText: 'CLOSE VIEW',
    });
  }

  function getStatusStyle(status: string) {
    switch (status) {
      case 'open': return 'bg-blue-500/20 text-blue-400 border-blue-500/50';
      case 'pending': return 'bg-amber-500/20 text-amber-400 border-amber-500/50';
      case 'closed': return 'bg-emerald-500/20 text-emerald-400 border-emerald-500/50';
      case 'failed': return 'bg-rose-500/20 text-rose-400 border-rose-500/50';
      default: return 'bg-slate-500/20 text-slate-400 border-slate-500/50';
    }
  }

  function nextPage() { if (currentPage < totalPages) currentPage++; }
  function prevPage() { if (currentPage > 1) currentPage--; }
  function setPage(p: number) { currentPage = p; }
</script>

<div class="p-8 max-w-7xl mx-auto space-y-8">
  <!-- Header Section -->
  <div class="flex flex-col md:flex-row md:items-center justify-between gap-6">
    <div class="space-y-1">
      <h1 class="text-3xl font-black tracking-tight text-white flex items-center gap-3">
        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-rose-500">
          <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>
        </svg>
        REPAIR API MONITOR
      </h1>
      <p class="text-slate-400 font-medium">Monitor and resolve API error tasks for <span class="text-rose-400 font-bold">{project?.name || 'Project'}</span></p>
    </div>

    <!-- Search Bar -->
    <div class="relative w-full md:w-96 group">
      <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-slate-500 group-focus-within:text-rose-400 transition-colors">
          <circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>
        </svg>
      </div>
      <input 
        type="text"
        bind:value={searchQuery}
        placeholder="Search tasks by API name, method or URL..."
        class="w-full bg-slate-900/50 border border-slate-800 rounded-2xl py-3 pl-11 pr-4 text-slate-200 text-sm focus:border-rose-500/50 outline-none transition-all placeholder:text-slate-600"
      />
    </div>
  </div>

  {#if isLoading}
    <div class="flex flex-col items-center justify-center py-24 space-y-4">
      <div class="w-12 h-12 border-4 border-rose-500/30 border-t-rose-500 rounded-full animate-spin"></div>
      <p class="text-slate-500 font-mono text-sm animate-pulse">FETCHING REPAIR TASKS...</p>
    </div>
  {:else if filteredTasks.length === 0}
    <div class="bg-slate-900/50 border border-slate-800 rounded-3xl p-12 text-center space-y-6">
      <div class="w-20 h-20 bg-slate-800/50 rounded-full flex items-center justify-center mx-auto text-slate-600">
        <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
      </div>
      <div class="space-y-2">
        <h3 class="text-xl font-bold text-slate-300">No results found</h3>
        <p class="text-slate-500">Try adjusting your search query to find what you're looking for.</p>
      </div>
    </div>
  {:else}
    <div class="grid grid-cols-1 gap-4">
      {#each paginatedTasks as task}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div 
          on:click={() => viewDetails(task)}
          class="bg-slate-900/40 border border-slate-800 hover:border-slate-700 transition-all rounded-2xl p-6 flex flex-col md:flex-row md:items-center justify-between gap-6 group {task.status === 'closed' || task.status === 'failed' ? 'cursor-pointer hover:bg-slate-800/30' : ''}"
        >
          <div class="flex items-start gap-4 flex-1 min-w-0">
             <div class="mt-1 w-10 h-10 rounded-xl flex items-center justify-center shrink-0 {task.status === 'open' ? 'bg-blue-500/10 text-blue-400' : task.status === 'pending' ? 'bg-amber-500/10 text-amber-400' : task.status === 'closed' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'}">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
             </div>
             <div class="space-y-1 min-w-0 flex-1">
               <div class="flex items-center gap-3">
                 <h3 class="font-bold text-slate-100 truncate">{task.api?.name || 'Unknown API'}</h3>
                 <span class="px-2.5 py-0.5 rounded-full text-[10px] font-black uppercase tracking-wider border {getStatusStyle(task.status)}">
                   {task.status}
                 </span>
               </div>
               <div class="flex items-center gap-2">
                 <span class="px-2 py-0.5 rounded-md bg-slate-800 text-[10px] font-black text-rose-400 border border-slate-700 uppercase tracking-tighter">{task.api?.method || 'N/A'}</span>
                 <p class="text-xs font-mono text-slate-500 truncate">{task.api?.url || 'N/A'}</p>
               </div>
               <p class="text-sm text-slate-400 line-clamp-2 mt-2">{task.error_message}</p>
               <p class="text-[10px] text-slate-600 uppercase font-bold tracking-tighter mt-1">Reported: {new Date(task.created_at).toLocaleString()}</p>
             </div>
          </div>

          <div class="flex items-center gap-3 shrink-0">
            {#if task.status === 'open'}
              <button 
                on:click|stopPropagation={() => openApprove(task)}
                class="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-xl text-xs font-bold transition-all shadow-lg shadow-blue-900/20"
              >
                APPROVE TASK
              </button>
            {:else if task.status === 'pending'}
               <button 
                on:click|stopPropagation={() => openFail(task)}
                class="px-4 py-2 border border-rose-500/30 text-rose-400 hover:bg-rose-500/10 rounded-xl text-xs font-bold transition-all"
              >
                FAIL TASK
              </button>
              <button 
                on:click|stopPropagation={() => openClose(task)}
                class="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl text-xs font-bold transition-all shadow-lg shadow-emerald-900/20"
              >
                CLOSE TASK
              </button>
            {:else if task.status === 'closed'}
               <div class="flex flex-col items-end gap-2">
                  <div class="text-right">
                    <p class="text-[10px] text-slate-500 font-bold uppercase">Closed at</p>
                    <p class="text-xs text-slate-400">{new Date(task.closed_at).toLocaleString()}</p>
                  </div>
                  {#if task.documents}
                    <div class="flex gap-1">
                      {#each JSON.parse(task.documents) as doc}
                        <a href="{API_BASE_URL}{doc}" target="_blank" class="w-7 h-7 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 hover:bg-emerald-500/20 transition-all" title="View Document">
                          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
                        </a>
                      {/each}
                    </div>
                  {:else if task.document_url}
                    <a href="{task.document_url}" target="_blank" class="text-[10px] font-black text-emerald-400 hover:underline uppercase">Legacy Doc ↗</a>
                  {/if}
               </div>
            {:else if task.status === 'failed'}
               <div class="text-right">
                  <p class="text-[10px] text-rose-500/70 font-bold uppercase">Resolution Failed</p>
                  <p class="text-xs text-slate-400 max-w-[200px] truncate">{task.description}</p>
               </div>
            {/if}
          </div>
        </div>
      {/each}
    </div>

    <!-- Pagination Controls -->
    {#if totalPages > 1}
      <div class="flex items-center justify-center gap-2 pt-6">
        <button 
          on:click={prevPage} 
          disabled={currentPage === 1}
          class="p-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-400 hover:text-white disabled:opacity-30 transition-all"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        </button>
        
        <div class="flex items-center gap-1">
          {#each Array(totalPages) as _, i}
            <button 
              on:click={() => setPage(i + 1)}
              class="w-10 h-10 rounded-xl font-bold text-xs transition-all border {currentPage === i + 1 ? 'bg-rose-600 border-rose-500 text-white shadow-lg shadow-rose-900/30' : 'bg-slate-900 border-slate-800 text-slate-500 hover:text-slate-300'}"
            >
              {i + 1}
            </button>
          {/each}
        </div>

        <button 
          on:click={nextPage} 
          disabled={currentPage === totalPages}
          class="p-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-400 hover:text-white disabled:opacity-30 transition-all"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
        </button>
      </div>
    {/if}
  {/if}
</div>
