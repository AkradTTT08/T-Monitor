<script lang="ts">
  import { onMount } from "svelte";
  import Modal from "$lib/components/Modal.svelte";
  import InputWithVariables from "$lib/components/InputWithVariables.svelte";
  import TextareaWithVariables from "$lib/components/TextareaWithVariables.svelte";

  let apis: any[] = [];
  let projects: any[] = [];
  let isLoading = true;
  let selectedProjectId = "";

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
    await fetchProjects();
    await fetchAPIs();
  });

  async function fetchProjects() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch("http://localhost:5273/api/v1/projects", {
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
      let url = "http://localhost:5273/api/v1/apis";
      if (selectedProjectId) {
        url += `?project_id=${selectedProjectId}`;
      }

      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) apis = await res.json();
    } catch (err) {
      console.error(err);
    } finally {
      isLoading = false;
    }
  }

  function handleFilterChange() {
    fetchAPIs();
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
      const res = await fetch("http://localhost:5273/api/v1/apis/test", {
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
  <div
    class="flex flex-col md:flex-row md:items-end justify-between gap-4 mb-8"
  >
    <div>
      <h1 class="text-3xl font-bold text-slate-900 tracking-tight">
        Open APIs
      </h1>
      <p class="text-slate-500 mt-2">
        Manage and monitor health across all registered API endpoints.
      </p>
    </div>

    <div class="flex items-center gap-3">
      <div class="relative min-w-[200px]">
        <select
          bind:value={selectedProjectId}
          on:change={handleFilterChange}
          class="w-full appearance-none bg-white border border-slate-200 text-slate-700 py-2.5 pl-4 pr-10 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500/50 shadow-sm transition-all text-sm font-medium cursor-pointer"
        >
          <option value="">All Projects</option>
          {#each projects as project}
            <option value={project.id}>{project.name}</option>
          {/each}
        </select>
        <div
          class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-3 text-slate-500"
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
            ><polyline points="6 9 12 15 18 9"></polyline></svg
          >
        </div>
      </div>
    </div>
  </div>

  <!-- Content -->
  {#if isLoading}
    <div class="flex justify-center p-12">
      <svg
        class="animate-spin h-8 w-8 text-blue-600"
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
      class="bg-white border text-center border-slate-200 rounded-2xl p-12 shadow-sm"
    >
      <div
        class="inline-flex items-center justify-center w-20 h-20 rounded-full bg-slate-50 mb-4"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="text-slate-400 h-10 w-10"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><path d="M12 2v20" /><path
            d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"
          /></svg
        >
      </div>
      <h3 class="text-xl font-bold text-slate-800 mb-2">No APIs Found</h3>
      <p class="text-slate-500 max-w-md mx-auto">
        Upload a Postman collection in one of your projects to populate APIs
        here.
      </p>
    </div>
  {:else}
    <div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
      {#each apis as api}
        <div
          class="bg-white border border-slate-200 rounded-xl p-5 shadow-sm hover-lift flex flex-col group transition-all relative overflow-hidden"
        >
          <div
            class="absolute top-0 right-0 h-full w-1 border-r-4 {api.notification_config
              ? 'border-green-400'
              : 'border-slate-200'}"
          ></div>

          <div class="flex justify-between items-start mb-3">
            <div class="flex items-center gap-3">
              <span
                class="px-2.5 py-1 rounded text-xs font-bold whitespace-nowrap
                {api.method === 'GET'
                  ? 'bg-green-100 text-green-700'
                  : api.method === 'POST'
                    ? 'bg-blue-100 text-blue-700'
                    : api.method === 'PUT'
                      ? 'bg-yellow-100 text-yellow-700'
                      : api.method === 'DELETE'
                        ? 'bg-red-100 text-red-700'
                        : 'bg-slate-100 text-slate-700'}"
              >
                {api.method}
              </span>
              <h3 class="font-bold text-slate-800 truncate" title={api.name}>
                {api.name}
              </h3>
            </div>
          </div>

          <div
            class="bg-slate-50 rounded-lg p-3 text-sm text-slate-600 font-mono truncate mb-4 border border-slate-100"
            title={api.url}
          >
            {api.url}
          </div>

          <div
            class="mt-auto flex justify-between items-center border-t border-slate-100 pt-4"
          >
            <div class="flex gap-4">
              <div class="flex flex-col">
                <span
                  class="text-[10px] text-slate-400 font-semibold uppercase tracking-wider"
                  >Method</span
                >
                <span class="text-sm font-medium text-slate-700"
                  >{api.method}</span
                >
              </div>
              <div class="flex flex-col">
                <span
                  class="text-[10px] text-slate-400 font-semibold uppercase tracking-wider"
                  >Expected</span
                >
                <span class="text-sm font-medium text-slate-700"
                  >{api.expected_status_code}</span
                >
              </div>
            </div>

            <a
              href={`/dashboard/projects/${api.project_id}`}
              class="flex items-center gap-1.5 text-sm font-medium text-blue-600 hover:text-blue-700 hover:underline group-hover:underline"
            >
              View Project
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><line x1="5" y1="12" x2="19" y2="12"></line><polyline
                  points="12 5 19 12 12 19"
                ></polyline></svg
              >
            </a>

            <button
              on:click={() => openTestModal(api)}
              class="flex items-center gap-1.5 text-sm font-bold text-slate-600 hover:text-amber-600 border border-slate-200 hover:border-amber-300 bg-white hover:bg-amber-50 px-3 py-1.5 rounded-lg transition-all ml-4"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><polygon points="5 3 19 12 5 21 5 3"></polygon></svg
              >
              Test API
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- API Testing Modal -->
<Modal
  bind:open={showApiTestModal}
  title="API Details & Testing"
  maxWidth="max-w-4xl"
>
  {#if selectedApi}
    <div class="space-y-6">
      <div
        class="bg-slate-50 border border-slate-200 rounded-xl p-4 flex items-center justify-between"
      >
        <div class="flex items-center gap-4 w-full">
          <span
            class="px-3 py-1.5 rounded text-sm font-black whitespace-nowrap
             {selectedApi.method === 'GET'
              ? 'bg-green-100 text-green-700'
              : selectedApi.method === 'POST'
                ? 'bg-blue-100 text-blue-700'
                : selectedApi.method === 'PUT'
                  ? 'bg-yellow-100 text-yellow-700'
                  : selectedApi.method === 'DELETE'
                    ? 'bg-red-100 text-red-700'
                    : 'bg-slate-200 text-slate-700'}"
          >
            {selectedApi.method}
          </span>
          <div
            class="flex-1 overflow-hidden flex items-center gap-2 group/copy"
          >
            <div class="flex-1">
              <InputWithVariables
                bind:value={reqUrl}
                variables={activeProjectEnvVars}
                placeholder="https://api.example.com/v1/resource"
              />
            </div>
            <button
              on:click={() => copyToClipboard(reqUrl, "url")}
              class="opacity-0 group-hover/copy:opacity-100 transition-opacity p-2 bg-white border border-slate-200 shadow-sm rounded-md text-slate-500 hover:text-slate-800 hover:bg-slate-50 cursor-pointer shrink-0"
              title="Copy URL"
            >
              {#if copyFeedback["url"]}
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  class="text-green-600"
                  ><polyline points="20 6 9 17 4 12"></polyline></svg
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
                  ><rect x="9" y="9" width="13" height="13" rx="2" ry="2"
                  ></rect><path
                    d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                  ></path></svg
                >
              {/if}
            </button>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <!-- Headers Editor -->
        <div
          class="border border-slate-200 rounded-xl overflow-hidden flex flex-col h-48"
        >
          <div
            class="bg-slate-50 border-b border-slate-200 px-3 py-2 flex justify-between items-center"
          >
            <span
              class="text-xs font-bold text-slate-600 uppercase tracking-widest"
              >Headers (JSON)</span
            >
            <button
              on:click={() => copyToClipboard(reqHeaders, "headers")}
              class="p-1 text-slate-400 hover:text-slate-700 transition-colors"
              title="Copy JSON"
            >
              {#if copyFeedback["headers"]}
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
                  class="text-green-600"
                  ><polyline points="20 6 9 17 4 12"></polyline></svg
                >
              {:else}
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  ><rect x="9" y="9" width="13" height="13" rx="2" ry="2"
                  ></rect><path
                    d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                  ></path></svg
                >
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
        <div
          class="border border-slate-200 rounded-xl overflow-hidden flex flex-col h-48"
        >
          <div
            class="bg-slate-50 border-b border-slate-200 px-3 py-2 flex justify-between items-center"
          >
            <span
              class="text-xs font-bold text-slate-600 uppercase tracking-widest"
              >Query Params (JSON)</span
            >
            <button
              on:click={() => copyToClipboard(reqParams, "params")}
              class="p-1 text-slate-400 hover:text-slate-700 transition-colors"
              title="Copy JSON"
            >
              {#if copyFeedback["params"]}
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
                  class="text-green-600"
                  ><polyline points="20 6 9 17 4 12"></polyline></svg
                >
              {:else}
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  ><rect x="9" y="9" width="13" height="13" rx="2" ry="2"
                  ></rect><path
                    d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                  ></path></svg
                >
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

      <!-- Body Editor -->
      {#if reqMethod !== "GET"}
        <div
          class="border border-slate-200 rounded-xl overflow-hidden flex flex-col h-48"
        >
          <div
            class="bg-slate-50 border-b border-slate-200 px-3 py-2 flex items-center justify-between"
          >
            <span
              class="text-xs font-bold text-slate-600 uppercase tracking-widest"
              >Request Body</span
            >
            <div class="flex items-center gap-2">
              <span
                class="text-[10px] bg-slate-200 text-slate-600 px-2 py-0.5 rounded uppercase font-bold"
                >Raw JSON</span
              >
              <button
                on:click={() => copyToClipboard(reqBody, "body")}
                class="p-1 text-slate-400 hover:text-slate-700 transition-colors"
                title="Copy JSON"
              >
                {#if copyFeedback["body"]}
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
                    class="text-green-600"
                    ><polyline points="20 6 9 17 4 12"></polyline></svg
                  >
                {:else}
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    ><rect x="9" y="9" width="13" height="13" rx="2" ry="2"
                    ></rect><path
                      d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                    ></path></svg
                  >
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

      <div class="flex justify-between items-center pt-2">
        <button
          on:click={() => (showApiTestModal = false)}
          class="px-5 py-2.5 text-slate-600 bg-white border border-slate-300 rounded-xl hover:bg-slate-50 font-bold transition-colors text-sm"
          >Close</button
        >
        <button
          on:click={executeApiTest}
          disabled={isTestingApi}
          class="px-6 py-2.5 bg-blue-600 text-white rounded-xl hover:bg-blue-700 font-bold transition-all shadow-md text-sm flex items-center gap-2 outline-none focus:ring-4 focus:ring-blue-500/30 disabled:opacity-75 relative overflow-hidden"
        >
          {#if isTestingApi}
            <svg
              class="animate-spin h-4 w-4 text-white"
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
            Firing Engine...
          {:else}
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><path d="M5 12h14"></path><path d="m12 5 7 7-7 7"></path></svg
            >
            Send Request
          {/if}
        </button>
      </div>

      <!-- Test Result Output -->
      {#if testResult}
        <div class="mt-8 border-t border-slate-200 pt-6 animate-fade-in">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-sm font-black text-slate-800 tracking-wide">
              RESPONSE
            </h3>
            <div class="flex gap-3">
              {#if testResult.status}
                <span
                  class="px-2.5 py-1 rounded text-[11px] font-black tracking-widest font-mono
                   {testResult.status >= 200 && testResult.status < 300
                    ? 'bg-green-100 text-green-700 border border-green-200'
                    : testResult.status >= 400 && testResult.status < 500
                      ? 'bg-amber-100 text-amber-700 border border-amber-200'
                      : testResult.status >= 500
                        ? 'bg-red-100 text-red-700 border border-red-200'
                        : 'bg-slate-100 text-slate-700'}"
                >
                  STATUS: {testResult.status}
                </span>
              {/if}
              {#if testResult.latency}
                <span
                  class="px-2.5 py-1 rounded text-[11px] font-black tracking-widest font-mono bg-blue-50 text-blue-700 border border-blue-200"
                >
                  {testResult.latency} MS
                </span>
              {/if}
              {#if testResult.error}
                <span
                  class="px-2.5 py-1 rounded text-[11px] font-black tracking-widest font-mono bg-red-100 text-red-700 border border-red-200"
                >
                  FAILED TO CONNECT
                </span>
              {/if}
            </div>
          </div>

          <div
            class="bg-[#0f172a] rounded-xl overflow-hidden border border-slate-700 relative group"
          >
            <div
              class="absolute top-0 w-full h-8 bg-slate-800/80 backdrop-blur-sm border-b border-slate-700 flex items-center justify-between px-4"
            >
              <div class="flex gap-1.5">
                <div class="w-2.5 h-2.5 rounded-full bg-red-500/80"></div>
                <div class="w-2.5 h-2.5 rounded-full bg-amber-500/80"></div>
                <div class="w-2.5 h-2.5 rounded-full bg-green-500/80"></div>
              </div>
              <button
                on:click={() =>
                  copyToClipboard(
                    testResult.error ||
                      (testResult.is_json
                        ? JSON.stringify(testResult.response, null, 2)
                        : testResult.response || ""),
                    "response",
                  )}
                class="p-1 text-slate-400 hover:text-white transition-colors"
                title="Copy Output"
              >
                {#if copyFeedback["response"]}
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
                    class="text-green-400"
                    ><polyline points="20 6 9 17 4 12"></polyline></svg
                  >
                {:else}
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    ><rect x="9" y="9" width="13" height="13" rx="2" ry="2"
                    ></rect><path
                      d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                    ></path></svg
                  >
                {/if}
              </button>
            </div>
            <div class="p-4 pt-10 max-h-96 overflow-y-auto">
              {#if testResult.error}
                <pre
                  class="text-red-400 font-mono text-sm whitespace-pre-wrap leading-relaxed">{testResult.error}</pre>
              {:else if testResult.is_json}
                <pre
                  class="text-emerald-400 font-mono text-sm whitespace-pre-wrap leading-relaxed">{JSON.stringify(
                    testResult.response,
                    null,
                    2,
                  )}</pre>
              {:else}
                <pre
                  class="text-slate-300 font-mono text-sm whitespace-pre-wrap leading-relaxed">{testResult.response ||
                    "Empty response"}</pre>
              {/if}
            </div>
          </div>
        </div>
      {/if}
    </div>
  {/if}
</Modal>

<style>
  .animate-fade-in {
    animation: slideUpFade 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }
  @keyframes slideUpFade {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
</style>
