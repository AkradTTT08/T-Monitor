<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import Modal from "$lib/components/Modal.svelte";
  import InputWithVariables from "$lib/components/InputWithVariables.svelte";
  import TextareaWithVariables from "$lib/components/TextareaWithVariables.svelte";
  import { API_BASE_URL } from "$lib/config";

  let apis: any[] = [];
  let projects: any[] = [];
  let isLoading = true;
  let selectedProjectId = "";

  // Pagination & Search
  let searchQuery = "";
  let currentPage = 1;
  let totalItems = 0;
  let itemsPerPage = 12;
  let searchTimeout: any;
  let isProjectDropdownOpen = false;

  $: totalPages = Math.ceil(totalItems / itemsPerPage);

  // API Test Modal State
  let showApiTestModal = false;
  let selectedApi: any = null;
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
    if (!selectedApi || !projects.length) return {};
    const activeProject = projects.find((p) => p.id === selectedApi.project_id);
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
    // Read project_id from URL query param, or fall back to localStorage
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
        // Handle both structured and legacy array responses
        if (result.data) {
          apis = result.data;
          totalItems = result.total;
        } else {
          apis = result;
          totalItems = result.length;
        }
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

  function openTestModal(api: any) {
    selectedApi = api;
    reqUrl = api.url;
    reqMethod = api.method;

    try {
      reqHeaders =
        api.headers && api.headers !== "{}"
          ? JSON.stringify(JSON.parse(api.headers), null, 2)
          : "{\n}";
    } catch (e) {
      reqHeaders = api.headers || "{\n}";
    }

    reqBody = api.body || "";

    try {
      reqParams =
        api.parameters && api.parameters !== "{}"
          ? JSON.stringify(JSON.parse(api.parameters), null, 2)
          : "{\n}";
    } catch (e) {
      reqParams = api.parameters || "{\n}";
    }

    testResult = null;
    showApiTestModal = true;
  }

  function replaceVariables(input: string, envVars: any): string {
    if (!input) return "";
    return input.replace(/\{\{([^}]+)\}\}/g, (match, key) => {
      const trimmedKey = key.trim();
      return envVars[trimmedKey] !== undefined ? envVars[trimmedKey] : match;
    });
  }

  async function executeApiTest() {
    if (!selectedApi) return;

    isTestingApi = true;
    testResult = null;

    // Get project env vars
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

<div class="fade-in max-w-full overflow-x-hidden">
  <!-- Header -->
    <div class="flex flex-col md:flex-row md:items-center gap-4 w-full">
       <div class="flex-1">
          <h1 class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase">
            OPEN_APIS
          </h1>
          <p class="text-cyan-500/80 mt-1 font-mono text-xs tracking-wide">
            MANAGE AND MONITOR HEALTH ACROSS ALL REGISTERED API ENDPOINTS.
          </p>
       </div>

       <!-- Search & Filter Controls -->
       <div class="flex flex-wrap items-center gap-3">
          <!-- Search Input -->
          <div class="relative min-w-[240px]">
            <input 
              type="text" 
              placeholder="SEARCH_BY_NAME_OR_URL..." 
              bind:value={searchQuery}
              on:input={handleSearchInput}
              aria-label="Search APIs by name or URL"
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

<!-- API Testing Modal -->
<Modal
  bind:open={showApiTestModal}
  title="API Details & Testing"
  size="2xl"
>
  {#if selectedApi}
    <!-- 2-column split layout -->
    <div class="api-test-layout">

      <!-- ===== LEFT: Request Panel ===== -->
      <div class="api-test-left">

        <!-- URL Bar -->
        <div class="bg-slate-900/60 border border-slate-700/60 rounded-xl p-3 flex items-center gap-3 shadow-[inset_0_0_30px_rgba(0,0,0,0.3)]">
          <span
            class="px-3 py-1.5 rounded text-sm font-black whitespace-nowrap
             {selectedApi.method === 'GET'
              ? 'bg-green-950/60 text-green-400 border border-green-500/30'
              : selectedApi.method === 'POST'
                ? 'bg-cyan-950/60 text-cyan-400 border border-cyan-500/30'
                : selectedApi.method === 'PUT'
                  ? 'bg-amber-950/60 text-amber-400 border border-amber-500/30'
                  : selectedApi.method === 'DELETE'
                    ? 'bg-red-950/60 text-red-400 border border-red-500/30'
                    : 'bg-slate-800 text-slate-400 border border-slate-600'}"
          >
            {selectedApi.method}
          </span>
          <div class="flex-1 overflow-hidden flex items-center gap-2 group/copy">
            <div class="flex-1">
              <InputWithVariables
                bind:value={reqUrl}
                variables={activeProjectEnvVars}
                placeholder="https://api.example.com/v1/resource"
              />
            </div>
            <button
              on:click={() => copyToClipboard(reqUrl, "url")}
              class="opacity-0 group-hover/copy:opacity-100 transition-opacity p-2 bg-slate-800 border border-slate-700 rounded-md text-slate-400 hover:text-cyan-400 hover:border-cyan-500/40 cursor-pointer shrink-0"
              title="Copy URL"
            >
              {#if copyFeedback["url"]}
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-green-400"><polyline points="20 6 9 17 4 12"></polyline></svg>
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
              {/if}
            </button>
          </div>
        </div>

        <!-- Headers + Params side by side -->
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
            {#if isTestingApi}
              <svg class="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
              Firing Engine...
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"></path><path d="m12 5 7 7-7 7"></path></svg>
              Send Request
            {/if}
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
        {:else}
          <!-- Empty state -->
          <div class="h-full flex flex-col items-center justify-center text-slate-600 select-none gap-4">
            <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round" class="opacity-30"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            <div class="text-center">
              <p class="text-xs font-mono uppercase tracking-widest text-slate-600">Awaiting Request</p>
              <p class="text-[10px] text-slate-700 mt-1">Hit "Send Request" to see the response here</p>
            </div>
          </div>
        {/if}
      </div>

    </div>
  {/if}
</Modal>

<style>
  /* 2-column split layout for API test modal */
  .api-test-layout {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1.25rem;
    height: calc(85vh - 120px);
    min-height: 400px;
    max-height: 680px;
  }

  .api-test-left {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    overflow-y: auto;
    padding-right: 4px;
    min-width: 0;
  }

  .api-test-right {
    display: flex;
    flex-direction: column;
    border-left: 1px solid rgba(71, 85, 105, 0.4);
    padding-left: 1.25rem;
    min-width: 0;
  }

  .animate-fade-in {
    animation: slideInRight 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }
  @keyframes slideInRight {
    from {
      opacity: 0;
      transform: translateX(8px);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }

  /* thin scrollbar for left panel */
  .api-test-left::-webkit-scrollbar {
    width: 4px;
  }
  .api-test-left::-webkit-scrollbar-track {
    background: transparent;
  }
  .api-test-left::-webkit-scrollbar-thumb {
    background: rgba(71, 85, 105, 0.4);
    border-radius: 4px;
  }
</style>
