<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/state";
  import InputWithVariables from "$lib/components/InputWithVariables.svelte";
  import { API_BASE_URL } from "$lib/config";

  let apis: any[] = [];
  let projects: any[] = [];
  let isLoading = true;

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
  // Silent project filter — read from URL param or localStorage, no UI dropdown
  let selectedProjectId = "";

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

  // ── Run All state ──
  let isRunningAll = false;
  let showRunAllModal = false;
  let runAllResults: { api: any; status: number | null; latency: number | null; passed: boolean; error: string | null }[] = [];
  let runAllProgress = 0;

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
    // Read project_id from URL param or localStorage (set by project page navigation)
    selectedProjectId =
      page.url.searchParams.get("project_id") ||
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


  function replaceVariables(input: string, envVars: any): string {
    if (!input) return "";
    return input.replace(/\{\{([^}]+)\}\}/g, (match, key) => {
      const trimmedKey = key.trim();
      return envVars[trimmedKey] !== undefined ? envVars[trimmedKey] : match;
    });
  }

  // Strip // single-line and /* */ multi-line comments from JSON-like strings
  function stripJSONComments(str: string): string {
    let result = '';
    let inString = false;
    let i = 0;
    while (i < str.length) {
      const ch = str[i];
      // Toggle string context (handle escaped quotes)
      if (ch === '"' && (i === 0 || str[i - 1] !== '\\')) {
        inString = !inString;
        result += ch;
        i++;
        continue;
      }
      if (!inString) {
        // Single-line comment
        if (ch === '/' && str[i + 1] === '/') {
          while (i < str.length && str[i] !== '\n') i++;
          continue;
        }
        // Multi-line comment
        if (ch === '/' && str[i + 1] === '*') {
          i += 2;
          while (i < str.length && !(str[i] === '*' && str[i + 1] === '/')) i++;
          i += 2;
          continue;
        }
      }
      result += ch;
      i++;
    }
    return result;
  }

  // Logs collected from script execution
  let scriptLogs: { type: 'log' | 'error' | 'warn'; msg: string }[] = [];
  let envUpdates: Record<string, string> = {};

  async function runResponseScript(script: string, responseBody: string, status: number) {
    if (!script || !script.trim()) return;

    scriptLogs = [];
    envUpdates = {};

    const currentEnv: Record<string, string> = { ...activeProjectEnvVars };

    const sandboxSetEnv = (key: string, value: string) => {
      currentEnv[key] = value;
      envUpdates[key] = value;
    };

    const sandboxGetEnv = (key: string): string => {
      return currentEnv[key] ?? '';
    };

    const sandboxConsole = {
      log: (...args: any[]) => scriptLogs = [...scriptLogs, { type: 'log', msg: args.map(String).join(' ') }],
      error: (...args: any[]) => scriptLogs = [...scriptLogs, { type: 'error', msg: args.map(String).join(' ') }],
      warn: (...args: any[]) => scriptLogs = [...scriptLogs, { type: 'warn', msg: args.map(String).join(' ') }],
    };

    const responseObj = {
      status,
      body: responseBody,
      headers: {},
    };

    try {
      const fn = new Function('response', 'setEnv', 'getEnv', 'console', `return (async () => { ${script} })()`);
      await fn(responseObj, sandboxSetEnv, sandboxGetEnv, sandboxConsole);
    } catch (e: any) {
      scriptLogs = [...scriptLogs, { type: 'error', msg: '❌ Script error: ' + e.message }];
    }

    // Persist ENV updates to backend if any setEnv() was called
    if (Object.keys(envUpdates).length > 0 && selectedDocApi?.project_id) {
      await persistEnvUpdates(selectedDocApi.project_id, currentEnv);
    }
  }

  async function persistEnvUpdates(projectId: string, newEnv: Record<string, string>) {
    try {
      const token = localStorage.getItem("monitor_token");
      await fetch(`${API_BASE_URL}/api/v1/projects/${projectId}`, {
        method: "PUT",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ environment_variables: JSON.stringify(newEnv) }),
      });
      // Refresh projects so activeProjectEnvVars reactive var reflects new ENV
      await fetchProjects();
    } catch (err) {
      console.error("Failed to persist ENV updates", err);
    }
  }

  async function executeApiTest() {
    if (!selectedDocApi) return;

    isTestingApi = true;
    testResult = null;
    scriptLogs = [];
    envUpdates = {};

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
          body: stripJSONComments(processedBody),
        }),
      });

      const data = await res.json();
      testResult = data;

      // ── Run response_script if present ──
      if (selectedDocApi.response_script && testResult && !testResult.error) {
        const rawBody = testResult.is_json
          ? JSON.stringify(testResult.response)
          : (testResult.response ?? '');
        await runResponseScript(selectedDocApi.response_script, rawBody, testResult.status ?? 200);
      }
    } catch (err: any) {
      testResult = {
        error: err.message || "Failed to connect to monitoring engine proxy",
        is_json: false,
      };
    } finally {
      isTestingApi = false;
    }
  }

  // ── Run All Tests ──
  async function runAllTests() {
    if (!apis.length || isRunningAll) return;
    isRunningAll = true;
    showRunAllModal = true;
    runAllResults = [];
    runAllProgress = 0;

    const token = localStorage.getItem('monitor_token');

    // Helper to get env for a given project_id
    function getEnvForProject(projectId: string): Record<string, string> {
      const proj = projects.find((p: any) => p.id === projectId);
      if (!proj || !proj.environment_variables) return {};
      try { return JSON.parse(proj.environment_variables); } catch { return {}; }
    }

    // Helper to build parsedHeaders
    function buildHeaders(api: any, env: Record<string, string>): Record<string, string> {
      try {
        const raw = JSON.parse(api.headers || '[]');
        const list: any[] = Array.isArray(raw) ? raw : Object.entries(raw).map(([k, v]) => ({ key: k, value: v }));
        const out: Record<string, string> = {};
        list.forEach((h: any) => { if (h.key?.trim()) out[h.key.trim()] = replaceVariables(String(h.value ?? ''), env); });
        return out;
      } catch { return {}; }
    }

    // Run all in parallel
    const promises = apis.map(async (api: any) => {
      const env = getEnvForProject(api.project_id);
      const url = replaceVariables(api.url, env);
      const headers = buildHeaders(api, env);
      const body = stripJSONComments(replaceVariables(api.body || '', env));

      try {
        const res = await fetch(`${API_BASE_URL}/api/v1/apis/test`, {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
          body: JSON.stringify({ method: api.method, url, headers, body }),
        });
        const data = await res.json();
        const expected = api.expected_status_code ?? 200;
        const passed = !data.error && data.status >= 200 && data.status < 400 && data.status === expected;
        runAllResults = [...runAllResults, { api, status: data.status ?? null, latency: data.latency ?? null, passed, error: data.error || null }];
      } catch (e: any) {
        runAllResults = [...runAllResults, { api, status: null, latency: null, passed: false, error: e.message }];
      } finally {
        runAllProgress += 1;
      }
    });

    await Promise.all(promises);

    // Sort: failed first, then by api name
    runAllResults = [...runAllResults].sort((a, b) => {
      if (a.passed !== b.passed) return a.passed ? 1 : -1;
      return (a.api.name || '').localeCompare(b.api.name || '');
    });

    isRunningAll = false;
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
    <!-- Run All button -->
    {#if apis.length > 0}
      <button on:click={runAllTests} disabled={isRunningAll}
        class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-[11px] font-black uppercase tracking-widest transition-all border
          {isRunningAll
            ? 'bg-slate-800 border-slate-700 text-slate-500 cursor-not-allowed'
            : 'bg-emerald-600/20 border-emerald-500/40 text-emerald-400 hover:bg-emerald-600/30 hover:border-emerald-500/60 shadow-[0_0_12px_rgba(52,211,153,0.15)]'}"
        title="ยิง API ทั้งหมดพร้อมกัน">
        {#if isRunningAll}
          <svg class="animate-spin h-3 w-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
          RUNNING {runAllProgress}/{apis.length}
        {:else}
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
          RUN ALL
          <span class="bg-emerald-500/20 text-emerald-300 px-1.5 py-px rounded text-[9px] font-black">{apis.length}</span>
        {/if}
      </button>
    {/if}
  </div>

  <!-- ── MAIN 3-COLUMN AREA ── -->
  <div class="flex flex-1 overflow-hidden">

    <!-- ── LEFT SIDEBAR ── -->
    <div class="w-[360px] shrink-0 border-r border-slate-800 bg-slate-950/40 flex flex-col overflow-hidden">
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
        <div class="w-full px-8 py-8 space-y-8">

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
                {#if Array.isArray(testResult.response)}
                  <!-- Array response: show badge + keys from first element -->
                  <div class="border-t border-slate-800/60 py-3 flex items-center gap-2">
                    <span class="px-2 py-0.5 text-[10px] font-black rounded border bg-blue-950 border-blue-500/40 text-blue-400">ARRAY</span>
                    <span class="text-slate-500 text-xs font-mono">{testResult.response.length} items</span>
                  </div>
                  {#each Object.entries(testResult.response[0] ?? {}) as [key, val]}
                    <div class="border-t border-slate-800/60 py-3">
                      <div class="flex items-center gap-3">
                        <code class="font-black text-slate-300 text-sm">{key}</code>
                        <span class="text-slate-600 text-xs font-mono">{Array.isArray(val) ? 'array' : typeof val}</span>
                      </div>
                    </div>
                  {/each}
                {:else}
                  {#each Object.entries(testResult.response) as [key, val]}
                    <div class="border-t border-slate-800/60 py-3">
                      <div class="flex items-center gap-3">
                        <code class="font-black text-slate-300 text-sm">{key}</code>
                        <span class="text-slate-600 text-xs font-mono">{Array.isArray(val) ? 'array' : typeof val}</span>
                      </div>
                    </div>
                  {/each}
                {/if}
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
        <div class="px-4 py-4 border-b border-slate-800/60 bg-slate-900/40 shrink-0 flex flex-col gap-3 w-full">
          <!-- URL override -->
          <div class="w-full bg-slate-950/50 border border-slate-700/50 rounded-lg overflow-hidden relative shadow-inner">
             <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-slate-500 font-bold text-[10px] tracking-widest z-30">
               URL
             </div>
             <div class="pl-10">
               <InputWithVariables bind:value={reqUrl} variables={activeProjectEnvVars} placeholder="https://api.example.com"/>
             </div>
          </div>
          <!-- Method selector row -->
          <div class="flex items-center gap-3 w-full">
            <select bind:value={reqMethod} class="bg-slate-950/50 border border-slate-700/50 rounded-lg text-xs font-mono font-bold text-slate-300 px-3 py-2.5 focus:outline-none focus:border-cyan-500/40 shadow-inner w-28 cursor-pointer hover:bg-slate-800 transition-colors">
              {#each ['GET','POST','PUT','PATCH','DELETE'] as m}
                <option value={m}>{m}</option>
              {/each}
            </select>
            <button on:click={executeApiTest} disabled={isTestingApi}
              class="flex-1 py-2.5 rounded-lg text-xs font-bold flex items-center justify-center gap-2 transition-all w-full
                {isTestingApi ? 'bg-slate-800 text-slate-500 border border-slate-700 cursor-not-allowed' : 'bg-cyan-600 hover:bg-cyan-500 text-cyan-50 shadow-[0_0_15px_rgba(6,182,212,0.4)] border border-cyan-500/50 tracking-widest uppercase'}">
              {#if isTestingApi}
                <svg class="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
                SENDING...
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                SEND REQUEST
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

            <!-- ENV Update Banner -->
            {#if Object.keys(envUpdates).length > 0}
              <div class="mx-4 mb-3 rounded-lg border border-emerald-500/40 bg-emerald-950/50 px-3 py-2.5">
                <div class="flex items-center gap-2 mb-1.5">
                  <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-emerald-400 shrink-0"><polyline points="20 6 9 17 4 12"/></svg>
                  <span class="text-[10px] font-black text-emerald-400 uppercase tracking-widest">ENV Updated</span>
                </div>
                {#each Object.entries(envUpdates) as [k, v]}
                  <div class="flex items-start gap-2 mt-1">
                    <code class="text-[10px] font-mono text-amber-300 shrink-0">{k}</code>
                    <span class="text-slate-600 text-[10px]">=</span>
                    <code class="text-[10px] font-mono text-slate-400 truncate" title={v}>{v.length > 32 ? v.slice(0, 32) + '…' : v}</code>
                  </div>
                {/each}
              </div>
            {/if}

            <!-- Script Console Logs -->
            {#if scriptLogs.length > 0}
              <div class="mx-4 mb-4 rounded-lg border border-slate-700/60 bg-slate-900/60 overflow-hidden">
                <div class="flex items-center gap-2 px-3 py-1.5 border-b border-slate-700/60 bg-slate-800/40">
                  <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-500"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
                  <span class="text-[9px] font-black uppercase tracking-widest text-slate-500">Script Console</span>
                </div>
                <div class="p-2 space-y-0.5 max-h-32 overflow-y-auto custom-scrollbar">
                  {#each scriptLogs as log}
                    <div class="flex items-start gap-1.5">
                      <span class="text-[9px] font-mono shrink-0 mt-px
                        {log.type === 'error' ? 'text-red-500' : log.type === 'warn' ? 'text-amber-400' : 'text-slate-500'}">
                        {log.type === 'error' ? '✖' : log.type === 'warn' ? '⚠' : '›'}
                      </span>
                      <pre class="text-[10px] font-mono whitespace-pre-wrap break-words leading-relaxed
                        {log.type === 'error' ? 'text-red-400' : log.type === 'warn' ? 'text-amber-300' : 'text-slate-300'}">{log.msg}</pre>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
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

<!-- ── RUN ALL RESULTS MODAL ── -->
{#if showRunAllModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4" style="background:rgba(0,0,0,0.75);backdrop-filter:blur(6px);">
    <div class="bg-slate-900 border border-slate-700/60 rounded-2xl shadow-2xl w-full max-w-2xl max-h-[80vh] flex flex-col overflow-hidden">

      <!-- Modal header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-slate-800">
        <div class="flex items-center gap-3">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-emerald-400"><polygon points="5 3 19 12 5 21 5 3"/></svg>
          <span class="font-black text-slate-100 tracking-tight">Run All Results</span>
          {#if isRunningAll}
            <span class="text-[10px] font-mono text-slate-500 animate-pulse">กำลังยิง {runAllProgress}/{apis.length} APIs...</span>
          {:else}
            <!-- Summary badges -->
            {@const passed = runAllResults.filter(r => r.passed).length}
            {@const failed = runAllResults.filter(r => !r.passed).length}
            <span class="px-2 py-0.5 text-[10px] font-black rounded-full bg-emerald-950 border border-emerald-500/40 text-emerald-400">{passed} PASS</span>
            <span class="px-2 py-0.5 text-[10px] font-black rounded-full bg-red-950 border border-red-500/40 text-red-400">{failed} FAIL</span>
          {/if}
        </div>
        <button on:click={() => { if (!isRunningAll) showRunAllModal = false; }}
          class="text-slate-500 hover:text-slate-300 transition-colors p-1 rounded" title="ปิด">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>

      <!-- Progress bar -->
      {#if isRunningAll}
        <div class="h-0.5 bg-slate-800">
          <div class="h-full bg-gradient-to-r from-cyan-500 to-emerald-500 transition-all duration-300"
            style="width: {apis.length ? (runAllProgress / apis.length) * 100 : 0}%"></div>
        </div>
      {/if}

      <!-- Results list -->
      <div class="flex-1 overflow-y-auto custom-scrollbar">
        {#if runAllResults.length === 0 && isRunningAll}
          <div class="flex items-center justify-center py-12 gap-3 text-slate-600">
            <svg class="animate-spin h-5 w-5 text-cyan-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
            <span class="text-sm font-mono">กำลังยิง APIs...</span>
          </div>
        {:else}
          <!-- Column headers -->
          <div class="grid grid-cols-[1fr_auto_auto_auto] gap-3 px-5 py-2 border-b border-slate-800/60 bg-slate-950/40">
            <span class="text-[9px] font-black uppercase tracking-widest text-slate-600">API</span>
            <span class="text-[9px] font-black uppercase tracking-widest text-slate-600 w-14 text-center">STATUS</span>
            <span class="text-[9px] font-black uppercase tracking-widest text-slate-600 w-16 text-right">LATENCY</span>
            <span class="text-[9px] font-black uppercase tracking-widest text-slate-600 w-12 text-center">RESULT</span>
          </div>
          {#each runAllResults as r}
            <div class="grid grid-cols-[1fr_auto_auto_auto] gap-3 items-center px-5 py-3 border-b border-slate-800/40
              hover:bg-slate-800/30 transition-colors cursor-pointer
              {r.passed ? 'hover:bg-emerald-950/10' : 'hover:bg-red-950/10'}"
              on:click={() => selectDocApi(r.api)} role="button" tabindex="0"
              on:keydown={(e) => e.key === 'Enter' && selectDocApi(r.api)}>

              <!-- API name + folder -->
              <div class="flex items-center gap-2 min-w-0">
                <span class="text-[8px] font-black px-1 py-0.5 rounded border shrink-0 {methodColor(r.api.method)}">{r.api.method}</span>
                <div class="min-w-0">
                  <p class="text-[11px] font-mono text-slate-300 truncate">{r.api.name}</p>
                  {#if r.error}
                    <p class="text-[9px] font-mono text-red-400/70 truncate">{r.error}</p>
                  {:else}
                    <p class="text-[9px] font-mono text-slate-600 truncate">{r.api.folder || 'Uncategorized'}</p>
                  {/if}
                </div>
              </div>

              <!-- Status code -->
              <div class="w-14 flex justify-center">
                {#if r.status}
                  <span class="text-[10px] font-black font-mono px-1.5 py-0.5 rounded border
                    {r.status >= 200 && r.status < 300 ? 'bg-emerald-950 border-emerald-500/40 text-emerald-400'
                    : r.status >= 400 ? 'bg-red-950 border-red-500/40 text-red-400'
                    : 'bg-slate-800 border-slate-600 text-slate-400'}">{r.status}</span>
                {:else}
                  <span class="text-[10px] font-mono text-slate-600">—</span>
                {/if}
              </div>

              <!-- Latency -->
              <div class="w-16 text-right">
                {#if r.latency !== null}
                  <span class="text-[10px] font-mono
                    {r.latency < 500 ? 'text-emerald-400' : r.latency < 2000 ? 'text-amber-400' : 'text-red-400'}">{r.latency}ms</span>
                {:else}
                  <span class="text-[10px] font-mono text-slate-600">—</span>
                {/if}
              </div>

              <!-- Pass/Fail badge -->
              <div class="w-12 flex justify-center">
                {#if r.passed}
                  <span class="flex items-center gap-1 text-[9px] font-black text-emerald-400">
                    <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg>
                    PASS
                  </span>
                {:else}
                  <span class="flex items-center gap-1 text-[9px] font-black text-red-400">
                    <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                    FAIL
                  </span>
                {/if}
              </div>

            </div>
          {/each}
          <!-- Loading rows for pending requests -->
          {#each {length: Math.max(0, apis.length - runAllResults.length)} as _}
            <div class="grid grid-cols-[1fr_auto_auto_auto] gap-3 items-center px-5 py-3 border-b border-slate-800/40">
              <div class="flex items-center gap-2">
                <div class="w-8 h-4 rounded bg-slate-800 animate-pulse"></div>
                <div class="h-3 w-40 rounded bg-slate-800 animate-pulse"></div>
              </div>
              <div class="w-14 flex justify-center"><div class="h-3 w-8 rounded bg-slate-800 animate-pulse"></div></div>
              <div class="w-16"><div class="h-3 w-10 rounded bg-slate-800 animate-pulse ml-auto"></div></div>
              <div class="w-12 flex justify-center"><div class="h-3 w-8 rounded bg-slate-800 animate-pulse"></div></div>
            </div>
          {/each}
        {/if}
      </div>

      <!-- Footer -->
      {#if !isRunningAll && runAllResults.length > 0}
        <div class="flex items-center justify-between px-5 py-3 border-t border-slate-800 bg-slate-950/40">
          {@const passed = runAllResults.filter(r => r.passed).length}
          {@const total = runAllResults.length}
          <span class="text-[11px] font-mono text-slate-500">{passed}/{total} APIs ผ่าน</span>
          <div class="flex gap-2">
            <button on:click={runAllTests}
              class="px-3 py-1.5 rounded-lg text-[10px] font-black uppercase tracking-widest border border-slate-700 text-slate-400 hover:border-slate-600 hover:text-slate-300 transition-colors">
              Re-run All
            </button>
            <button on:click={() => showRunAllModal = false}
              class="px-3 py-1.5 rounded-lg text-[10px] font-black uppercase tracking-widest bg-slate-800 border border-slate-700 text-slate-300 hover:bg-slate-700 transition-colors">
              Close
            </button>
          </div>
        </div>
      {/if}

    </div>
  </div>
{/if}

<style>
  .custom-scrollbar::-webkit-scrollbar { width: 4px; }
  .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
  .custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(71,85,105,0.4); border-radius: 4px; }
  .custom-scrollbar::-webkit-scrollbar-thumb:hover { background: rgba(6,182,212,0.3); }
</style>
