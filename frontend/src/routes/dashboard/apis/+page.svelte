<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import InputWithVariables from "$lib/components/InputWithVariables.svelte";
  import { API_BASE_URL } from "$lib/config";

  let apis: any[] = [];
  let projects: any[] = [];
  let isLoading = true;
  let selectedProjectId = "";

  // Doc layout state
  let selectedDocApi: any = null;
  let expandedFolders: Record<string, boolean> = {};
  let isSidebarOpen = true;

  // Pagination & Search
  let searchQuery = "";
  let currentPage = 1;
  let totalItems = 0;
  let itemsPerPage = 100;
  let searchTimeout: any;
  let isProjectDropdownOpen = false;

  $: totalPages = Math.ceil(totalItems / itemsPerPage);

  // Group APIs by folder
  $: groupedByFolder = (() => {
    const groups: Record<string, any[]> = {};
    apis.forEach(api => {
      const folder = api.folder || 'Uncategorized';
      if (!groups[folder]) { groups[folder] = []; expandedFolders[folder] = true; }
      groups[folder].push(api);
    });
    return groups;
  })();

  function selectDocApi(api: any) {
    selectedDocApi = api;
    reqUrl = api.url;
    reqMethod = api.method;
    try { reqHeaders = api.headers && api.headers !== '{}' ? JSON.stringify(JSON.parse(api.headers), null, 2) : '{\n}'; } catch { reqHeaders = api.headers || '{\n}'; }
    reqBody = api.body || '';
    try { reqParams = api.parameters && api.parameters !== '{}' ? JSON.stringify(JSON.parse(api.parameters), null, 2) : '{\n}'; } catch { reqParams = api.parameters || '{\n}'; }
    testResult = null;
  }

  function generateCurl(api: any, envVars: any): string {
    if (!api) return '';
    const url = replaceVariables(api.url, envVars);
    let cmd = `curl -X ${api.method} '${url}'`;
    try {
      const hdrs = JSON.parse(api.headers || '[]');
      const list = Array.isArray(hdrs) ? hdrs : Object.entries(hdrs).map(([k,v]) => ({key:k,value:v}));
      list.forEach((h: any) => { if (h.key) cmd += ` \\\n  -H '${h.key}: ${replaceVariables(String(h.value||''), envVars)}'`; });
    } catch {}
    if (api.method !== 'GET' && api.body) {
      const b = replaceVariables(api.body, envVars);
      cmd += ` \\\n  -d '${b}'`;
    }
    return cmd;
  }

  function methodColor(m: string) {
    if (m === 'GET') return 'bg-emerald-950 border-emerald-500/40 text-emerald-400';
    if (m === 'POST') return 'bg-blue-950 border-blue-500/40 text-blue-400';
    if (m === 'PUT') return 'bg-amber-950 border-amber-500/40 text-amber-400';
    if (m === 'DELETE') return 'bg-red-950 border-red-500/40 text-red-400';
    return 'bg-slate-800 border-slate-600 text-slate-300';
  }

  // Inline test state
  let isTestingApi = false;
  let testResult: any = null;

  // Editable request fields
  let reqUrl = "";
  let reqMethod = "";
  let reqHeaders = "";
  let reqBody = "";
  let reqParams = "";

  // Custom copy feedback state
  let copyFeedback: Record<string, boolean> = {};

  async function copyToClipboard(text: string, id: string) {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      copyFeedback[id] = true;
      setTimeout(() => {
        copyFeedback[id] = false;
      }, 2000);
    } catch (err) {
      console.error("Failed to copy", err);
    }
  }

  $: activeProjectEnvVars = (() => {
    if (!selectedDocApi || !projects.length) return {};
    const activeProject = projects.find((p) => p.id === selectedDocApi.project_id);
    if (
      activeProject &&
      activeProject.environment_variables &&
      activeProject.environment_variables !== "{}"
    ) {
      try {
        return JSON.parse(activeProject.environment_variables);
      } catch (e) {}
    }
    return {};
  })();

  onMount(async () => {
    selectedProjectId =
      $page.url.searchParams.get("project_id") ||
      localStorage.getItem("monitor_selected_project") ||
      "";
    await fetchProjects();
    await fetchAPIs();
  });

  async function fetchProjects() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        projects = await res.json();
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function fetchAPIs() {
    isLoading = true;
    try {
      const token = localStorage.getItem("monitor_token");
      let url = `${API_BASE_URL}/api/v1/apis?page=${currentPage}&limit=${itemsPerPage}`;
      if (selectedProjectId) {
        url += `&project_id=${selectedProjectId}`;
      }
      if (searchQuery) {
        url += `&search=${encodeURIComponent(searchQuery)}`;
      }

      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const result = await res.json();
        if (result.data) {
          apis = result.data;
          totalItems = result.total;
        } else {
          apis = result;
          totalItems = result.length;
        }
        // Auto-select first API
        if (apis.length > 0 && !selectedDocApi) selectDocApi(apis[0]);
      }
    } catch (err) {
      console.error(err);
    } finally {
      isLoading = false;
    }
  }

  function handleSearchInput() {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      currentPage = 1;
      fetchAPIs();
    }, 500);
  }

  function changePage(page: number) {
    if (page < 1 || page > totalPages) return;
    currentPage = page;
    fetchAPIs();
  }

  function handleFilterChange() {
    currentPage = 1;
    isProjectDropdownOpen = false;
    fetchAPIs();
  }

  function selectProject(id: string) {
    selectedProjectId = id;
    handleFilterChange();
  }

  function replaceVariables(input: string, envVars: any): string {
    if (!input) return "";
    return input.replace(/\{\{([^}]+)\}\}/g, (match, key) => {
      const trimmedKey = key.trim();
      return envVars[trimmedKey] !== undefined ? envVars[trimmedKey] : match;
    });
  }

  async function executeApiTest() {
    if (!selectedDocApi) return;

    isTestingApi = true;
    testResult = null;

    const envVars = activeProjectEnvVars;

    // Apply regex replacement
    const processedUrl = replaceVariables(reqUrl, envVars);
    const processedHeaders = replaceVariables(reqHeaders, envVars);
    const processedBody = replaceVariables(reqBody, envVars);
    const processedParams = replaceVariables(reqParams, envVars);

    // Parse headers if valid JSON
    let parsedHeaders: any = {};
    try {
      if (
        processedHeaders.trim() &&
        processedHeaders.trim() !== "{}" &&
        processedHeaders.trim() !== "{\n}" &&
        processedHeaders.trim() !== "[]"
      ) {
        const rawHeaders = JSON.parse(processedHeaders);
        if (Array.isArray(rawHeaders)) {
          rawHeaders.forEach((item) => {
            if (item.key && item.key.trim())
              parsedHeaders[item.key.trim()] = item.value;
          });
        } else {
          parsedHeaders = rawHeaders;
        }
      }
    } catch (e) {
      testResult = { error: "Invalid JSON format in Headers", is_json: false };
      isTestingApi = false;
      return;
    }

    // Construct final URL with URL-encoded parameters if they exist
    let finalUrl = processedUrl;
    try {
      if (
        processedParams.trim() &&
        processedParams.trim() !== "{}" &&
        processedParams.trim() !== "{\n}" &&
        processedParams.trim() !== "[]"
      ) {
        const parsedParams = JSON.parse(processedParams);
        const urlObj = new URL(finalUrl);
        if (Array.isArray(parsedParams)) {
          parsedParams.forEach((item) => {
            if (item.key && item.key.trim())
              urlObj.searchParams.append(item.key.trim(), item.value);
          });
        } else {
          Object.keys(parsedParams).forEach((key) => {
            urlObj.searchParams.append(key, parsedParams[key]);
          });
        }
        finalUrl = urlObj.toString();
      }
    } catch (e) {
      testResult = {
        error: "Invalid JSON format in Parameters or Invalid Base URL",
        is_json: false,
      };
      isTestingApi = false;
      return;
    }

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/apis/test`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          method: reqMethod,
          url: finalUrl,
          headers: parsedHeaders,
          body: processedBody,
        }),
      });

      const data = await res.json();
      testResult = data;
    } catch (err: any) {
      testResult = {
        error: err.message || "Failed to connect to monitoring engine proxy",
        is_json: false,
      };
    } finally {
      isTestingApi = false;
    }
  }
</script>

<!-- ========== 3-COLUMN API DOC LAYOUT ========== -->
<div class="fade-in flex flex-col" style="height: calc(100vh - 64px); overflow: hidden;">

  <!-- ── TOP BAR ── -->
  <div class="flex items-center gap-3 px-4 py-3 border-b border-slate-800 shrink-0 bg-slate-950/60 backdrop-blur">
    <div class="flex-1 flex items-center gap-2">
      <h1 class="text-lg font-black font-mono text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight uppercase">API_DOCS</h1>
      {#if selectedDocApi}
        <span class="text-slate-600 font-mono text-xs">/</span>
        <span class="text-slate-400 font-mono text-xs truncate max-w-[200px]">{selectedDocApi.name}</span>
      {/if}
    </div>
    <!-- Search -->
    <div class="relative">
      <input type="text" placeholder="ค้นหา API..." bind:value={searchQuery}
        on:input={handleSearchInput}
        class="bg-slate-900 border border-slate-700 rounded-lg px-8 py-1.5 text-xs text-slate-300 font-mono focus:outline-none focus:border-cyan-500/50 w-52 placeholder:text-slate-600"/>
      <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
    </div>
    <!-- Project selector -->
    <div class="relative">
      <button on:click={() => (isProjectDropdownOpen = !isProjectDropdownOpen)}
        class="flex items-center gap-2 bg-slate-900 border border-slate-700 rounded-lg px-3 py-1.5 text-xs text-cyan-400 font-mono hover:border-cyan-500/40 transition-all min-w-[160px] justify-between">
        <span class="truncate">{selectedProjectId ? (projects.find(p => p.id === selectedProjectId)?.name || 'All Projects') : 'All Projects'}</span>
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="{isProjectDropdownOpen ? 'rotate-180' : ''} transition-transform"><path d="m6 9 6 6 6-6"/></svg>
      </button>
      {#if isProjectDropdownOpen}
        <div class="absolute top-full left-0 mt-1 w-full min-w-[200px] bg-slate-900 border border-slate-700 rounded-xl z-50 shadow-2xl overflow-hidden">
          <div class="max-h-[280px] overflow-y-auto p-1">
            <button on:click={() => { selectedProjectId=''; handleFilterChange(); }}
              class="w-full text-left px-3 py-2 rounded-lg text-xs font-mono hover:bg-slate-800 {selectedProjectId==='' ? 'text-cyan-400 font-black' : 'text-slate-400'}">All Projects</button>
            {#each projects as p}
              <button on:click={() => { selectedProjectId = p.id; handleFilterChange(); }}
                class="w-full text-left px-3 py-2 rounded-lg text-xs font-mono hover:bg-slate-800 {selectedProjectId===p.id ? 'text-cyan-400 font-black' : 'text-slate-400'}">{p.name}</button>
            {/each}
          </div>
        </div>
        <div class="fixed inset-0 z-40" role="button" tabindex="-1" aria-label="close" on:click={() => (isProjectDropdownOpen=false)} on:keydown={(e) => e.key==='Escape' && (isProjectDropdownOpen=false)}></div>
      {/if}
    </div>
  </div>

  <!-- ── MAIN 3-COLUMN AREA ── -->
  <div class="flex flex-1 overflow-hidden">

    <!-- ── LEFT SIDEBAR ── -->
    <div class="w-60 shrink-0 border-r border-slate-800 bg-slate-950/40 flex flex-col overflow-hidden">
      <div class="flex-1 overflow-y-auto py-2 custom-scrollbar">
        {#if isLoading}
          <div class="flex justify-center py-8"><svg class="animate-spin h-5 w-5 text-cyan-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg></div>
        {:else}
          {#each Object.entries(groupedByFolder) as [folder, folderApis]}
            <div class="mb-1">
              <!-- Folder header -->
              <button on:click={() => (expandedFolders[folder] = !expandedFolders[folder])}
                class="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-slate-800/50 transition-colors group">
                <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-500 {expandedFolders[folder] ? 'rotate-90' : ''} transition-transform shrink-0"><path d="m9 18 6-6-6-6"/></svg>
                <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-cyan-600 shrink-0"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                <span class="text-[11px] font-bold text-slate-400 uppercase tracking-wide truncate">{folder}</span>
                <span class="ml-auto text-[10px] text-slate-600 shrink-0">{folderApis.length}</span>
              </button>
              <!-- API list -->
              {#if expandedFolders[folder]}
                {#each folderApis as api}
                  <button on:click={() => selectDocApi(api)}
                    class="w-full flex items-center gap-2 px-3 py-2 pl-7 text-left transition-all hover:bg-slate-800/60
                      {selectedDocApi?.id === api.id ? 'bg-slate-800/80 border-r-2 border-cyan-500' : ''}">
                    <span class="text-[9px] font-black px-1.5 py-0.5 rounded border shrink-0 {methodColor(api.method)}">{api.method}</span>
                    <span class="text-[11px] font-mono text-slate-300 truncate {selectedDocApi?.id === api.id ? 'text-cyan-300' : ''}">{api.name}</span>
                  </button>
                {/each}
              {/if}
            </div>
          {/each}
          {#if Object.keys(groupedByFolder).length === 0}
            <p class="text-center text-slate-600 text-xs font-mono py-8">No APIs found</p>
          {/if}
        {/if}
      </div>
    </div>

    <!-- ── CENTER: DOC PANEL ── -->
    <div class="flex-1 overflow-y-auto custom-scrollbar bg-slate-950/20">
      {#if selectedDocApi}
        <div class="max-w-2xl mx-auto px-8 py-8 space-y-8">

          <!-- Title -->
          <div>
            <div class="flex items-center gap-3 mb-3">
              <span class="px-2.5 py-1 rounded border text-xs font-black {methodColor(selectedDocApi.method)}">{selectedDocApi.method}</span>
              <h1 class="text-2xl font-black text-slate-100">{selectedDocApi.name}</h1>
            </div>
            <div class="bg-slate-900 border border-slate-700/60 rounded-lg px-4 py-2.5 font-mono text-xs text-cyan-300/80 break-all">{selectedDocApi.url}</div>
          </div>

          <hr class="border-slate-800"/>

          <!-- Authentication -->
          {#if (() => { try { const h = JSON.parse(selectedDocApi.headers||'[]'); const list = Array.isArray(h)?h:Object.entries(h).map(([k,v])=>({key:k,value:v})); return list.some((x:any)=>x.key?.toLowerCase().includes('auth')||x.key?.toLowerCase().includes('token')); } catch { return false; } })()}
          <div>
            <h2 class="text-base font-black text-slate-200 mb-4">Authentication</h2>
            {#each (() => { try { const h = JSON.parse(selectedDocApi.headers||'[]'); return Array.isArray(h)?h:Object.entries(h).map(([k,v])=>({key:k,value:v})); } catch { return []; } })() as hdr}
              {#if hdr.key?.toLowerCase().includes('auth') || hdr.key?.toLowerCase().includes('token')}
                <div class="border-t border-slate-800 py-4">
                  <div class="flex items-center gap-3 mb-1">
                    <span class="font-black text-slate-200 text-sm">{hdr.key}</span>
                    <span class="text-slate-500 text-xs font-mono">{typeof hdr.value === 'string' && hdr.value.toLowerCase().includes('bearer') ? 'Bearer Token' : 'string'}</span>
                  </div>
                  <code class="text-cyan-400 text-xs font-mono">{hdr.value || ''}</code>
                </div>
              {/if}
            {/each}
          </div>
          <hr class="border-slate-800"/>
          {/if}

          <!-- Request Headers -->
          {#if selectedDocApi.headers && selectedDocApi.headers !== '[]' && selectedDocApi.headers !== '{}'}
          <div>
            <h2 class="text-base font-black text-slate-200 mb-4">Request Headers</h2>
            {#each (() => { try { const h = JSON.parse(selectedDocApi.headers); return Array.isArray(h)?h:Object.entries(h).map(([k,v])=>({key:k,value:v})); } catch { return []; } })() as hdr}
              {#if hdr.key}
              <div class="border-t border-slate-800 py-3 flex items-start gap-4">
                <code class="font-black text-slate-300 text-sm w-48 shrink-0">{hdr.key}</code>
                <span class="text-slate-500 text-xs font-mono mt-0.5">string</span>
                <code class="text-slate-400 text-xs font-mono ml-auto truncate max-w-[200px]" title={String(hdr.value||'')}>{String(hdr.value||'')}</code>
              </div>
              {/if}
            {/each}
          </div>
          <hr class="border-slate-800"/>
          {/if}

          <!-- Request Body -->
          {#if selectedDocApi.method !== 'GET' && selectedDocApi.body && selectedDocApi.body !== '{}'}
          <div>
            <h2 class="text-base font-black text-slate-200 mb-2">Request Body</h2>
            <p class="text-slate-500 text-sm mb-4">application/json</p>
            {#each (() => { try { const b = JSON.parse(selectedDocApi.body); return Object.entries(b).map(([k,v])=>({key:k,value:v})); } catch { return []; } })() as field}
              <div class="border-t border-slate-800 py-3">
                <div class="flex items-center gap-3">
                  <code class="font-black text-slate-200 text-sm">{field.key}</code>
                  <span class="text-slate-500 text-xs font-mono">{typeof field.value}</span>
                </div>
                {#if typeof field.value === 'string'}
                  <code class="text-cyan-400/70 text-xs font-mono mt-1 block">{field.value}</code>
                {/if}
              </div>
            {/each}
          </div>
          <hr class="border-slate-800"/>
          {/if}

          <!-- Response -->
          <div>
            <h2 class="text-base font-black text-slate-200 mb-4">Response</h2>
            <div class="border-t border-slate-800 py-4 flex items-center gap-4">
              <span class="px-2 py-0.5 text-xs font-black rounded border bg-emerald-950 border-emerald-500/40 text-emerald-400">{selectedDocApi.expected_status_code}</span>
              <span class="text-slate-400 text-sm">OK — ระบบตอบสนองสำเร็จ</span>
            </div>
            {#if testResult && !testResult.error}
              {#if testResult.is_json && testResult.response}
                {#each Object.entries(testResult.response) as [key, val]}
                  <div class="border-t border-slate-800/60 py-3">
                    <div class="flex items-center gap-3">
                      <code class="font-black text-slate-300 text-sm">{key}</code>
                      <span class="text-slate-600 text-xs font-mono">{Array.isArray(val) ? 'list of objects' : typeof val}</span>
                    </div>
                  </div>
                {/each}
              {/if}
            {:else}
              <div class="border-t border-slate-800/60 py-3">
                <p class="text-slate-600 text-sm font-mono italic">กด Send Request เพื่อดู Response fields</p>
              </div>
            {/if}
          </div>

        </div>
      {:else}
        <div class="flex items-center justify-center h-full text-slate-700 flex-col gap-3">
          <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" class="opacity-30"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
          <p class="font-mono text-sm">เลือก API จาก sidebar</p>
        </div>
      {/if}
    </div>

    <!-- ── RIGHT: CODE + TEST PANEL ── -->
    <div class="w-96 shrink-0 border-l border-slate-800 bg-slate-950/60 flex flex-col overflow-hidden">
      {#if selectedDocApi}
        <!-- cURL block -->
        <div class="border-b border-slate-800 shrink-0">
          <div class="flex items-center justify-between px-4 py-2 bg-slate-900/60">
            <div class="flex items-center gap-2">
              <span class="px-1.5 py-0.5 text-[10px] font-black rounded border {methodColor(selectedDocApi.method)}">{selectedDocApi.method}</span>
              <span class="text-slate-500 font-mono text-[10px] truncate max-w-[200px]">{selectedDocApi.url.replace(/\{\{[^}]+\}\}/g, '...')}</span>
            </div>
            <button on:click={() => copyToClipboard(generateCurl(selectedDocApi, activeProjectEnvVars), 'curl')}
              class="text-slate-500 hover:text-cyan-400 transition-colors p-1" title="Copy cURL">
              {#if copyFeedback['curl']}
                <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-emerald-400"><polyline points="20 6 9 17 4 12"/></svg>
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
              {/if}
            </button>
          </div>
          <div class="bg-[#0b1120] px-4 py-3 overflow-x-auto" style="max-height:160px">
            <pre class="text-[11px] font-mono text-slate-300 whitespace-pre">{generateCurl(selectedDocApi, activeProjectEnvVars)}</pre>
          </div>
        </div>

        <!-- Test controls -->
        <div class="px-4 py-3 border-b border-slate-800 shrink-0 space-y-2">
          <!-- URL override -->
          <InputWithVariables bind:value={reqUrl} variables={activeProjectEnvVars} placeholder="URL"/>
          <!-- Method selector row -->
          <div class="flex items-center gap-2">
            <select bind:value={reqMethod} class="bg-slate-900 border border-slate-700 rounded text-xs font-mono text-slate-300 px-2 py-1 focus:outline-none focus:border-cyan-500/40">
              {#each ['GET','POST','PUT','PATCH','DELETE'] as m}
                <option value={m}>{m}</option>
              {/each}
            </select>
            <button on:click={executeApiTest} disabled={isTestingApi}
              class="flex-1 py-1.5 rounded-lg text-xs font-bold flex items-center justify-center gap-2 transition-all
                {isTestingApi ? 'bg-slate-700 text-slate-500' : 'bg-cyan-600 hover:bg-cyan-500 text-white shadow-[0_0_12px_rgba(6,182,212,0.3)]'}">
              {#if isTestingApi}
                <svg class="animate-spin h-3 w-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
                Sending...
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                Send Request
              {/if}
            </button>
          </div>
        </div>

        <!-- Response result -->
        <div class="flex-1 overflow-y-auto custom-scrollbar">
          {#if testResult}
            <!-- Status bar -->
            <div class="flex items-center gap-2 px-4 py-2 border-b border-slate-800 shrink-0 bg-slate-900/40">
              {#if testResult.status}
                <span class="text-[10px] font-black font-mono px-2 py-0.5 rounded border
                  {testResult.status>=200&&testResult.status<300 ? 'bg-emerald-950 border-emerald-500/40 text-emerald-400'
                  : testResult.status>=400 ? 'bg-red-950 border-red-500/40 text-red-400'
                  : 'bg-slate-800 border-slate-600 text-slate-400'}">
                  {testResult.status}
                </span>
              {/if}
              {#if testResult.latency}
                <span class="text-[10px] font-mono text-slate-500">{testResult.latency}ms</span>
              {/if}
              {#if testResult.error}
                <span class="text-[10px] font-black text-red-400">FAILED</span>
              {/if}
              <button on:click={() => copyToClipboard(testResult.error || (testResult.is_json ? JSON.stringify(testResult.response,null,2) : testResult.response||''), 'response')}
                class="ml-auto text-slate-500 hover:text-cyan-400 transition-colors p-1">
                {#if copyFeedback['response']}
                  <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-emerald-400"><polyline points="20 6 9 17 4 12"/></svg>
                {:else}
                  <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                {/if}
              </button>
            </div>
            <!-- Body -->
            <div class="p-4">
              {#if testResult.error}
                <pre class="text-red-400 font-mono text-[11px] whitespace-pre-wrap break-words">{testResult.error}</pre>
              {:else if testResult.is_json}
                <pre class="text-emerald-400 font-mono text-[11px] whitespace-pre-wrap break-words leading-relaxed">{JSON.stringify(testResult.response,null,2)}</pre>
              {:else}
                <pre class="text-slate-300 font-mono text-[11px] whitespace-pre-wrap break-words">{testResult.response||'Empty response'}</pre>
              {/if}
            </div>
          {:else}
            <div class="flex flex-col items-center justify-center h-full gap-3 text-slate-700 py-12">
              <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" class="opacity-30"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
              <p class="text-xs font-mono">กด Send Request เพื่อดู Response</p>
            </div>
          {/if}
        </div>
      {:else}
        <div class="flex items-center justify-center h-full text-slate-700 flex-col gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" class="opacity-20"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
          <p class="text-xs font-mono">เลือก API เพื่อทดสอบ</p>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .custom-scrollbar::-webkit-scrollbar { width: 4px; }
  .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
  .custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(71,85,105,0.4); border-radius: 4px; }
  .custom-scrollbar::-webkit-scrollbar-thumb:hover { background: rgba(6,182,212,0.3); }
</style>

              class="w-full bg-slate-900/60 border border-slate-700/50 rounded-2xl px-10 py-2.5 text-xs text-cyan-50 font-mono focus:outline-none focus:border-cyan-500/50 focus:ring-4 focus:ring-cyan-500/10 transition-all placeholder:text-slate-600"
            />
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-500"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
            {#if searchQuery}
              <button 
                on:click={() => { searchQuery = ""; handleSearchInput(); }}
                class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-cyan-400"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
              </button>
            {/if}
          </div>

          <!-- Custom Project Dropdown -->
          <div class="relative">
            <button 
              on:click={() => (isProjectDropdownOpen = !isProjectDropdownOpen)}
              class="flex items-center gap-3 bg-slate-900/60 border border-slate-700/50 rounded-2xl px-5 py-2.5 text-xs text-cyan-400 font-mono hover:border-cyan-500/50 transition-all min-w-[180px] justify-between group shadow-lg shadow-black/20"
            >
              <span class="truncate uppercase">
                {selectedProjectId ? projects.find(p => p.id.toString() === selectedProjectId)?.name || 'ALL_PROJECTS' : 'ALL_PROJECTS'}
              </span>
              <svg 
                xmlns="http://www.w3.org/2000/svg" 
                width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" 
                class="text-slate-500 group-hover:text-cyan-400 transition-transform {isProjectDropdownOpen ? 'rotate-180' : ''}"
              >
                <path d="m6 9 6 6 6-6"/>
              </svg>
            </button>

            {#if isProjectDropdownOpen}
              <div 
                class="absolute top-full left-0 mt-2 w-full min-w-[220px] bg-slate-900/90 backdrop-blur-xl border border-slate-700/50 rounded-2xl overflow-hidden z-[100] shadow-2xl animate-in fade-in zoom-in-95 duration-200"
              >
                <div class="max-h-[300px] overflow-y-auto custom-scrollbar p-1.5">
                  <button 
                    on:click={() => selectProject("")}
                    class="w-full text-left px-4 py-2.5 rounded-xl text-xs font-mono transition-all hover:bg-cyan-500/10 {selectedProjectId === '' ? 'text-cyan-400 bg-cyan-500/5 font-black' : 'text-slate-400'}"
                  >
                    ALL_PROJECTS
                  </button>
                  {#each projects as project}
                    <button 
                      on:click={() => selectProject(project.id.toString())}
                      class="w-full text-left px-4 py-2.5 rounded-xl text-xs font-mono transition-all hover:bg-cyan-500/10 {selectedProjectId === project.id.toString() ? 'text-cyan-400 bg-cyan-500/5 font-black' : 'text-slate-400'}"
                    >
                      {project.name.toUpperCase()}
                    </button>
                  {/each}
                </div>
              </div>

              <!-- Click outside overlay for this specific dropdown -->
              <div 
                class="fixed inset-0 z-[90]" 
                on:click={() => (isProjectDropdownOpen = false)}
                on:keydown={(e) => e.key === 'Escape' && (isProjectDropdownOpen = false)}
                role="button"
                tabindex="-1"
                aria-label="Close dropdown"
              ></div>
            {/if}
          </div>
       </div>
    </div>

  <!-- Content -->
  <div class="mt-8">
  {#if isLoading}
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
  {:else if apis.length === 0}
    <div
      class="bg-slate-800/40 backdrop-blur-xl border border-slate-700/50 text-center rounded-3xl p-16 shadow-[0_8px_30px_rgb(0,0,0,0.5)] relative overflow-hidden group/empty"
    >
      <div
        class="absolute inset-0 bg-cyan-900/5 opacity-0 group-hover/empty:opacity-100 transition-opacity duration-500"
      ></div>
      <div
        class="inline-flex items-center justify-center w-24 h-24 rounded-full bg-slate-900 border border-cyan-500/30 mb-6 shadow-[0_0_15px_rgba(6,182,212,0.2)] relative z-10"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="text-cyan-400 h-10 w-10"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><path d="M12 2v20" /><path
            d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"
          /></svg
        >
      </div>
      <h3
        class="text-2xl font-bold text-cyan-50 mb-3 font-mono tracking-wide relative z-10"
      >
        NO_APIS_FOUND
      </h3>
      <p
        class="text-slate-400/80 max-w-md mx-auto mb-10 font-mono text-sm relative z-10"
      >
        UPLOAD A POSTMAN COLLECTION IN A PROJECT TO POPULATE ENDPOINTS.
      </p>
    </div>
  {:else}
    <div class="grid grid-cols-1 xl:grid-cols-2 gap-4 relative z-10">
      {#each apis as api}
        <div
          class="bg-slate-800/40 backdrop-blur-xl border border-slate-700/50 rounded-2xl p-5 shadow-[0_8px_30px_rgb(0,0,0,0.5)] transition-all duration-500 hover:shadow-[0_0px_30px_rgba(6,182,212,0.15)] hover:border-cyan-500/40 hover:-translate-y-1 flex flex-col group relative overflow-hidden"
        >
          <div
            class="absolute inset-0 bg-gradient-to-br from-cyan-900/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"
          ></div>

          <div
            class="absolute top-0 right-0 h-full w-1 border-r-4 {api.notification_config
              ? 'border-emerald-400/80 shadow-[0_0_10px_rgba(52,211,153,0.5)]'
              : 'border-slate-700/50'}"
          ></div>

          <div class="flex justify-between items-start mb-3 relative z-10">
            <div class="flex items-center gap-3">
              <span
                class="px-2 py-0.5 rounded border text-[10px] font-bold whitespace-nowrap tracking-wider
                {api.method === 'GET'
                  ? 'bg-emerald-950/50 border-emerald-500/40 text-emerald-400'
                  : api.method === 'POST'
                    ? 'bg-blue-950/50 border-blue-500/40 text-blue-400'
                    : api.method === 'PUT'
                      ? 'bg-amber-950/50 border-amber-500/40 text-amber-400'
                      : api.method === 'DELETE'
                        ? 'bg-red-950/50 border-red-500/40 text-red-400'
                        : 'bg-slate-800 border-slate-600 text-slate-300'}"
              >
                {api.method}
              </span>
              <h3
                class="font-bold text-cyan-50 tracking-wide font-mono truncate"
                title={api.name}
              >
                {api.name}
              </h3>
            </div>
          </div>

          <div
            class="bg-slate-900 border border-slate-700/50 rounded-lg p-3 text-xs text-slate-400 font-mono truncate mb-4 select-all shadow-inner relative z-10"
            title={api.url}
          >
            {api.url}
          </div>

          <div
            class="mt-auto flex justify-between items-center border-t border-slate-700/50 pt-4 relative z-10"
          >
            <div class="flex gap-4">
              <div class="flex flex-col">
                <span
                  class="text-[9px] text-slate-500 font-bold uppercase tracking-widest font-mono"
                  >METHOD</span
                >
                <span class="text-sm font-bold text-cyan-400 font-mono"
                  >{api.method}</span
                >
              </div>
              <div class="flex flex-col">
                <span
                  class="text-[9px] text-slate-500 font-bold uppercase tracking-widest font-mono"
                  >EXPECTED</span
                >
                <span class="text-sm font-bold text-cyan-400 font-mono"
                  >{api.expected_status_code}</span
                >
              </div>
            </div>

            <a
              href={`/dashboard/projects/${api.project_id}`}
              class="flex items-center gap-1.5 text-xs font-bold text-cyan-500/80 hover:text-cyan-300 transition-colors tracking-widest font-mono uppercase ml-auto"
            >
              PROJECT
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
                class="group-hover:translate-x-1 transition-transform"
                ><line x1="5" y1="12" x2="19" y2="12"></line><polyline
                  points="12 5 19 12 12 19"
                ></polyline></svg
              >
            </a>

            <button
              on:click={() => openTestModal(api)}
              class="flex items-center gap-1.5 text-xs font-bold text-slate-400 hover:text-amber-400 border border-slate-700 hover:border-amber-500/50 bg-slate-900 hover:bg-amber-950/30 hover:shadow-[0_0_15px_rgba(245,158,11,0.2)] px-3 py-1.5 rounded-lg transition-all ml-4 tracking-wider font-mono uppercase"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="currentColor"
                class="text-amber-500"
                ><polygon points="5 3 19 12 5 21 5 3"></polygon></svg
              >
              TEST_API
            </button>
          </div>
        </div>
      {/each}
    </div>

    <!-- Pagination Controls -->
    {#if totalPages > 1}
      <div class="flex items-center justify-center gap-4 mt-12 pb-8 relative z-10">
        <button 
          on:click={() => changePage(currentPage - 1)}
          disabled={currentPage === 1}
          class="flex items-center gap-2 px-4 py-2 bg-slate-900/60 border border-slate-700/50 rounded-xl text-xs font-bold text-slate-400 hover:text-cyan-400 hover:border-cyan-500/30 transition-all disabled:opacity-30 disabled:cursor-not-allowed group"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="group-hover:-translate-x-1 transition-transform"><path d="m15 18-6-6 6-6"/></svg>
          PREV
        </button>

        <div class="flex items-center gap-2 font-mono text-xs">
          <span class="text-cyan-500 font-black">{currentPage}</span>
          <span class="text-slate-600">/</span>
          <span class="text-slate-400">{totalPages}</span>
        </div>

        <button 
          on:click={() => changePage(currentPage + 1)}
          disabled={currentPage === totalPages}
          class="flex items-center gap-2 px-4 py-2 bg-slate-900/60 border border-slate-700/50 rounded-xl text-xs font-bold text-slate-400 hover:text-cyan-400 hover:border-cyan-500/30 transition-all disabled:opacity-30 disabled:cursor-not-allowed group"
        >
          NEXT
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="group-hover:translate-x-1 transition-transform"><path d="m9 18 6-6-6-6"/></svg>
        </button>
      </div>
    {/if}
  {/if}
  </div>
</div>
              />
            </div>
            <button
            >
              {#if copyFeedback["url"]}
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-green-400"><polyline points="20 6 9 17 4 12"></polyline></svg>
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
              {/if}
            </button>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <!-- Headers Editor -->
          <div class="border border-slate-700/60 rounded-xl overflow-hidden flex flex-col bg-slate-950/30" style="height:180px">
            <div class="bg-slate-800/70 border-b border-slate-700/60 px-3 py-2 flex justify-between items-center shrink-0">
              <span class="text-xs font-bold text-cyan-500/80 uppercase tracking-widest font-mono">Headers (JSON)</span>
              <button on:click={() => copyToClipboard(reqHeaders, "headers")} class="p-1 text-slate-500 hover:text-cyan-400 transition-colors" title="Copy JSON">
                {#if copyFeedback["headers"]}
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-green-400"><polyline points="20 6 9 17 4 12"></polyline></svg>
                {:else}
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                {/if}
              </button>
            </div>
            <TextareaWithVariables
              bind:value={reqHeaders}
              variables={activeProjectEnvVars}
              outerClass="h-full bg-slate-900 border-0"
              innerClass="w-full h-full p-3 resize-none"
              textClass="text-green-400 font-mono text-xs"
            />
          </div>

          <!-- Parameters Editor -->
          <div class="border border-slate-700/60 rounded-xl overflow-hidden flex flex-col bg-slate-950/30" style="height:180px">
            <div class="bg-slate-800/70 border-b border-slate-700/60 px-3 py-2 flex justify-between items-center shrink-0">
              <span class="text-xs font-bold text-amber-400/80 uppercase tracking-widest font-mono">Query Params (JSON)</span>
              <button on:click={() => copyToClipboard(reqParams, "params")} class="p-1 text-slate-500 hover:text-cyan-400 transition-colors" title="Copy JSON">
                {#if copyFeedback["params"]}
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-green-400"><polyline points="20 6 9 17 4 12"></polyline></svg>
                {:else}
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                {/if}
              </button>
            </div>
            <TextareaWithVariables
              bind:value={reqParams}
              variables={activeProjectEnvVars}
              outerClass="h-full bg-slate-900 border-0"
              innerClass="w-full h-full p-3 resize-none"
              textClass="text-amber-400 font-mono text-xs"
            />
          </div>
        </div>

        <!-- Body Editor (non-GET) -->
        {#if reqMethod !== "GET"}
          <div class="border border-slate-700/60 rounded-xl overflow-hidden flex flex-col bg-slate-950/30" style="height:180px">
            <div class="bg-slate-800/70 border-b border-slate-700/60 px-3 py-2 flex items-center justify-between shrink-0">
              <span class="text-xs font-bold text-indigo-400/80 uppercase tracking-widest font-mono">Request Body</span>
              <div class="flex items-center gap-2">
                <span class="text-[10px] bg-slate-700 text-indigo-300 border border-slate-600 px-2 py-0.5 rounded uppercase font-mono font-bold">Raw JSON</span>
                <button on:click={() => copyToClipboard(reqBody, "body")} class="p-1 text-slate-500 hover:text-cyan-400 transition-colors" title="Copy JSON">
                  {#if copyFeedback["body"]}
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-green-400"><polyline points="20 6 9 17 4 12"></polyline></svg>
                  {:else}
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                  {/if}
                </button>
              </div>
            </div>
            <TextareaWithVariables
              bind:value={reqBody}
              variables={activeProjectEnvVars}
              outerClass="h-full bg-slate-900 border-0"
              innerClass="w-full h-full p-3 resize-none"
              textClass="text-blue-300 font-mono text-xs"
            />
          </div>
        {/if}

        <!-- Action buttons (pinned to bottom of left panel) -->
        <div class="flex justify-between items-center pt-2 mt-auto">
          <button
            on:click={() => (showApiTestModal = false)}
            class="px-4 py-2 text-slate-400 bg-slate-800 border border-slate-700 rounded-xl hover:bg-slate-700 hover:text-cyan-400 font-bold transition-colors text-xs"
          >Close</button>
          <button
            on:click={executeApiTest}
            disabled={isTestingApi}
            class="px-5 py-2 bg-cyan-600 text-cyan-50 rounded-xl hover:bg-cyan-700 font-bold transition-all shadow-[0_0_15px_rgba(6,182,212,0.3)] text-xs flex items-center gap-2 outline-none focus:ring-4 focus:ring-cyan-500/30 disabled:opacity-75"
          >
          </button>
        </div>
      </div>

      <!-- ===== RIGHT: Response Panel ===== -->
      <div class="api-test-right">
        {#if testResult}
          <div class="animate-fade-in h-full flex flex-col">
            <!-- Response header bar -->
            <div class="flex items-center justify-between mb-3 shrink-0">
              <h3 class="text-xs font-black text-slate-400 tracking-widest font-mono uppercase">Response</h3>
              <div class="flex gap-2">
                {#if testResult.status}
                  <span class="px-2.5 py-1 rounded text-[10px] font-black tracking-widest font-mono
                   {testResult.status >= 200 && testResult.status < 300
                    ? 'bg-green-950/50 text-green-400 border border-green-500/30'
                    : testResult.status >= 400 && testResult.status < 500
                      ? 'bg-amber-950/50 text-amber-400 border border-amber-500/30'
                      : testResult.status >= 500
                        ? 'bg-red-950/50 text-red-400 border border-red-500/30'
                        : 'bg-slate-800 text-slate-400'}">
                    STATUS: {testResult.status}
                  </span>
                {/if}
                {#if testResult.latency}
                  <span class="px-2.5 py-1 rounded text-[10px] font-black tracking-widest font-mono bg-cyan-950/50 text-cyan-400 border border-cyan-500/30">
                    {testResult.latency} MS
                  </span>
                {/if}
                {#if testResult.error}
                  <span class="px-2.5 py-1 rounded text-[10px] font-black tracking-widest font-mono bg-red-950/50 text-red-400 border border-red-500/30">
                    FAILED
                  </span>
                {/if}
              </div>
            </div>

            <!-- Code block -->
            <div class="bg-[#0f172a] rounded-xl overflow-hidden border border-slate-700 flex flex-col flex-1 min-h-0">
              <!-- Toolbar -->
              <div class="h-8 bg-slate-800/80 backdrop-blur-sm border-b border-slate-700 flex items-center justify-between px-4 shrink-0">
                <div class="flex gap-1.5">
                  <div class="w-2.5 h-2.5 rounded-full bg-red-500/80"></div>
                  <div class="w-2.5 h-2.5 rounded-full bg-amber-500/80"></div>
                  <div class="w-2.5 h-2.5 rounded-full bg-green-500/80"></div>
                </div>
                <button
                  on:click={() => copyToClipboard(
                    testResult.error || (testResult.is_json ? JSON.stringify(testResult.response, null, 2) : testResult.response || ""),
                    "response"
                  )}
                  class="p-1 text-slate-400 hover:text-white transition-colors"
                  title="Copy Output"
                >
                  {#if copyFeedback["response"]}
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-green-400"><polyline points="20 6 9 17 4 12"></polyline></svg>
                  {:else}
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                  {/if}
                </button>
              </div>
              <!-- Scrollable response body -->
              <div class="p-4 overflow-auto flex-1 min-h-0">
                {#if testResult.error}
                  <pre class="text-red-400 font-mono text-xs whitespace-pre-wrap break-words leading-relaxed">{testResult.error}</pre>
                {:else if testResult.is_json}
                  <pre class="text-emerald-400 font-mono text-xs whitespace-pre-wrap break-words leading-relaxed">{JSON.stringify(testResult.response, null, 2)}</pre>
                {:else}
                  <pre class="text-slate-300 font-mono text-xs whitespace-pre-wrap break-words leading-relaxed">{testResult.response || "Empty response"}</pre>
                {/if}
              </div>
            </div>
          </div>

