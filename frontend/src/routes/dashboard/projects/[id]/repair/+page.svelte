<script lang="ts">
  import { page } from "$app/stores";
  import { onMount } from "svelte";
  import Swal from "sweetalert2";
  import { systemAlert, systemToast } from "$lib/swal-design";
  import { API_BASE_URL } from "$lib/config";

  $: projectId = $page.params.id;

  interface User {
    id: string;
    email: string;
    name?: string;
  }

  interface Member {
    id: string;
    email: string;
    user?: User;
  }

  interface Project {
    id: string;
    name: string;
    user?: User;
  }

  interface RepairTask {
    id: string;
    status: string;
    reason?: string;
    description?: string;
    fixer_name?: string;
    document_url?: string;
    documents?: string | string[];
    closed_at: string;
    created_at?: string;
    project_id: string;
    api?: {
      name: string;
      url: string;
      method: string;
    };
    approver?: Member;
    error_message?: string;
  }

  let project: Project | null = null;
  let tasks: RepairTask[] = [];
  let members: Member[] = [];
  let isLoading = true;

  // Search and Pagination
  let searchQuery = "";
  let currentPage = 1;
  const itemsPerPage = 10;

  // Selected Task
  let selectedTask: RepairTask | null = null;

  $: filteredTasks = tasks.filter(task => {
    const searchLow = searchQuery.toLowerCase();
    const apiName = task.api?.name?.toLowerCase() || "";
    const apiUrl = task.api?.url?.toLowerCase() || "";
    const apiMethod = task.api?.method?.toLowerCase() || "";
    const errMsg = (task as any).error_message?.toLowerCase() || "";
    return apiName.includes(searchLow) || apiUrl.includes(searchLow) || apiMethod.includes(searchLow) || errMsg.includes(searchLow);
  });

  $: paginatedTasks = filteredTasks.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage);
  $: totalPages = Math.ceil(filteredTasks.length / itemsPerPage);

  onMount(async () => {
    if (projectId) {
      fetchData();
    }
  });

  async function fetchData() {
    isLoading = true;
    try {
      const token = localStorage.getItem("monitor_token");
      const [projRes, tasksRes, membersRes] = await Promise.all([
        fetch(`${API_BASE_URL}/api/v1/projects/${projectId}`, { headers: { Authorization: `Bearer ${token}` } }),
        fetch(`${API_BASE_URL}/api/v1/projects/${projectId}/repair-tasks`, { headers: { Authorization: `Bearer ${token}` } }),
        fetch(`${API_BASE_URL}/api/v1/companies/members`, { headers: { Authorization: `Bearer ${token}` } })
      ]);

      if (projRes.ok) project = await projRes.json();
      if (tasksRes.ok) tasks = await tasksRes.json();
      if (membersRes.ok) members = await membersRes.json();
    } catch (err) {
      console.error(err);
      systemToast.fire({ icon: 'error', title: 'Connection error' });
    } finally {
      isLoading = false;
    }
  }

  async function openApprove(task: RepairTask) {
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

  function getStatusStyle(status: string) {
    if (status === 'open') return 'border-blue-500/30 text-blue-400';
    if (status === 'pending') return 'border-amber-500/30 text-amber-400';
    if (status === 'closed') return 'border-emerald-500/30 text-emerald-400';
    return 'border-rose-500/30 text-rose-400';
  }

  function viewDetails(task: RepairTask) {
    if (task.status === 'open' || task.status === 'pending') return;
    
    selectedTask = task;
    systemAlert.fire({
      title: 'REPAIR_DETAILS',
      html: `
        <div class="text-left space-y-4">
          <div class="p-4 bg-slate-950 border border-slate-800 rounded-2xl">
            <p class="text-[10px] font-black text-slate-500 uppercase tracking-widest mb-2">Technician_Summary</p>
            <p class="text-sm text-slate-300 leading-relaxed">${task.description || 'No description provided.'}</p>
          </div>
          <div class="grid grid-cols-2 gap-3">
             <div class="p-3 bg-slate-800/50 rounded-xl border border-slate-700/30">
               <p class="text-[9px] font-bold text-slate-500 uppercase">Fixed_By</p>
               <p class="text-xs text-emerald-400 font-bold">${task.fixer_name || 'N/A'}</p>
             </div>
             <div class="p-3 bg-slate-800/50 rounded-xl border border-slate-700/30">
               <p class="text-[9px] font-bold text-slate-500 uppercase">Status</p>
               <p class="text-xs text-rose-400 font-bold">${task.status.toUpperCase()}</p>
             </div>
          </div>
        </div>
      `,
      confirmButtonText: 'CLOSE_PREVIEW'
    });
  }

  function prevPage() { if (currentPage > 1) currentPage--; }
  function nextPage() { if (currentPage < totalPages) currentPage++; }
  function setPage(p: number) { currentPage = p; }

  async function openFail(task: RepairTask) {
    const { value: reason } = await systemAlert.fire({
      title: 'FAIL_REPAIR_TASK',
      input: 'textarea',
      inputLabel: 'Reason for Failure',
      inputPlaceholder: 'Why did the repair fail?',
      inputAttributes: { 'aria-label': 'Why did the repair fail?' },
      showCancelButton: true,
      confirmButtonText: 'MARK_AS_FAILED',
      confirmButtonColor: '#e11d48'
    });

    if (reason) {
      handleFail(task.id, reason);
    }
  }

  async function handleFail(taskId: string, reason: string) {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/repair-tasks/${taskId}/fail`, {
        method: "POST",
        headers: { 
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ description: reason })
      });

      if (res.ok) {
        await fetchData();
        systemToast.fire({ icon: 'success', title: 'Task Marked Failed' });
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function openClose(task: any) {
    selectedTask = task;
    
    // Prepare list of potential fixers (Owner + Members)
    const fixerOptions = [];
    if (project?.user) {
        fixerOptions.push(project.user.name || project.user.email);
    }
    members.forEach(m => {
        const name = (m.user?.name || m.user?.email) as string;
        if (name && !fixerOptions.includes(name)) {
            fixerOptions.push(name);
        }
    });

    // Add historical fixer names from previously closed tasks
    tasks.forEach(t => {
        if (t.fixer_name && !fixerOptions.includes(t.fixer_name)) {
            fixerOptions.push(t.fixer_name);
        }
    });

    const { value: formValues } = await systemAlert.fire({
      title: 'CLOSE REPAIR TASK',
      html: `
        <div class="text-left space-y-5 p-1">
          <div class="space-y-2 relative">
            <div class="flex items-center justify-between">
              <label class="text-[10px] font-black text-slate-500 uppercase tracking-[0.2em]">Repair Performed By</label>
              <div class="flex items-center gap-2">
                <span id="label-mode" class="text-[9px] font-bold text-emerald-500/50 uppercase tracking-tight">Search Team</span>
                <div class="w-px h-2 bg-slate-800"></div>
                <button id="toggle-fixer-input" class="text-[10px] font-bold text-emerald-500 hover:text-emerald-400 flex items-center gap-1 transition-all uppercase group">
                    <span id="toggle-text">Add Custom</span>
                    <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" class="group-hover:rotate-90 transition-transform"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
                </button>
              </div>
            </div>
            
            <div id="fixer-select-container" class="relative group">
              <div class="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500 group-focus-within:text-emerald-500 transition-colors">
                <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
              </div>
              <input 
                id="swal-fixer-search" 
                type="text" 
                class="w-full bg-slate-950/50 border border-slate-800 rounded-2xl py-4 pl-12 pr-12 text-sm text-slate-200 placeholder:text-slate-600 focus:border-emerald-500/50 focus:ring-4 focus:ring-emerald-500/5 outline-none transition-all cursor-pointer" 
                placeholder="Search team or select from list..."
                readonly
              >
              <div class="absolute right-4 top-1/2 -translate-y-1/2 text-slate-600 group-focus-within:text-emerald-500 pointer-events-none transition-all group-focus-within:rotate-180">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
              </div>

              <!-- Custom Dropdown Menu -->
              <div id="fixer-dropdown-menu" class="hidden absolute top-full left-0 right-0 mt-2 bg-slate-900/95 backdrop-blur-xl border border-slate-800 rounded-2xl shadow-2xl z-[100] max-h-60 overflow-y-auto scrollbar-hide animate-in fade-in slide-in-from-top-2 duration-200">
                <div class="p-2 space-y-1">
                  ${fixerOptions.length > 0 ? fixerOptions.map(opt => `
                    <button class="fixer-option w-full text-left px-4 py-3 text-xs font-medium text-slate-400 hover:text-emerald-400 hover:bg-emerald-500/10 rounded-xl transition-all flex items-center justify-between group/opt" data-value="${opt}">
                      <span>${opt}</span>
                      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="opacity-0 group-hover/opt:opacity-100 transition-opacity"><polyline points="20 6 9 17 4 12"/></svg>
                    </button>
                  `).join('') : '<p class="p-4 text-[10px] text-slate-600 text-center uppercase tracking-widest italic">No members found</p>'}
                </div>
              </div>
            </div>

            <div id="fixer-input-container" class="hidden relative group">
              <div class="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500 group-focus-within:text-emerald-500 transition-colors">
                <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
              </div>
              <input id="swal-fixer-custom" type="text" class="w-full bg-slate-950/50 border border-slate-800 rounded-2xl py-4 pl-12 pr-4 text-sm text-slate-200 placeholder:text-slate-600 focus:border-emerald-500/50 focus:ring-4 focus:ring-emerald-500/5 outline-none transition-all" placeholder="Enter name of the technician...">
            </div>
          </div>

          <div class="space-y-2">
            <label class="text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] px-1">Resolution Reason</label>
            <div class="relative group">
               <textarea id="swal-reason" class="w-full bg-slate-950/50 border border-slate-800 rounded-2xl p-4 text-sm text-slate-300 focus:border-emerald-500/50 focus:ring-4 focus:ring-emerald-500/5 outline-none transition-all h-32 leading-relaxed resize-none scrollbar-hide" placeholder="Briefly describe how you fixed this endpoint..."></textarea>
            </div>
          </div>

          <div class="grid grid-cols-1 gap-4">
            <div class="space-y-2">
              <label class="text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] px-1">Evidence & Logs</label>
              <div class="relative group">
                <input id="swal-files" type="file" multiple class="hidden">
                <label for="swal-files" class="flex flex-col items-center justify-center gap-2 w-full bg-slate-950/30 border-2 border-dashed border-slate-800/50 hover:border-emerald-500/30 hover:bg-emerald-500/5 rounded-2xl p-6 text-slate-500 cursor-pointer transition-all">
                  <div class="w-10 h-10 rounded-full bg-slate-900 flex items-center justify-center border border-slate-800 group-hover:border-emerald-500/30 group-hover:bg-emerald-500/10 transition-all">
                    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="group-hover:text-emerald-400 transition-colors"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                  </div>
                  <span id="file-count" class="text-[10px] font-bold uppercase tracking-widest text-slate-600 group-hover:text-emerald-500/70">Upload Proof (Multiple)</span>
                </label>
              </div>
              <!-- Added file list container -->
              <div id="swal-file-list" class="space-y-2 mt-2"></div>
            </div>

            <div class="space-y-2">
              <label class="text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] px-1">Case Tracking URL (Optional)</label>
              <div class="relative group">
                <div class="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500 group-focus-within:text-emerald-500 transition-colors">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
                </div>
                <input id="swal-url" type="text" class="w-full bg-slate-950/30 border border-slate-800 rounded-2xl py-4 pl-12 pr-4 text-xs text-emerald-400/70 placeholder:text-slate-700 focus:border-emerald-500/50 outline-none transition-all font-mono" placeholder="Jira, Trello, or Slack link...">
              </div>
            </div>
          </div>
        </div>
      `,
      showCancelButton: true,
      confirmButtonText: 'FINISH & CLOSE',
      cancelButtonText: 'CANCEL',
      confirmButtonColor: '#059669',
      preConfirm: async () => {
        const isCustom = !document.getElementById('fixer-input-container')?.classList.contains('hidden');
        const fixer_name = isCustom 
          ? (document.getElementById('swal-fixer-custom') as HTMLInputElement).value
          : (document.getElementById('swal-fixer-search') as HTMLInputElement).value;

        const reason = (document.getElementById('swal-reason') as HTMLTextAreaElement).value;
        const document_url = (document.getElementById('swal-url') as HTMLInputElement).value;
        
        const filesToUpload = (window as any).__swal_files || [];
        
        if (!fixer_name || fixer_name === '-- Select Person --') {
          Swal.showValidationMessage('Identity of the fixer is required');
          return false;
        }
        if (!reason) {
          Swal.showValidationMessage('A resolution summary is required');
          return false;
        }

        let documents: string[] = [];
        if (filesToUpload.length > 0) {
          Swal.showLoading();
          const formData = new FormData();
          for (let i = 0; i < filesToUpload.length; i++) {
            formData.append('files', filesToUpload[i]);
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
              Swal.showValidationMessage('Artifact upload failed');
              return false;
            }
          } catch (err) {
            Swal.showValidationMessage('Network error during upload');
            return false;
          }
        }

        delete (window as any).__swal_files;
        return { reason, document_url, documents, fixer_name };
      },
      didRender: () => {
        const toggleBtn = systemAlert.getPopup()?.querySelector('#toggle-fixer-input') as HTMLButtonElement;
        const selectContainer = systemAlert.getPopup()?.querySelector('#fixer-select-container') as HTMLElement;
        const inputContainer = systemAlert.getPopup()?.querySelector('#fixer-input-container') as HTMLElement;
        const toggleText = systemAlert.getPopup()?.querySelector('#toggle-text') as HTMLElement;
        const labelMode = systemAlert.getPopup()?.querySelector('#label-mode') as HTMLElement;

        const searchInput = systemAlert.getPopup()?.querySelector('#swal-fixer-search') as HTMLInputElement;
        const dropdownMenu = systemAlert.getPopup()?.querySelector('#fixer-dropdown-menu') as HTMLElement;
        const fixerOptions = systemAlert.getPopup()?.querySelectorAll('.fixer-option') as NodeListOf<HTMLButtonElement>;

        const fileInput = systemAlert.getPopup()?.querySelector('#swal-files') as HTMLInputElement;
        const fileCount = systemAlert.getPopup()?.querySelector('#file-count') as HTMLElement;
        const fileListContainer = systemAlert.getPopup()?.querySelector('#swal-file-list') as HTMLElement;

        let selectedFiles: File[] = [];

        const updateFileListUI = () => {
          if (!fileListContainer || !fileCount) return;
          
          if (selectedFiles.length === 0) {
            fileListContainer.innerHTML = `
              <div class="flex flex-col items-center justify-center p-8 bg-slate-950/20 border border-dashed border-slate-800 rounded-2xl text-slate-600">
                <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="opacity-20 mb-2"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
                <span class="text-[10px] font-black uppercase tracking-[0.2em] opacity-40">No evidence attached yet</span>
              </div>
            `;
            fileCount.innerText = 'Upload Proof (Multiple)';
            return;
          }

          fileListContainer.innerHTML = `
            <div class="flex items-center justify-between mb-3 px-1">
              <div class="flex items-center gap-2">
                <p class="text-[10px] font-black text-slate-500 uppercase tracking-[0.1em]">PREPARED EVIDENCE</p>
                <span class="px-1.5 py-0.5 bg-emerald-500/10 border border-emerald-500/20 rounded text-[9px] font-black text-emerald-500">${selectedFiles.length}</span>
              </div>
              <button id="clear-all-files" class="text-[10px] font-black text-rose-500/60 hover:text-rose-500 transition-colors uppercase tracking-tight hover:underline flex items-center gap-1 group/clear">
                <span>Clear All</span>
                <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" class="group-hover/clear:rotate-90 transition-transform"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </div>
            <div class="grid grid-cols-1 gap-2.5 max-h-[240px] overflow-y-auto pr-1 scrollbar-hide">
              ${selectedFiles.map((file, index) => `
                <div class="flex items-center justify-between p-3.5 bg-slate-900/40 border border-slate-800/80 rounded-2xl hover:border-emerald-500/40 hover:bg-slate-800/30 transition-all group/file animate-in fade-in slide-in-from-right-3 duration-300">
                  <div class="flex items-center gap-4">
                    <div class="w-11 h-11 flex items-center justify-center rounded-xl bg-slate-950 border border-slate-800 text-emerald-500 shadow-xl group-hover/file:scale-105 transition-transform">
                      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
                    </div>
                    <div class="flex flex-col min-w-0">
                      <span class="text-xs text-slate-100 font-bold truncate max-w-[190px] leading-tight mb-0.5">${file.name}</span>
                      <div class="flex items-center gap-2">
                        <span class="text-[9px] text-slate-500 uppercase font-black tracking-widest">${(file.size / 1024).toFixed(1)} KB</span>
                        <div class="w-1 h-1 rounded-full bg-slate-700"></div>
                        <span class="text-[9px] text-emerald-500/60 uppercase font-black tracking-widest">${file.type.split('/')[1] || 'FILE'}</span>
                      </div>
                    </div>
                  </div>
                  <button class="remove-file-btn w-9 h-9 flex items-center justify-center rounded-xl bg-slate-950 border border-slate-800 text-slate-600 hover:text-rose-500 hover:border-rose-500/40 hover:bg-rose-500/10 transition-all group-hover/file:border-slate-700 shadow-lg" data-index="${index}">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                  </button>
                </div>
              `).join('')}
            </div>
          `;

          fileCount.innerText = `${selectedFiles.length} Artifacts Prepared`;
          
          fileListContainer.querySelectorAll('.remove-file-btn').forEach(btn => {
            (btn as HTMLButtonElement).onclick = (e) => {
              e.stopPropagation();
              const index = parseInt(btn.getAttribute('data-index') || '0');
              selectedFiles.splice(index, 1);
              updateFileListUI();
            };
          });

          const clearAllBtn = fileListContainer.querySelector('#clear-all-files') as HTMLButtonElement;
          if (clearAllBtn) {
            clearAllBtn.onclick = () => {
              selectedFiles = [];
              updateFileListUI();
            };
          }

          (window as any).__swal_files = selectedFiles;
        };

        // Initialize empty state
        updateFileListUI();

        // Toggle logic
        if (toggleBtn) {
          toggleBtn.onclick = () => {
            const isShowingInput = !inputContainer.classList.contains('hidden');
            if (isShowingInput) {
              inputContainer.classList.add('hidden');
              selectContainer.classList.remove('hidden');
              toggleText.innerText = 'Add Custom';
              labelMode.innerText = 'Search Team';
            } else {
              inputContainer.classList.remove('hidden');
              selectContainer.classList.add('hidden');
              toggleText.innerText = 'Back to List';
              labelMode.innerText = 'Manual Entry';
            }
          };
        }

        if (searchInput && dropdownMenu) {
            searchInput.onclick = (e) => {
                e.stopPropagation();
                dropdownMenu.classList.toggle('hidden');
            };

            const hideDropdown = (e: any) => {
                if (!searchInput.contains(e.target as Node) && !dropdownMenu.contains(e.target as Node)) {
                    dropdownMenu.classList.add('hidden');
                }
            };
            document.addEventListener('click', hideDropdown);

            fixerOptions.forEach(opt => {
                opt.onclick = () => {
                   searchInput.value = opt.getAttribute('data-value') || '';
                   dropdownMenu.classList.add('hidden');
                };
            });
        }

        if (fileInput) {
          fileInput.onchange = () => {
            if (fileInput.files) {
              for (let i = 0; i < fileInput.files.length; i++) {
                selectedFiles.push(fileInput.files[i]);
              }
              fileInput.value = '';
              updateFileListUI();
            }
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

  function downloadFile(path: string, filename: string) {
    const token = localStorage.getItem("monitor_token");
    // Use window.open or a temporary link to trigger the download through our new endpoint
    const url = `${API_BASE_URL}/api/v1/download?path=${encodeURIComponent(path)}&token=${token}`;
    
    // We can't easily pass Authorization header via window.open, 
    // so our backend DownloadFile handler should ideally handle token as a query param 
    // if we want to keep it protected, OR we make the static downloads public.
    // For now, let's try a simple link approach.
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }

  async function viewDetails(task: any) {
    if (task.status === 'open' || task.status === 'pending') return;

    let html = '';
    if (task.status === 'closed') {
      let docsHtml = '';
      if (task.documents) {
        try {
          const docs = Array.isArray(task.documents) ? task.documents : JSON.parse(task.documents);
          if (docs && docs.length > 0) {
            docsHtml = `
              <div class="mt-4 space-y-2 animate-in fade-in slide-in-from-bottom-2">
                <p class="text-[10px] font-black text-slate-500 uppercase tracking-widest px-1">ATTACHED EVIDENCE (${docs.length})</p>
                <div class="flex flex-wrap gap-2">
                  ${docs.map((doc: string, idx: number) => `
                    <div class="flex items-center gap-2 p-1.5 bg-slate-900 border border-slate-800 rounded-xl hover:border-emerald-500/50 hover:bg-slate-800/50 transition-all group">
                      <a href="${API_BASE_URL}${doc}" target="_blank" class="flex items-center gap-2 px-1.5 py-0.5">
                        <div class="w-8 h-8 flex items-center justify-center rounded-lg bg-slate-800 border border-slate-700 text-emerald-500 group-hover:bg-emerald-500/10 transition-colors">
                           <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>
                        </div>
                        <div class="flex flex-col min-w-0">
                          <span class="text-[10px] text-slate-300 font-bold truncate max-w-[100px] leading-tight">${doc.split('/').pop()}</span>
                          <span class="text-[8px] text-slate-600 uppercase font-black">View</span>
                        </div>
                      </a>
                      <button class="download-doc-btn w-8 h-8 flex items-center justify-center rounded-lg bg-slate-950 border border-slate-800 text-slate-500 hover:text-emerald-400 hover:bg-emerald-500/10 transition-all" 
                        data-url="${API_BASE_URL}${doc}" 
                        data-name="${doc.split('/').pop()}"
                        title="Download Document">
                        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                      </button>
                    </div>
                  `).join('')}
                </div>
              </div>
            `;
          }
        } catch (e) {
            console.error("Failed to parse documents:", e);
        }
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
          <div class="flex items-center justify-between bg-slate-950/40 border border-slate-800/50 rounded-2xl p-4 gap-4">
            <div class="space-y-1">
              <p class="text-[10px] font-black text-slate-500 uppercase tracking-widest px-1">REPAIR PERFORMED BY</p>
              <p class="text-xs text-emerald-400 font-bold px-1">${task.fixer_name || 'N/A'}</p>
            </div>
            <div class="text-right space-y-1">
              <p class="text-[10px] font-black text-slate-500 uppercase tracking-widest px-1">TASK APPROVED BY</p>
              <p class="text-xs text-blue-400 font-bold px-1">${task.approver?.name || task.approver?.email || 'System'}</p>
            </div>
          </div>
          <div class="space-y-1">
            <p class="text-[10px] font-black text-slate-500 uppercase tracking-widest px-1">RESOLUTION REASON</p>
            <div class="bg-slate-950 border border-slate-800 rounded-2xl p-4 text-sm text-slate-300 leading-relaxed">
              ${task.reason || 'No reason provided.'}
            </div>
          </div>
          ${docsHtml}
          <div class="pt-2 border-t border-slate-800/50 space-y-1">
             <p class="text-[10px] font-black text-slate-600 uppercase tracking-tighter">CLOSED AT ${new Date(task.closed_at).toLocaleString()}</p>
             ${task.approver ? `<p class="text-[10px] font-black text-blue-500/60 uppercase tracking-tighter">APPROVED BY: ${task.approver.name || task.approver.email}</p>` : ''}
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
          ${task.approver ? `
          <div class="pt-2 border-t border-slate-800/50">
             <p class="text-[10px] font-black text-blue-500/60 uppercase tracking-tighter">APPROVED BY: ${task.approver.name || task.approver.email}</p>
          </div>` : ''}
        </div>
      `;
    }

    await systemAlert.fire({
      title: task.status === 'closed' ? 'TASK RESOLUTION DETAIL' : 'TASK FAILURE DETAIL',
      html,
      showConfirmButton: true,
      confirmButtonText: 'CLOSE VIEW',
      didRender: () => {
        const downloadBtns = systemAlert.getPopup()?.querySelectorAll('.download-doc-btn');
        downloadBtns?.forEach(btn => {
          (btn as HTMLButtonElement).onclick = () => {
            const url = btn.getAttribute('data-url') || '';
            const name = btn.getAttribute('data-name') || 'document';
            downloadFile(url, name);
          };
        });
      }
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
               <p class="text-[10px] text-slate-600 uppercase font-bold tracking-tighter mt-1">Reported: {new Date(task.created_at).toLocaleString()}</p>
               {#if task.approver}
                 <p class="text-[10px] text-blue-500/70 uppercase font-bold tracking-tighter mt-0.5 flex items-center gap-1">
                   <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6L9 17l-5-5"/></svg>
                   Approved by: {task.approver.name || task.approver.email}
                 </p>
               {/if}
             </div>
          </div>

          <div class="flex items-center gap-3 shrink-0">
            {#if task.status === 'open'}
              <button 
                on:click|stopPropagation={() => openApprove(task)}
                aria-label="Approve this task"
                class="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-xl text-xs font-bold transition-all shadow-lg shadow-blue-900/20"
              >
                APPROVE TASK
              </button>
            {:else if task.status === 'pending'}
               <button 
                on:click|stopPropagation={() => openFail(task)}
                aria-label="Mark task as failed"
                class="px-4 py-2 border border-rose-500/30 text-rose-400 hover:bg-rose-500/10 rounded-xl text-xs font-bold transition-all"
              >
                FAIL TASK
              </button>
              <button 
                on:click|stopPropagation={() => openClose(task)}
                aria-label="Complete and close task"
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
                    <div class="flex gap-1" aria-label="Attached Documents">
                      {#each (() => {
                        if (!task.documents) return [];
                        if (Array.isArray(task.documents)) return task.documents;
                        try { return JSON.parse(task.documents); } catch { return []; }
                      })() as doc}
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
          aria-label="Previous page"
          class="p-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-400 hover:text-white disabled:opacity-30 transition-all"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        </button>
        
        <div class="flex items-center gap-1">
          {#each Array(totalPages) as _, i}
            <button 
              on:click={() => setPage(i + 1)}
              aria-label="Go to page {i + 1}"
              class="w-10 h-10 rounded-xl font-bold text-xs transition-all border {currentPage === i + 1 ? 'bg-rose-600 border-rose-500 text-white shadow-lg shadow-rose-900/30' : 'bg-slate-900 border-slate-800 text-slate-500 hover:text-slate-300'}"
            >
              {i + 1}
            </button>
          {/each}
        </div>

        <button 
          on:click={nextPage} 
          disabled={currentPage === totalPages}
          aria-label="Next page"
          class="p-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-400 hover:text-white disabled:opacity-30 transition-all"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
        </button>
      </div>
    {/if}
  {/if}
</div>
