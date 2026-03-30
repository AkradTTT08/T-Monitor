<script lang="ts">
  import { page } from "$app/stores";
  import { onMount, tick } from "svelte";
  import Swal from "sweetalert2";
  import { API_BASE_URL } from "$lib/config";
  import Modal from "$lib/components/Modal.svelte";
  import InputWithVariables from "$lib/components/InputWithVariables.svelte";
  import TextareaWithVariables from "$lib/components/TextareaWithVariables.svelte";

  $: projectId = $page.params.id;

  let project: any = null;
  let apis: any[] = [];

  // Derive environment variables for highlighting
  $: envVarDict = (() => {
    if (!project || !project.environment_variables) return {};
    try {
      return JSON.parse(project.environment_variables);
    } catch (e) {
      return {};
    }
  })();
  let isLoading = true;
  let isUploading = false;

  // Bulk Delete State
  let selectedApiIds: number[] = [];
  let showBulkDeleteModal = false;

  // Modals state
  let showAddApiModal = false;
  let showEditApiModal = false;
  let showDeleteApiModal = false;
  let showImportModeModal = false;
  let showAddModeModal = false;
  let showPauseApiModal = false;
  let pauseDurationHours: number = 1;
  let pauseMinutes: number = 60;
  let pauseType: 'duration' | 'indefinite' | 'resume' = 'duration';

  // API Reference for Edit/Delete
  let selectedApi: any = null;

  // Form State for manual ADD/EDIT
  let apiForm = {
    folder: "",
    name: "",
    method: "GET",
    url: "",
    headers: "[]",
    body: "{}",
    parameters: "[]",
    expected_status_code: 200,
    interval: 60,
  };

  // KV Arrays for Form Toggles
  let headerMode: "json" | "kv" = "json";
  let bodyMode: "json" | "kv" = "json";
  let paramMode: "json" | "kv" = "json";

  let headersKV: { key: string; value: string }[] = [{ key: "", value: "" }];
  let bodyKV: { key: string; value: string }[] = [{ key: "", value: "" }];
  let paramsKV: { key: string; value: string }[] = [{ key: "", value: "" }];

  function parseJSONSafe(str: string): any {
    try {
      return JSON.parse(str);
    } catch {
      return null;
    }
  }

  function objectToKVArray(obj: any): { key: string; value: string }[] {
    if (!obj || typeof obj !== "object" || Array.isArray(obj))
      return [{ key: "", value: "" }];
    const entries = Object.entries(obj);
    if (entries.length === 0) return [{ key: "", value: "" }];
    return entries.map(([key, value]) => {
      const valStr =
        typeof value === "object" ? JSON.stringify(value) : String(value);
      return { key, value: valStr };
    });
  }

  function parseToKVArray(str: string): { key: string; value: string }[] {
    const parsed = parseJSONSafe(str);
    if (!parsed) return [{ key: "", value: "" }];

    if (Array.isArray(parsed)) {
      if (
        parsed.length > 0 &&
        typeof parsed[0] === "object" &&
        "key" in parsed[0]
      ) {
        return parsed;
      }
      return [{ key: "", value: "" }];
    }
    return objectToKVArray(parsed);
  }

  function kvArrayToMapString(
    kvArray: { key: string; value: string }[],
  ): string {
    const obj: Record<string, string> = {};
    let hasKeys = false;
    for (const item of kvArray) {
      if (item.key.trim()) {
        obj[item.key.trim()] = item.value;
        hasKeys = true;
      }
    }
    return hasKeys ? JSON.stringify(obj, null, 2) : "{}";
  }

  function kvArrayToArrayString(
    kvArray: { key: string; value: string }[],
  ): string {
    const arr = kvArray.filter((item) => item.key.trim() !== "");
    if (arr.length === 0) return "[]";
    return JSON.stringify(arr);
  }

  function syncKVToJSON() {
    if (headerMode === "kv") apiForm.headers = kvArrayToArrayString(headersKV);
    if (bodyMode === "kv") apiForm.body = kvArrayToMapString(bodyKV);
    if (paramMode === "kv") apiForm.parameters = kvArrayToArrayString(paramsKV);
  }

  function toggleHeaderMode(mode: "json" | "kv") {
    if (headerMode === mode) return;
    if (mode === "json") {
      apiForm.headers = kvArrayToArrayString(headersKV);
    } else {
      headersKV = parseToKVArray(apiForm.headers);
    }
    headerMode = mode;
  }

  function toggleParamMode(mode: "json" | "kv") {
    if (paramMode === mode) return;
    if (mode === "json") {
      apiForm.parameters = kvArrayToArrayString(paramsKV);
    } else {
      paramsKV = parseToKVArray(apiForm.parameters);
    }
    paramMode = mode;
  }

  function toggleBodyMode(mode: "json" | "kv") {
    if (bodyMode === mode) return;
    if (mode === "json") {
      apiForm.body = kvArrayToMapString(bodyKV);
    } else {
      bodyKV = parseToKVArray(apiForm.body);
    }
    bodyMode = mode;
  }

  // Temp state for handling file before asking import mode
  let pendingFile: File | null = null;

  let customFolders: string[] = [];
  let showFolderModal = false;
  let newFolderName = "";

  // Folder Edit/Delete state
  let showEditFolderModal = false;
  let showDeleteFolderModal = false;
  let selectedFolderToEdit = "";
  let selectedFolderToDelete = "";
  let editFolderName = "";

  // Drag & Drop State
  let draggingApiId: number | null = null;
  let dragOverItem: { folder: string; index: number } | null = null;

  // Derived state to group APIs by Folder
  $: groupedApis = (() => {
    const groups: Record<string, any[]> = {};

    // Initialize custom folders empty
    customFolders.forEach((f) => (groups[f] = []));

    apis
      .sort((a, b) => a.order_index - b.order_index)
      .forEach((api) => {
        const folder = api.folder || "Uncategorized";
        if (!groups[folder]) groups[folder] = [];
        groups[folder].push(api);
      });

    // Make sure 'Uncategorized' defaults first or last
    if (!groups["Uncategorized"] && Object.keys(groups).length === 0) {
      groups["Uncategorized"] = [];
    }

    return groups;
  })();

  // Parse cURL Paste
  function handleUrlPaste(e: ClipboardEvent) {
    const pastedText = e.clipboardData?.getData("text");
    if (!pastedText) return;

    // Check if it looks like a curl command
    if (pastedText.trim().startsWith("curl ")) {
      e.preventDefault();
      try {
        parseAndApplyCurl(pastedText);
      } catch (err) {
        console.error("Failed to parse cURL:", err);
        Swal.fire({
          icon: "error",
          title: "Import Failed",
          text: "Could not parse the pasted cURL command.",
          toast: true,
          position: "top-end",
          showConfirmButton: false,
          timer: 3000,
        });
      }
    }
  }

  function parseAndApplyCurl(curlStr: string) {
    let method = "GET";
    let url = "";
    const headers: Record<string, string> = {};
    let bodyText = "";

    // Remove newlines and backslashes for easier parsing, or parse token by token
    const tokens = curlStr.match(/(?:[^\s"']+|"[^"]*"|'[^']*')+/g) || [];

    for (let i = 1; i < tokens.length; i++) {
      let token = tokens[i].replace(/^['"]|['"]$/g, "");

      if (["--request", "-X"].includes(tokens[i])) {
        method = tokens[++i].replace(/^['"]|['"]$/g, "").toUpperCase();
      } else if (["--header", "-H"].includes(tokens[i])) {
        const headerToken = tokens[++i].replace(/^['"]|['"]$/g, "");
        const sepIdx = headerToken.indexOf(":");
        if (sepIdx > -1) {
          headers[headerToken.substring(0, sepIdx).trim()] = headerToken
            .substring(sepIdx + 1)
            .trim();
        }
      } else if (
        ["--data", "--data-raw", "--data-binary", "-d"].includes(tokens[i])
      ) {
        let rawBody = tokens[++i];
        if (rawBody) {
          // Remove wrapping quotes and optional bash string prefix `$'...'`
          rawBody = rawBody.replace(/^(\$)?['"]|['"]$/g, "");
          // Unescape any escaped characters (like \") if present
          rawBody = rawBody
            .replace(/\\"/g, '"')
            .replace(/\\'/g, "'")
            .replace(/\\\\/g, "\\");
          bodyText = rawBody;
        }
        if (method === "GET") method = "POST"; // curl defaults to POST if --data is used without -X
      } else if (
        token.startsWith("http://") ||
        token.startsWith("https://") ||
        token.startsWith("{{")
      ) {
        url = token;
      }
    }

    if (url) {
      apiForm.url = url;
      apiForm.method = method;

      const headerKeys = Object.keys(headers);
      if (headerKeys.length > 0) {
        headerMode = "json";
        apiForm.headers = JSON.stringify(headers, null, 2);
        headersKV = parseToKVArray(apiForm.headers);
      }

      if (bodyText) {
        bodyMode = "json";
        try {
          apiForm.body = JSON.stringify(JSON.parse(bodyText), null, 2);
        } catch {
          apiForm.body = bodyText;
        }
        bodyKV = parseToKVArray(apiForm.body);
      }

      Swal.fire({
        icon: "success",
        title: "cURL Imported",
        text: "Successfully parsed cURL command.",
        toast: true,
        position: "top-end",
        showConfirmButton: false,
        timer: 3000,
      });
    }
  }

  function handleAddFolder() {
    if (newFolderName.trim() && !customFolders.includes(newFolderName.trim())) {
      customFolders = [...customFolders, newFolderName.trim()];
    }
    showFolderModal = false;
    newFolderName = "";
  }

  function openEditFolder(folder: string) {
    selectedFolderToEdit = folder;
    editFolderName = folder;
    showEditFolderModal = true;
  }

  function openDeleteFolder(folder: string) {
    selectedFolderToDelete = folder;
    showDeleteFolderModal = true;
  }

  async function handleEditFolderSubmit() {
    const newName = editFolderName.trim();
    if (
      !newName ||
      newName === selectedFolderToEdit ||
      newName === "Uncategorized"
    )
      return;

    if (customFolders.includes(selectedFolderToEdit)) {
      customFolders = customFolders.map((f) =>
        f === selectedFolderToEdit ? newName : f,
      );
    } else if (!customFolders.includes(newName)) {
      customFolders = [...customFolders, newName];
    }

    let updatedApis = apis.filter((a) => a.folder === selectedFolderToEdit);
    updatedApis.forEach((a) => (a.folder = newName));
    apis = [...apis];

    showEditFolderModal = false;
    if (updatedApis.length > 0) {
      await saveReorder(updatedApis);
    }
  }

  async function handleDeleteFolderSubmit() {
    customFolders = customFolders.filter((f) => f !== selectedFolderToDelete);

    let updatedApis = apis.filter((a) => a.folder === selectedFolderToDelete);
    updatedApis.forEach((a) => (a.folder = "Uncategorized"));

    apis = [...apis];
    let uncategorizedApis = apis
      .filter((a) => a.folder === "Uncategorized")
      .sort((a, b) => a.order_index - b.order_index);
    uncategorizedApis.forEach((fa, idx) => (fa.order_index = idx));
    apis = [...apis];

    showDeleteFolderModal = false;
    if (uncategorizedApis.length > 0) {
      await saveReorder(uncategorizedApis);
    }
  }

  // --- Drag & Drop Reordering Logic --- //
  function handleDragStart(e: DragEvent, apiId: number) {
    draggingApiId = apiId;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = "move";
      e.dataTransfer.setData("text/plain", apiId.toString());
    }
  }

  function handleDragOver(e: DragEvent, folder: string, index: number = -1) {
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = "move";
    dragOverItem = { folder, index };
  }

  function handleDragLeave() {
    dragOverItem = null;
  }

  async function handleDrop(
    e: DragEvent,
    targetFolder: string,
    targetIndex: number,
  ) {
    e.preventDefault();
    dragOverItem = null;
    if (!draggingApiId) return;

    const sourceIndex = apis.findIndex((a) => a.id === draggingApiId);
    if (sourceIndex === -1) return;

    const api = apis[sourceIndex];
    let newApis = [...apis];

    newApis.splice(sourceIndex, 1);
    api.folder = targetFolder;

    const folderApis = newApis
      .filter((a) => a.folder === targetFolder)
      .sort((a, b) => a.order_index - b.order_index);

    if (targetIndex === -1) {
      folderApis.push(api);
    } else {
      folderApis.splice(targetIndex, 0, api);
    }

    folderApis.forEach((fa, idx) => (fa.order_index = idx));
    newApis = newApis
      .filter((a) => a.folder !== targetFolder)
      .concat(folderApis);

    apis = newApis;
    draggingApiId = null;

    await saveReorder(folderApis);
  }

  async function saveReorder(reorderedConfigs: any[]) {
    try {
      const token = localStorage.getItem("monitor_token");
      const payload = reorderedConfigs.map((a) => ({
        id: a.id,
        folder: a.folder,
        order_index: a.order_index,
      }));

      const res = await fetch(
        `${API_BASE_URL}/api/v1/apis/reorder/${projectId}`,
        {
          method: "PUT",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
        },
      );
    } catch (err) {
      console.error("Failed to save reorder", err);
    }
  }

  // Re-fetch whenever projectId changes
  $: if (projectId) {
    fetchProjectDetails(projectId);
  }

  async function fetchProjectDetails(id?: string) {
    const targetId = id || projectId;
    if (!targetId) return;

    isLoading = true;
    try {
      const token = localStorage.getItem("monitor_token");

      const [projRes, apisRes] = await Promise.all([
        fetch(`${API_BASE_URL}/api/v1/projects/${targetId}`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
        fetch(`${API_BASE_URL}/api/v1/apis?project_id=${targetId}`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
      ]);

      if (projRes.ok) {
        project = await projRes.json();
      } else {
        localStorage.removeItem("monitor_selected_project");
        window.location.href = "/dashboard";
        return;
      }
      if (apisRes.ok) apis = await apisRes.json();
    } catch (err) {
      console.error(err);
      // window.location.href = "/dashboard"; // Removed to prevent accidental redirects during dev or transient errors
    } finally {
      isLoading = false;
    }
  }

  // --- Bulk Selection Logic --- //
  $: allApisList = apis; // Flatted list of all APIs for the project
  $: allSelected = apis.length > 0 && selectedApiIds.length === apis.length;
  $: indeterminate =
    selectedApiIds.length > 0 && selectedApiIds.length < apis.length;

  function toggleAllSelection() {
    if (allSelected) {
      selectedApiIds = [];
    } else {
      selectedApiIds = apis.map((a) => a.id);
    }
  }

  function toggleSelection(id: number) {
    if (selectedApiIds.includes(id)) {
      selectedApiIds = selectedApiIds.filter((i) => i !== id);
    } else {
      selectedApiIds = [...selectedApiIds, id];
    }
  }

  // --- Postman Import Handlers --- //
  function handleFileSelect(event: Event) {
    const target = event.target as HTMLInputElement;
    if (!target.files || target.files.length === 0) return;

    pendingFile = target.files[0];
    target.value = ""; // Reset input so same file can be selected again

    if (apis.length > 0) {
      showImportModeModal = true;
    } else {
      executePostmanUpload("append"); // default if empty
    }
  }

  async function executePostmanUpload(mode: "append" | "replace") {
    if (!pendingFile) return;
    showImportModeModal = false;
    isUploading = true;

    try {
      const token = localStorage.getItem("monitor_token");
      const formData = new FormData();
      formData.append("collection", pendingFile);

      const res = await fetch(
        `${API_BASE_URL}/api/v1/apis/import-postman?project_id=${projectId}&mode=${mode}`,
        {
          method: "POST",
          headers: { Authorization: `Bearer ${token}` },
          body: formData,
        },
      );

      if (res.ok) {
        await fetchProjectDetails();
      }
    } catch (err) {
      console.error(err);
    } finally {
      isUploading = false;
      pendingFile = null;
    }
  }

  // --- Manual Add API Handlers --- //
  function openAddApiModal() {
    apiForm = {
      folder: "",
      name: "",
      method: "GET",
      url: "",
      headers: "[]",
      body: "{}",
      parameters: "[]",
      expected_status_code: 200,
      interval: 60,
    };
    headerMode = "json";
    bodyMode = "json";
    paramMode = "json";
    headersKV = [{ key: "", value: "" }];
    bodyKV = [{ key: "", value: "" }];
    paramsKV = [{ key: "", value: "" }];
    showAddApiModal = true;
  }

  function openEditApiModal(api: any) {
    selectedApi = api;
    apiForm = {
      folder: api.folder || "",
      name: api.name,
      method: api.method,
      url: api.url,
      headers: api.headers || "[]",
      body: api.body || "{}",
      parameters: api.parameters || "[]",
      expected_status_code: api.expected_status_code || 200,
      interval: api.interval || 60,
    };

    // Parse into KV states
    headersKV = parseToKVArray(apiForm.headers);
    bodyKV = parseToKVArray(apiForm.body);
    paramsKV = parseToKVArray(apiForm.parameters);

    headerMode = "json";
    bodyMode = "json";
    paramMode = "json";
    showEditApiModal = true;
  }

  function openDeleteApiModal(api: any) {
    selectedApi = api;
    showDeleteApiModal = true;
  }

  // --- Schedule Config Modal Handlers --- //
  let showScheduleModal = false;
  let scheduleConfig: any = {
    mode: "Minute timer",
    value: 1,
    day: "Every day",
    time: "4:00 PM",
  };

  const scheduleModes = ["Minute timer", "Hour timer", "Week timer"];
  const weekDays = [
    "Every day",
    "Every weekday (Monday-Friday)",
    "Every Monday",
    "Every Tuesday",
    "Every Wednesday",
    "Every Thursday",
    "Every Friday",
    "Every Saturday",
    "Every Sunday",
  ];
  const timeOptions = Array.from({ length: 24 }, (_, i) => {
    const hour = i % 12 || 12;
    const ampm = i < 12 ? "AM" : "PM";
    return `${hour}:00 ${ampm}`;
  });

  function openScheduleModal(api: any) {
    selectedApi = api;
    if (api.schedule_config) {
      try {
        scheduleConfig = JSON.parse(api.schedule_config);
      } catch (e) {
        console.error(e);
      }
    } else {
      scheduleConfig = {
        mode: "Minute timer",
        value: Math.max(1, Math.floor((api.interval || 60) / 60)),
        day: "Every day",
        time: "4:00 PM",
      };
    }
    showScheduleModal = true;
  }

  async function handleScheduleSubmit() {
    showScheduleModal = false;
    let intervalSeconds = 60;
    if (scheduleConfig.mode === "Minute timer") {
      intervalSeconds = scheduleConfig.value * 60;
    } else if (scheduleConfig.mode === "Hour timer") {
      intervalSeconds = scheduleConfig.value * 3600;
    } else {
      intervalSeconds = 86400; // fallback daily
    }

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/apis/${selectedApi.id}`,
        {
          method: "PUT",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            project_id: parseInt(projectId),
            folder: selectedApi.folder,
            name: selectedApi.name,
            method: selectedApi.method,
            url: selectedApi.url,
            headers: selectedApi.headers || "[]",
            body: selectedApi.body || "{}",
            parameters: selectedApi.parameters || "[]",
            expected_status_code: selectedApi.expected_status_code,
            interval: intervalSeconds,
            schedule_config: JSON.stringify(scheduleConfig),
          }),
        },
      );
      if (res.ok) await fetchProjectDetails();
    } catch (err) {
      console.error(err);
    }
  }

  function handleAddApiSubmit() {
    if (apis.length > 0) {
      showAddApiModal = false;
      showAddModeModal = true;
    } else {
      executeAddApi("append"); // default if empty
    }
  }

  async function handleEditApiSubmit() {
    syncKVToJSON();
    showEditApiModal = false;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/apis/${selectedApi.id}`,
        {
          method: "PUT",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            project_id: parseInt(projectId),
            folder: apiForm.folder,
            name: apiForm.name,
            method: apiForm.method,
            url: apiForm.url,
            parameters: apiForm.parameters,
            headers: apiForm.headers,
            body: apiForm.body,
            expected_status_code: apiForm.expected_status_code,
            interval: apiForm.interval,
          }),
        },
      );

      if (res.ok) {
        await fetchProjectDetails();
        Swal.fire({
          icon: "success",
          title: "Saved",
          text: "API endpoint updated successfully!",
          toast: true,
          position: "top-end",
          showConfirmButton: false,
          timer: 3000,
        });
      } else {
        throw new Error("Failed to update API");
      }
    } catch (err) {
      console.error(err);
      Swal.fire({
        icon: "error",
        title: "Error",
        text: "Failed to save changes.",
        toast: true,
        position: "top-end",
        showConfirmButton: false,
        timer: 3000,
      });
    }
  }

  async function handleDeleteApiSubmit() {
    showDeleteApiModal = false;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/apis/${selectedApi.id}`,
        {
          method: "DELETE",
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      );

      if (res.ok) {
        await fetchProjectDetails();
      }
    } catch (err) {
      console.error(err);
    }
  }

  function openPauseApiModal(api: any) {
    selectedApi = api;
    pauseMinutes = 60;
    pauseType = 'duration';
    showPauseApiModal = true;
  }

  async function handlePauseApiSubmit() {
    showPauseApiModal = false;
    let pause_hours = 0;
    let label = '';
    if (pauseType === 'indefinite') {
      pause_hours = -1;
      label = 'Monitor paused indefinitely.';
    } else if (pauseType === 'duration') {
      pause_hours = pauseMinutes / 60;
      label = `Monitor paused for ${pauseMinutes} minute(s).`;
    } else {
      pause_hours = 0;
      label = 'Monitor resumed.';
    }
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/apis/${selectedApi.id}/pause`,
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ pause_hours }),
        },
      );

      if (res.ok) {
        await fetchProjectDetails();
        Swal.fire({
          icon: "success",
          title: pause_hours !== 0 ? "Monitor Paused" : "Monitor Resumed",
          text: label,
          toast: true,
          position: "top-end",
          showConfirmButton: false,
          timer: 3000,
        });
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function handleBulkDeleteSubmit() {
    try {
      const token = localStorage.getItem("monitor_token");
      // Fire requests concurrently using Promise.all
      const deletePromises = selectedApiIds.map((id) =>
        fetch(`${API_BASE_URL}/api/v1/apis/${id}`, {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
        }),
      );

      const results = await Promise.all(deletePromises);
      const allOk = results.every((res) => res.ok);

      if (allOk || results.some((res) => res.ok)) {
        showBulkDeleteModal = false;
        selectedApiIds = [];
        await fetchProjectDetails();
      } else {
        console.error("Some or all deletions failed");
      }
    } catch (err) {
      console.error("Bulk delete failed:", err);
    }
  }

  async function executeAddApi(mode: "append" | "replace") {
    syncKVToJSON();
    showAddModeModal = false;
    showAddApiModal = false;

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/apis?mode=${mode}`,
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            project_id: parseInt(projectId),
            folder: apiForm.folder,
            name: apiForm.name,
            method: apiForm.method,
            url: apiForm.url,
            parameters: apiForm.parameters,
            headers: apiForm.headers,
            body: apiForm.body,
            expected_status_code: apiForm.expected_status_code,
            interval: apiForm.interval,
          }),
        },
      );

      if (res.ok) {
        await fetchProjectDetails();
      }
    } catch (err) {
      console.error(err);
    }
  }

  // --- Env Vars Modal State --- //
  let showEnvVarsModal = false;
  let envVarsKV: { key: string; value: string }[] = [{ key: "", value: "" }];
  let isSavingEnvVars = false;

  function openEnvVarsModal() {
    let parsed: any = {};
    if (
      project &&
      project.environment_variables &&
      project.environment_variables !== "{}"
    ) {
      try {
        parsed = JSON.parse(project.environment_variables);
      } catch (e) {}
    }

    const entries = Object.entries(parsed);
    if (entries.length > 0) {
      envVarsKV = entries.map(([key, value]) => ({
        key,
        value: String(value),
      }));
    } else {
      envVarsKV = [{ key: "", value: "" }];
    }
    showEnvVarsModal = true;
  }

  function addEnvVarRow() {
    envVarsKV = [...envVarsKV, { key: "", value: "" }];
  }

  function removeEnvVarRow(index: number) {
    envVarsKV = envVarsKV.filter((_, i) => i !== index);
    if (envVarsKV.length === 0) {
      envVarsKV = [{ key: "", value: "" }];
    }
  }

  async function saveEnvVars() {
    isSavingEnvVars = true;

    const envObj: any = {};
    envVarsKV.forEach((item) => {
      const k = item.key.trim();
      const v = item.value.trim();
      if (k) envObj[k] = v;
    });

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/projects/${projectId}`,
        {
          method: "PUT",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            name: project.name,
            description: project.description,
            environment_variables: JSON.stringify(envObj),
          }),
        },
      );

      if (res.ok) {
        showEnvVarsModal = false;
        await fetchProjectDetails();
      }
    } catch (err) {
      console.error(err);
    } finally {
      isSavingEnvVars = false;
    }
  }
</script>

<div class="fade-in max-w-full overflow-x-hidden">
  {#if isLoading}
    <div class="flex justify-center p-12">
      <svg
        class="animate-spin h-8 w-8 text-cyan-400"
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
  {:else if project}
    <!-- Header -->
    <div
      class="bg-slate-800/40 backdrop-blur-xl p-6 md:p-8 rounded-3xl border border-slate-700/50 shadow-[0_8px_30px_rgb(0,0,0,0.5)] mb-8 relative overflow-hidden break-words group/header"
    >
      <!-- Decor -->
      <div
        class="absolute top-0 right-0 w-64 h-64 bg-gradient-to-br from-cyan-900/20 to-transparent rounded-bl-[100px] -z-10 group-hover/header:opacity-70 transition-opacity duration-500"
      ></div>

      <div
        class="flex flex-col lg:flex-row lg:items-center justify-between gap-6 relative z-10"
      >
        <div class="min-w-0">
          <div class="flex items-center gap-3 mb-2">
            <a
              href="/dashboard"
              class="text-cyan-500/80 hover:text-cyan-400 transition-colors shrink-0"
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
                ><line x1="19" y1="12" x2="5" y2="12"></line><polyline
                  points="12 19 5 12 12 5"
                ></polyline></svg
              >
            </a>
            <h1
              class="text-2xl md:text-3xl font-bold text-cyan-50 tracking-tight truncate font-mono"
            >
              {project.name}
            </h1>
          </div>
          <p
            class="text-cyan-500/80 max-w-2xl text-sm md:text-base font-mono tracking-wide"
          >
            {project.description}
          </p>
        </div>

        <div class="flex items-center gap-3 shrink-0 flex-wrap lg:flex-nowrap">
          <button
            on:click={openEnvVarsModal}
            class="bg-emerald-950/30 border border-emerald-500/40 text-emerald-400 hover:bg-emerald-900/50 hover:border-emerald-400 font-bold py-2 px-4 rounded-lg shadow-[0_0_10px_rgba(16,185,129,0.15)] transition-all flex items-center gap-2 text-sm w-full md:w-auto justify-center font-mono tracking-wide"
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
              ><path d="M4 22h14a2 2 0 0 0 2-2V7l-5-5H6a2 2 0 0 0-2-2v4"
              ></path><path d="M14 2v4a2 2 0 0 0 2 2h4"></path><path
                d="M10.4 12.6a2 2 0 1 1 3 3L8 21l-4 1 1.5-4.5L10.4 12.6z"
              ></path></svg
            >
            ENV_VARS {"{x}"}
          </button>
          <a
            href="/dashboard/projects/{projectId}/notifications"
            class="bg-amber-950/30 border border-amber-500/40 text-amber-400 hover:bg-amber-900/50 hover:border-amber-400 font-bold py-2 px-4 rounded-lg shadow-[0_0_10px_rgba(245,158,11,0.15)] transition-all flex items-center gap-2 text-sm w-full md:w-auto justify-center font-mono tracking-wide"
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
              ><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline
                points="22 4 12 14.01 9 11.01"
              ></polyline></svg
            >
            CHANNELS
          </a>
          <button
            on:click={() => (showFolderModal = true)}
            class="bg-indigo-950/30 border border-indigo-500/40 text-indigo-400 hover:bg-indigo-900/50 hover:border-indigo-400 font-bold py-2 px-4 rounded-lg shadow-[0_0_10px_rgba(99,102,241,0.15)] transition-all flex items-center gap-2 text-sm w-full md:w-auto justify-center font-mono tracking-wide"
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
                d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
              ></path><line x1="12" y1="11" x2="12" y2="17"></line><line
                x1="9"
                y1="14"
                x2="15"
                y2="14"
              ></line></svg
            >
            +FOLDER
          </button>
          <button
            on:click={openAddApiModal}
            class="bg-cyan-950 border border-cyan-500/50 text-cyan-400 hover:bg-cyan-900 hover:border-cyan-400 hover:text-cyan-300 font-bold py-2 px-4 rounded-lg shadow-[0_0_15px_rgba(6,182,212,0.3)] transition-all flex items-center gap-2 text-sm w-full md:w-auto justify-center font-mono tracking-wide relative overflow-hidden group"
          >
            <div
              class="absolute inset-0 w-full h-full bg-cyan-400/10 -translate-x-full group-hover:animate-[shimmer_1.5s_infinite] skew-x-12"
            ></div>
            <span class="relative z-10 flex items-center gap-2">
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
                ><line x1="12" y1="5" x2="12" y2="19"></line><line
                  x1="5"
                  y1="12"
                  x2="19"
                  y2="12"
                ></line></svg
              >
              ADD_API
            </span>
          </button>
          <label
            class="cursor-pointer bg-slate-900 border border-slate-700 text-slate-500 hover:bg-slate-800 hover:text-cyan-400 hover:border-cyan-500/50 font-bold py-2 px-4 rounded-lg shadow-sm transition-all flex items-center gap-2 text-sm w-full md:w-auto justify-center font-mono tracking-wide"
          >
            {#if isUploading}
              <svg
                class="animate-spin h-4 w-4 text-cyan-400"
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
              Importing...
            {:else}
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"
                ></path><polyline points="17 8 12 3 7 8"></polyline><line
                  x1="12"
                  y1="3"
                  x2="12"
                  y2="15"
                ></line></svg
              >
              Upload Collection
            {/if}
            <input
              type="file"
              accept=".json"
              class="hidden"
              on:change={handleFileSelect}
              disabled={isUploading}
            />
          </label>
        </div>
      </div>
    </div>

    <!-- API List -->
    <div
      class="mb-6 flex flex-col md:flex-row md:items-center justify-between gap-4 mt-6"
    >
      <div class="flex items-center gap-4">
        <h2
          class="text-xl md:text-2xl font-bold text-cyan-50 font-mono tracking-wide"
        >
          MONITORED_ENDPOINTS
        </h2>
        <span
          class="bg-cyan-900 border border-cyan-500/50 text-cyan-300 text-xs font-bold px-3 py-1 rounded-md shadow-[0_0_10px_rgba(6,182,212,0.2)] font-mono tracking-wider w-fit"
          >TOTAL: {apis.length}</span
        >
      </div>

      {#if selectedApiIds.length > 0}
        <div
          class="flex items-center gap-3 animate-fade-in bg-slate-900/80 px-4 py-2 rounded-xl border border-slate-700/50 shadow-[0_0_15px_rgba(0,0,0,0.5)] backdrop-blur-xl"
        >
          <span
            class="text-sm font-bold text-cyan-400 border-r border-slate-700 pr-4 font-mono tracking-wide"
            >{selectedApiIds.length} SELECTED</span
          >
          <button
            on:click={() => (showBulkDeleteModal = true)}
            class="bg-red-500/10 text-red-500 hover:bg-red-500/20 hover:text-red-400 border border-red-500/30 font-bold py-1.5 px-4 rounded-lg shadow-[0_0_10px_rgba(239,68,68,0.1)] transition-colors text-sm flex items-center gap-2 font-mono tracking-wide"
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
              ><path d="M3 6h18"></path><path
                d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"
              ></path><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"></path></svg
            >
            Delete Selected
          </button>
        </div>
      {/if}
    </div>

    {#if apis.length === 0}
      <div
        class="border-2 border-dashed border-slate-700 rounded-3xl p-8 md:p-12 text-center bg-slate-900/40 backdrop-blur-md shadow-[0_0_20px_rgba(0,0,0,0.3)] relative overflow-hidden"
      >
        <p class="text-cyan-400 font-bold mb-2 font-mono text-lg tracking-wide">
          NO_ENDPOINTS_CONFIGURED
        </p>
        <p class="text-sm text-cyan-500/80 font-mono tracking-wide">
          IMPORT A COLLECTION OR MANUAL ENDPOINT TO INITIATE MONITORING CYCLE.
        </p>
      </div>
    {:else}
      <div
        class="bg-slate-900/80 backdrop-blur-xl border border-slate-700/50 rounded-3xl shadow-[0_8px_30px_rgb(0,0,0,0.5)] overflow-x-auto"
      >
        <table
          class="w-full text-left border-collapse min-w-[700px] font-mono text-sm tracking-wide"
        >
          <thead>
            <tr
              class="bg-slate-950/80 border-b border-slate-700/80 text-xs font-bold text-slate-500 uppercase tracking-widest"
            >
              <th class="p-3 md:p-4 w-12 text-center">
                <input
                  type="checkbox"
                  checked={allSelected}
                  {indeterminate}
                  on:change={toggleAllSelection}
                  class="w-4 h-4 text-cyan-500 bg-slate-900 border border-slate-600 rounded focus:ring-cyan-500/50 focus:ring-offset-slate-900 cursor-pointer appearance-none checked:bg-cyan-500 checked:border-cyan-500 transition-colors shadow-[0_0_10px_rgba(6,182,212,0.2)]"
                />
              </th>
              <th class="p-3 md:p-4">Method</th>
              <th class="p-3 md:p-4">Endpoint Name</th>
              <th class="p-3 md:p-4">URL</th>
              <th class="p-3 md:p-4">Expected</th>
              <th class="p-3 md:p-4 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            {#each Object.entries(groupedApis) as [folderName, folderApis]}
              <!-- Folder Header Row -->
              <tr
                class="bg-slate-800/60 border-y border-slate-700/80 transition-colors group hover:bg-slate-700/60 {dragOverItem?.folder ===
                  folderName && dragOverItem?.index === -1
                  ? '!bg-cyan-900/30'
                  : ''}"
                on:dragover={(e) => handleDragOver(e, folderName, -1)}
                on:dragleave={handleDragLeave}
                on:drop={(e) => handleDrop(e, folderName, -1)}
              >
                <td colspan="6" class="px-4 py-2">
                  <div class="flex items-center justify-between">
                    <div
                      class="flex items-center gap-2 text-cyan-100 font-bold text-sm tracking-wide"
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
                        ><path
                          d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                        ></path></svg
                      >
                      {folderName}
                      <span
                        class="text-[10px] bg-slate-900 border border-slate-700 px-2 py-0.5 rounded-md text-cyan-400 font-bold ml-2 shadow-[0_0_8px_rgba(6,182,212,0.2)]"
                        >{folderApis.length}</span
                      >
                    </div>
                    {#if folderName !== "Uncategorized"}
                      <div
                        class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <button
                          on:click={() => openEditFolder(folderName)}
                          class="text-cyan-500/80 hover:text-cyan-400 transition-colors p-1 rounded hover:bg-slate-900 border border-transparent hover:border-cyan-500/30 hover:shadow-[0_0_10px_rgba(6,182,212,0.2)]"
                          title="Rename Folder"
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
                            ><path d="M12 20h9"></path><path
                              d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"
                            ></path></svg
                          >
                        </button>
                        <button
                          on:click={() => openDeleteFolder(folderName)}
                          class="text-cyan-500/80 hover:text-red-400 transition-colors p-1 rounded hover:bg-slate-900 border border-transparent hover:border-red-500/30 hover:shadow-[0_0_10px_rgba(2ef,68,68,0.2)]"
                          title="Delete Folder"
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
                            ><polyline points="3 6 5 6 21 6"></polyline><path
                              d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                            ></path><line x1="10" y1="11" x2="10" y2="17"
                            ></line><line x1="14" y1="11" x2="14" y2="17"
                            ></line></svg
                          >
                        </button>
                      </div>
                    {/if}
                  </div>
                </td>
              </tr>

              <!-- Folder API Items -->
              {#each folderApis as api, i}
                <tr
                  draggable="true"
                  on:dragstart={(e) => handleDragStart(e, api.id)}
                  on:dragover={(e) => handleDragOver(e, folderName, i)}
                  on:dragleave={handleDragLeave}
                  on:drop={(e) => handleDrop(e, folderName, i)}
                  class="hover:bg-slate-800/40 border-b border-slate-800 transition-colors cursor-grab active:cursor-grabbing group/apirow relative"
                  class:border-t-2={dragOverItem?.folder === folderName &&
                    dragOverItem?.index === i}
                  class:border-cyan-500={dragOverItem?.folder === folderName &&
                    dragOverItem?.index === i}
                >
                  <td class="p-3 md:p-4 text-center">
                    <input
                      type="checkbox"
                      checked={selectedApiIds.includes(api.id)}
                      on:change={() => toggleSelection(api.id)}
                      class="w-4 h-4 text-cyan-500 bg-slate-900 border border-slate-600 rounded focus:ring-cyan-500/50 focus:ring-offset-slate-900 cursor-pointer appearance-none checked:bg-cyan-500 checked:border-cyan-500 transition-colors shadow-[0_0_10px_rgba(6,182,212,0.2)]"
                    />
                  </td>
                  <td class="p-3 md:p-4">
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
                  </td>
                  <td
                    class="p-3 md:p-4 text-cyan-50 truncate max-w-[150px] md:max-w-xs"
                  >
                    <div class="flex items-center gap-2">
                      <span class="font-bold">{api.name}</span>
                      {#if api.paused_until && new Date(api.paused_until) > new Date()}
                        <span class="px-1.5 py-0.5 bg-amber-950/50 border border-amber-500/40 text-amber-400 text-[9px] font-bold rounded tracking-wider">PAUSED</span>
                      {/if}
                    </div>
                  </td>
                  <td
                    class="p-3 md:p-4 text-slate-500 text-sm truncate max-w-[150px] md:max-w-xs"
                    title={api.url}>{api.url}</td
                  >
                  <td class="p-3 md:p-4 text-slate-500 text-sm">
                    <span
                      class="px-2 py-1 bg-slate-900 border border-slate-700 rounded text-[10px] font-mono"
                      >{api.expected_status_code}</span
                    >
                  </td>
                  <td
                    class="p-3 md:p-4 text-right flex items-center justify-end gap-1 sm:gap-2"
                  >
                    <button
                      on:click={() => openEditApiModal(api)}
                      class="text-cyan-500/80 hover:text-cyan-400 transition-colors p-1.5 rounded-lg hover:bg-slate-900 border border-transparent hover:border-cyan-500/30 hover:shadow-[0_0_10px_rgba(6,182,212,0.2)]"
                      title="Edit Endpoint"
                    >
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="18"
                        height="18"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        ><path d="M12 20h9"></path><path
                          d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"
                        ></path></svg
                      >
                    </button>
                    <button
                      on:click={() => openScheduleModal(api)}
                      class="text-cyan-500/80 hover:text-indigo-400 transition-colors p-1.5 rounded-lg hover:bg-slate-900 border border-transparent hover:border-indigo-500/30 hover:shadow-[0_0_10px_rgba(99,102,241,0.2)]"
                      title="Schedule Settings"
                    >
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="18"
                        height="18"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        ><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"
                        ></path><path d="M13.73 21a2 2 0 0 1-3.46 0"
                        ></path></svg
                      >
                    </button>
                    <button
                      on:click={() => openPauseApiModal(api)}
                      class="text-cyan-500/80 hover:text-amber-400 transition-colors p-1.5 rounded-lg hover:bg-slate-900 border border-transparent hover:border-amber-500/30 hover:shadow-[0_0_10px_rgba(245,158,11,0.2)]"
                      title="Pause/Resume Monitor"
                    >
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="18"
                        height="18"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                      >
                        <rect x="6" y="4" width="4" height="16"></rect>
                        <rect x="14" y="4" width="4" height="16"></rect>
                      </svg>
                    </button>
                    <button
                      on:click={() => openDeleteApiModal(api)}
                      class="text-cyan-500/80 hover:text-red-400 transition-colors p-1.5 rounded-lg hover:bg-slate-900 border border-transparent hover:border-red-500/30 hover:shadow-[0_0_10px_rgba(239,68,68,0.2)]"
                      title="Delete Endpoint"
                    >
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="18"
                        height="18"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        ><polyline points="3 6 5 6 21 6"></polyline><path
                          d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                        ></path><line x1="10" y1="11" x2="10" y2="17"
                        ></line><line x1="14" y1="11" x2="14" y2="17"
                        ></line></svg
                      >
                    </button>
                  </td>
                </tr>
              {/each}

              <!-- Invisible Drop Zone at end of folder if it has items -->
              {#if folderApis.length > 0}
                <tr
                  class="h-2 transition-all"
                  class:bg-blue-100={dragOverItem?.folder === folderName &&
                    dragOverItem?.index === folderApis.length}
                  on:dragover={(e) =>
                    handleDragOver(e, folderName, folderApis.length)}
                  on:dragleave={handleDragLeave}
                  on:drop={(e) => handleDrop(e, folderName, folderApis.length)}
                >
                  <td colspan="6" class="p-0"></td>
                </tr>
              {/if}
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {/if}
</div>

<!-- ================= MODALS ================= -->

<!-- 1. Import Mode Conflict Modal -->
<Modal bind:open={showImportModeModal} title="Import Collection">
  <div class="space-y-4">
    <p class="text-sm text-slate-500">
      This workspace already contains APIs. How would you like to handle this
      import?
    </p>
    <div class="grid grid-cols-2 gap-3">
      <button
        on:click={() => executePostmanUpload("append")}
        class="border border-cyan-500/30 bg-cyan-950/30 text-cyan-400 hover:bg-cyan-900/50 rounded-xl p-4 flex flex-col items-center justify-center text-center transition-colors"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="mb-2"
          width="24"
          height="24"
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
            y2="16"
          ></line><line x1="8" y1="12" x2="16" y2="12"></line></svg
        >
        <span class="font-bold text-sm">Append Mode</span>
        <span class="text-xs mt-1 text-cyan-500/80"
          >Add new APIs alongside the existing ones.</span
        >
      </button>

      <button
        on:click={() => executePostmanUpload("replace")}
        class="border border-red-500/30 bg-red-950/30 text-red-400 hover:bg-red-900/50 rounded-xl p-4 flex flex-col items-center justify-center text-center transition-colors"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="mb-2"
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><path d="M21 4H8l-7 8 7 8h13a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z"
          ></path><line x1="18" y1="9" x2="12" y2="15"></line><line
            x1="12"
            y1="9"
            x2="18"
            y2="15"
          ></line></svg
        >
        <span class="font-bold text-sm">Replace Mode</span>
        <span class="text-xs mt-1 text-cyan-500/80"
          >Delete all current APIs and use only the new ones.</span
        >
      </button>
    </div>
  </div>
</Modal>

<!-- 2. Manual Add Mode Conflict Modal -->
<Modal bind:open={showAddModeModal} title="Save API">
  <div class="space-y-4">
    <p class="text-sm text-slate-500">
      You are about to add a manual API endpoint. How would you like to process
      this?
    </p>
    <div class="grid grid-cols-2 gap-3">
      <button
        on:click={() => executeAddApi("append")}
        class="border border-cyan-500/30 bg-cyan-950/30 text-cyan-400 hover:bg-cyan-900/50 rounded-xl p-4 flex flex-col items-center justify-center text-center transition-colors"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="mb-2"
          width="24"
          height="24"
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
            y2="16"
          ></line><line x1="8" y1="12" x2="16" y2="12"></line></svg
        >
        <span class="font-bold text-sm">Append Mode</span>
        <span class="text-xs mt-1 text-cyan-500/80"
          >Add alongside the existing APIs.</span
        >
      </button>

      <button
        on:click={() => executeAddApi("replace")}
        class="border border-red-500/30 bg-red-950/30 text-red-400 hover:bg-red-900/50 rounded-xl p-4 flex flex-col items-center justify-center text-center transition-colors"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="mb-2"
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><path d="M21 4H8l-7 8 7 8h13a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z"
          ></path><line x1="18" y1="9" x2="12" y2="15"></line><line
            x1="12"
            y1="9"
            x2="18"
            y2="15"
          ></line></svg
        >
        <span class="font-bold text-sm">Replace Mode</span>
        <span class="text-xs mt-1 text-cyan-500/80"
          >Clear project and add only this API.</span
        >
      </button>
    </div>
  </div>
</Modal>

<!-- 3. Add API Manual Input Modal -->
<Modal
  bind:open={showAddApiModal}
  title="Create API Endpoint"
  maxWidth="max-w-2xl"
>
  <form on:submit|preventDefault={handleAddApiSubmit} class="space-y-4">
    <div
      class="grid grid-cols-1 md:grid-cols-2 gap-4 border-b border-slate-800 pb-4"
    >
      <div class="md:col-span-1">
        <label
          for="api_folder"
          class="block text-sm font-semibold text-cyan-50 mb-1"
          >Folder Name (Optional)</label
        >
        <input
          id="api_folder"
          type="text"
          bind:value={apiForm.folder}
          class="w-full px-4 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all text-sm"
          placeholder="e.g. Authentication"
        />
      </div>

      <div class="md:col-span-1">
        <label
          for="api_name"
          class="block text-sm font-semibold text-cyan-50 mb-1"
          >Endpoint Name</label
        >
        <input
          id="api_name"
          type="text"
          bind:value={apiForm.name}
          required
          class="w-full px-4 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all text-sm"
          placeholder="e.g. Fetch User Data"
        />
      </div>

      <div class="md:col-span-1">
        <label
          for="api_method"
          class="block text-sm font-semibold text-cyan-50 mb-1">Method</label
        >
        <select
          id="api_method"
          bind:value={apiForm.method}
          class="w-full px-4 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all text-sm font-medium"
        >
          <option value="GET">GET</option>
          <option value="POST">POST</option>
          <option value="PUT">PUT</option>
          <option value="DELETE">DELETE</option>
          <option value="PATCH">PATCH</option>
        </select>
      </div>

      <div class="md:col-span-2">
        <label
          for="api_url"
          class="block text-sm font-semibold text-cyan-50 mb-1"
          >Request URL</label
        >
        <div class="flex">
          <span
            class="inline-flex items-center px-3 rounded-l-lg border border-r-0 border-slate-700/50 bg-slate-800 text-cyan-500/80 text-sm font-medium"
            >URL</span
          >
          <div
            class="flex-1 w-full bg-slate-900/50 rounded-r-lg border border-slate-700/50 focus-within:ring-2 focus-within:ring-blue-500/50 transition-all text-sm overflow-hidden h-[38px] relative"
          >
            <InputWithVariables
              bind:value={apiForm.url}
              placeholder="&#123;&#123;base_url&#125;&#125;/api/v1/users"
              required={true}
              variables={envVarDict}
              on:paste={handleUrlPaste}
            />
          </div>
        </div>
      </div>
    </div>

    <div class="space-y-4">
      <!-- Parameters Toggle (Only on GET/PUT/DELETE) -->
      {#if ["GET", "PUT", "DELETE"].includes(apiForm.method)}
        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="block text-sm font-semibold text-cyan-50"
              >Query Parameters</label
            >
            <div
              class="flex bg-slate-800 p-0.5 rounded-lg border border-slate-700/50"
            >
              <button
                type="button"
                class="px-3 py-1 text-xs font-semibold rounded-md transition-all {paramMode ===
                'json'
                  ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                  : 'text-cyan-500/80 hover:text-cyan-50'}"
                on:click={() => toggleParamMode("json")}>JSON</button
              >
              <button
                type="button"
                class="px-3 py-1 text-xs font-semibold rounded-md transition-all {paramMode ===
                'kv'
                  ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                  : 'text-cyan-500/80 hover:text-cyan-50'}"
                on:click={() => toggleParamMode("kv")}>Key-Value</button
              >
            </div>
          </div>
          {#if paramMode === "json"}
            <TextareaWithVariables
              rows={2}
              bind:value={apiForm.parameters}
              variables={envVarDict}
              placeholder={`[\n  {"key": "search", "value": "keyword"}\n]`}
            />
          {:else}
            <div class="space-y-2">
              {#each paramsKV as param, i}
                <div class="flex gap-2">
                  <input
                    type="text"
                    bind:value={param.key}
                    placeholder="Key"
                    class="flex-1 px-3 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-xs font-mono"
                  />
                  <div
                    class="flex-1 bg-slate-900/50 rounded-lg border border-slate-700/50 focus-within:ring-2 focus-within:ring-blue-500/50 text-xs font-mono h-[34px] overflow-hidden relative"
                  >
                    <InputWithVariables
                      bind:value={param.value}
                      placeholder="Value"
                      variables={envVarDict}
                    />
                  </div>
                  <button
                    type="button"
                    on:click={() =>
                      (paramsKV = paramsKV.filter((_, idx) => idx !== i))}
                    class="p-2 text-slate-500 hover:text-red-500 bg-slate-900/50 hover:bg-red-950/30 border border-slate-700/50 rounded-lg transition-colors"
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      ><line x1="18" y1="6" x2="6" y2="18"></line><line
                        x1="6"
                        y1="6"
                        x2="18"
                        y2="18"
                      ></line></svg
                    >
                  </button>
                </div>
              {/each}
              <button
                type="button"
                on:click={() =>
                  (paramsKV = [...paramsKV, { key: "", value: "" }])}
                class="text-xs font-medium text-cyan-400 hover:text-cyan-400 flex items-center gap-1 mt-1"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  ><line x1="12" y1="5" x2="12" y2="19"></line><line
                    x1="5"
                    y1="12"
                    x2="19"
                    y2="12"
                  ></line></svg
                > Add Parameter
              </button>
            </div>
          {/if}
        </div>
      {/if}

      <!-- Headers Toggle -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <label class="block text-sm font-semibold text-cyan-50">Headers</label
          >
          <div
            class="flex bg-slate-800 p-0.5 rounded-lg border border-slate-700/50"
          >
            <button
              type="button"
              class="px-3 py-1 text-xs font-semibold rounded-md transition-all {headerMode ===
              'json'
                ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                : 'text-cyan-500/80 hover:text-cyan-50'}"
              on:click={() => toggleHeaderMode("json")}>JSON</button
            >
            <button
              type="button"
              class="px-3 py-1 text-xs font-semibold rounded-md transition-all {headerMode ===
              'kv'
                ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                : 'text-cyan-500/80 hover:text-cyan-50'}"
              on:click={() => toggleHeaderMode("kv")}>Key-Value</button
            >
          </div>
        </div>
        {#if headerMode === "json"}
          <TextareaWithVariables
            rows={2}
            bind:value={apiForm.headers}
            variables={envVarDict}
            placeholder={`[\n  {"key": "Authorization", "value": "Bearer token"}\n]`}
          />
        {:else}
          <div class="space-y-2">
            {#each headersKV as hdr, i}
              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={hdr.key}
                  placeholder="Header Key"
                  class="flex-1 px-3 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-xs font-mono"
                />
                <div
                  class="flex-1 bg-slate-900/50 rounded-lg border border-slate-700/50 focus-within:ring-2 focus-within:ring-blue-500/50 text-xs font-mono h-[34px] overflow-hidden relative"
                >
                  <InputWithVariables
                    bind:value={hdr.value}
                    placeholder="Value"
                    variables={envVarDict}
                  />
                </div>
                <button
                  type="button"
                  on:click={() =>
                    (headersKV = headersKV.filter((_, idx) => idx !== i))}
                  class="p-2 text-slate-500 hover:text-red-500 bg-slate-900/50 hover:bg-red-950/30 border border-slate-700/50 rounded-lg transition-colors"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    ><line x1="18" y1="6" x2="6" y2="18"></line><line
                      x1="6"
                      y1="6"
                      x2="18"
                      y2="18"
                    ></line></svg
                  >
                </button>
              </div>
            {/each}
            <button
              type="button"
              on:click={() =>
                (headersKV = [...headersKV, { key: "", value: "" }])}
              class="text-xs font-medium text-cyan-400 hover:text-cyan-400 flex items-center gap-1 mt-1"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                ><line x1="12" y1="5" x2="12" y2="19"></line><line
                  x1="5"
                  y1="12"
                  x2="19"
                  y2="12"
                ></line></svg
              > Add Header
            </button>
          </div>
        {/if}
      </div>

      <!-- Body Toggle -->
      <div class={apiForm.method === "GET" ? "opacity-50" : ""}>
        <div class="flex items-center justify-between mb-2">
          <label class="block text-sm font-semibold text-cyan-50"
            >Body / Payload</label
          >
          <div
            class="flex bg-slate-800 p-0.5 rounded-lg border border-slate-700/50"
          >
            <button
              type="button"
              disabled={apiForm.method === "GET"}
              class="px-3 py-1 text-xs font-semibold rounded-md transition-all {bodyMode ===
              'json'
                ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                : 'text-cyan-500/80 hover:text-cyan-50'} disabled:cursor-not-allowed"
              on:click={() => toggleBodyMode("json")}>JSON</button
            >
            <button
              type="button"
              disabled={apiForm.method === "GET"}
              class="px-3 py-1 text-xs font-semibold rounded-md transition-all {bodyMode ===
              'kv'
                ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                : 'text-cyan-500/80 hover:text-cyan-50'} disabled:cursor-not-allowed"
              on:click={() => toggleBodyMode("kv")}>Key-Value</button
            >
          </div>
        </div>
        {#if bodyMode === "json"}
          <TextareaWithVariables
            rows={3}
            bind:value={apiForm.body}
            variables={envVarDict}
            disabled={apiForm.method === "GET"}
            placeholder={`{ "key": "value" }`}
          />
        {:else}
          <div class="space-y-2">
            {#each bodyKV as bdy, i}
              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={bdy.key}
                  disabled={apiForm.method === "GET"}
                  placeholder="Body Key"
                  class="flex-1 px-3 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-xs font-mono disabled:cursor-not-allowed"
                />
                <div
                  class="flex-1 bg-slate-900/50 rounded-lg border border-slate-700/50 focus-within:ring-2 focus-within:ring-blue-500/50 text-xs font-mono h-[34px] overflow-hidden relative {apiForm.method ===
                  'GET'
                    ? 'opacity-50 cursor-not-allowed hidden'
                    : ''}"
                >
                  <InputWithVariables
                    bind:value={bdy.value}
                    disabled={apiForm.method === "GET"}
                    placeholder="Value"
                    variables={envVarDict}
                  />
                </div>
                <!-- Fallback input when GET mode triggers disabling -->
                <input
                  type="text"
                  bind:value={bdy.value}
                  disabled={true}
                  placeholder="Value"
                  class="flex-1 px-3 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-xs font-mono disabled:cursor-not-allowed {apiForm.method !==
                  'GET'
                    ? 'hidden'
                    : ''}"
                />
                <button
                  type="button"
                  disabled={apiForm.method === "GET"}
                  on:click={() =>
                    (bodyKV = bodyKV.filter((_, idx) => idx !== i))}
                  class="p-2 text-slate-500 hover:text-red-500 bg-slate-900/50 hover:bg-red-950/30 border border-slate-700/50 rounded-lg transition-colors disabled:cursor-not-allowed disabled:hover:bg-slate-900/50 disabled:hover:text-slate-500"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    ><line x1="18" y1="6" x2="6" y2="18"></line><line
                      x1="6"
                      y1="6"
                      x2="18"
                      y2="18"
                    ></line></svg
                  >
                </button>
              </div>
            {/each}
            <button
              type="button"
              disabled={apiForm.method === "GET"}
              on:click={() => (bodyKV = [...bodyKV, { key: "", value: "" }])}
              class="text-xs font-medium text-cyan-400 hover:text-cyan-400 flex items-center gap-1 mt-1 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:text-cyan-400"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                ><line x1="12" y1="5" x2="12" y2="19"></line><line
                  x1="5"
                  y1="12"
                  x2="19"
                  y2="12"
                ></line></svg
              > Add Body Field
            </button>
          </div>
        {/if}
        {#if apiForm.method === "GET"}
          <p class="text-xs text-orange-500 mt-1">
            Request body is disabled for GET requests.
          </p>
        {/if}
      </div>

      <div class="w-full md:w-1/2">
        <label
          for="api_expected_status"
          class="block text-sm font-semibold text-cyan-50 mb-1"
          >Expected HTTP Status Code</label
        >
        <input
          id="api_expected_status"
          type="number"
          min="100"
          max="599"
          bind:value={apiForm.expected_status_code}
          required
          class="w-full px-4 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all text-sm font-mono"
        />
      </div>
    </div>

    <div class="pb-12"></div>

    <div
      class="pt-3 flex justify-end gap-3 border-t border-slate-800 sticky bottom-0 bg-slate-900/40 z-10 -mx-6 px-6 -mb-6 pb-4 mt-2 backdrop-blur-md"
    >
      <button
        type="button"
        on:click={() => (showAddApiModal = false)}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-slate-500 bg-slate-800 hover:bg-slate-700 transition-colors"
        >Cancel</button
      >
      <button
        type="submit"
        class="px-4 py-2 text-xs rounded-xl font-semibold text-white bg-cyan-600 hover:bg-cyan-700 transition-colors shadow-sm shadow-cyan-500/20"
        >Save Endpoint</button
      >
    </div>
  </form>
</Modal>

<!-- 4. Edit API Modal -->
<Modal
  bind:open={showEditApiModal}
  title="Edit API Endpoint"
  maxWidth="max-w-2xl"
>
  <form on:submit|preventDefault={handleEditApiSubmit} class="space-y-4">
    <div
      class="grid grid-cols-1 md:grid-cols-2 gap-4 border-b border-slate-800 pb-4"
    >
      <div class="md:col-span-1">
        <label
          for="api_folder_edit"
          class="block text-sm font-semibold text-cyan-50 mb-1"
          >Folder Name (Optional)</label
        >
        <input
          id="api_folder_edit"
          type="text"
          bind:value={apiForm.folder}
          class="w-full px-4 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all text-sm"
          placeholder="e.g. Authentication"
        />
      </div>

      <div class="md:col-span-1">
        <label
          for="api_name_edit"
          class="block text-sm font-semibold text-cyan-50 mb-1"
          >Endpoint Name</label
        >
        <input
          id="api_name_edit"
          type="text"
          bind:value={apiForm.name}
          required
          class="w-full px-4 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all text-sm"
          placeholder="e.g. Fetch User Data"
        />
      </div>

      <div class="md:col-span-1">
        <label
          for="api_method_edit"
          class="block text-sm font-semibold text-cyan-50 mb-1">Method</label
        >
        <select
          id="api_method_edit"
          bind:value={apiForm.method}
          class="w-full px-4 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all text-sm font-medium"
        >
          <option value="GET">GET</option>
          <option value="POST">POST</option>
          <option value="PUT">PUT</option>
          <option value="DELETE">DELETE</option>
          <option value="PATCH">PATCH</option>
        </select>
      </div>

      <div class="md:col-span-2">
        <label
          for="api_url_edit"
          class="block text-sm font-semibold text-cyan-50 mb-1"
          >Request URL</label
        >
        <div class="flex">
          <span
            class="inline-flex items-center px-3 rounded-l-lg border border-r-0 border-slate-700/50 bg-slate-800 text-cyan-500/80 text-sm font-medium"
            >URL</span
          >
          <div
            class="flex-1 w-full bg-slate-900/50 rounded-r-lg border border-slate-700/50 focus-within:ring-2 focus-within:ring-blue-500/50 transition-all text-sm overflow-hidden h-[38px] relative"
          >
            <InputWithVariables
              bind:value={apiForm.url}
              placeholder="&#123;&#123;base_url&#125;&#125;/api/v1/users"
              required={true}
              variables={envVarDict}
              on:paste={handleUrlPaste}
            />
          </div>
        </div>
      </div>
    </div>

    <div class="space-y-4">
      <!-- Parameters Toggle (Only on GET/PUT/DELETE) -->
      {#if ["GET", "PUT", "DELETE"].includes(apiForm.method)}
        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="block text-sm font-semibold text-cyan-50"
              >Query Parameters</label
            >
            <div
              class="flex bg-slate-800 p-0.5 rounded-lg border border-slate-700/50"
            >
              <button
                type="button"
                class="px-3 py-1 text-xs font-semibold rounded-md transition-all {paramMode ===
                'json'
                  ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                  : 'text-cyan-500/80 hover:text-cyan-50'}"
                on:click={() => toggleParamMode("json")}>JSON</button
              >
              <button
                type="button"
                class="px-3 py-1 text-xs font-semibold rounded-md transition-all {paramMode ===
                'kv'
                  ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                  : 'text-cyan-500/80 hover:text-cyan-50'}"
                on:click={() => toggleParamMode("kv")}>Key-Value</button
              >
            </div>
          </div>
          {#if paramMode === "json"}
            <TextareaWithVariables
              rows={2}
              bind:value={apiForm.parameters}
              variables={envVarDict}
              placeholder={`[\n  {"key": "search", "value": "keyword"}\n]`}
            />
          {:else}
            <div class="space-y-2">
              {#each paramsKV as param, i}
                <div class="flex gap-2">
                  <input
                    type="text"
                    bind:value={param.key}
                    placeholder="Key"
                    class="flex-1 px-3 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-xs font-mono"
                  />
                  <div
                    class="flex-1 bg-slate-900/50 rounded-lg border border-slate-700/50 focus-within:ring-2 focus-within:ring-blue-500/50 text-xs font-mono h-[34px] overflow-hidden relative"
                  >
                    <InputWithVariables
                      bind:value={param.value}
                      placeholder="Value"
                      variables={envVarDict}
                    />
                  </div>
                  <button
                    type="button"
                    on:click={() =>
                      (paramsKV = paramsKV.filter((_, idx) => idx !== i))}
                    class="p-2 text-slate-500 hover:text-red-500 bg-slate-900/50 hover:bg-red-950/30 border border-slate-700/50 rounded-lg transition-colors"
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      ><line x1="18" y1="6" x2="6" y2="18"></line><line
                        x1="6"
                        y1="6"
                        x2="18"
                        y2="18"
                      ></line></svg
                    >
                  </button>
                </div>
              {/each}
              <button
                type="button"
                on:click={() =>
                  (paramsKV = [...paramsKV, { key: "", value: "" }])}
                class="text-xs font-medium text-cyan-400 hover:text-cyan-400 flex items-center gap-1 mt-1"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  ><line x1="12" y1="5" x2="12" y2="19"></line><line
                    x1="5"
                    y1="12"
                    x2="19"
                    y2="12"
                  ></line></svg
                > Add Parameter
              </button>
            </div>
          {/if}
        </div>
      {/if}

      <!-- Headers Toggle -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <label class="block text-sm font-semibold text-cyan-50">Headers</label
          >
          <div
            class="flex bg-slate-800 p-0.5 rounded-lg border border-slate-700/50"
          >
            <button
              type="button"
              class="px-3 py-1 text-xs font-semibold rounded-md transition-all {headerMode ===
              'json'
                ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                : 'text-cyan-500/80 hover:text-cyan-50'}"
              on:click={() => toggleHeaderMode("json")}>JSON</button
            >
            <button
              type="button"
              class="px-3 py-1 text-xs font-semibold rounded-md transition-all {headerMode ===
              'kv'
                ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                : 'text-cyan-500/80 hover:text-cyan-50'}"
              on:click={() => toggleHeaderMode("kv")}>Key-Value</button
            >
          </div>
        </div>
        {#if headerMode === "json"}
          <TextareaWithVariables
            rows={2}
            bind:value={apiForm.headers}
            variables={envVarDict}
            placeholder={`[\n  {"key": "Authorization", "value": "Bearer token"}\n]`}
          />
        {:else}
          <div class="space-y-2">
            {#each headersKV as hdr, i}
              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={hdr.key}
                  placeholder="Header Key"
                  class="flex-1 px-3 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-xs font-mono"
                />
                <div
                  class="flex-1 bg-slate-900/50 rounded-lg border border-slate-700/50 focus-within:ring-2 focus-within:ring-blue-500/50 text-xs font-mono h-[34px] overflow-hidden relative"
                >
                  <InputWithVariables
                    bind:value={hdr.value}
                    placeholder="Value"
                    variables={envVarDict}
                  />
                </div>
                <button
                  type="button"
                  on:click={() =>
                    (headersKV = headersKV.filter((_, idx) => idx !== i))}
                  class="p-2 text-slate-500 hover:text-red-500 bg-slate-900/50 hover:bg-red-950/30 border border-slate-700/50 rounded-lg transition-colors"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    ><line x1="18" y1="6" x2="6" y2="18"></line><line
                      x1="6"
                      y1="6"
                      x2="18"
                      y2="18"
                    ></line></svg
                  >
                </button>
              </div>
            {/each}
            <button
              type="button"
              on:click={() =>
                (headersKV = [...headersKV, { key: "", value: "" }])}
              class="text-xs font-medium text-cyan-400 hover:text-cyan-400 flex items-center gap-1 mt-1"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                ><line x1="12" y1="5" x2="12" y2="19"></line><line
                  x1="5"
                  y1="12"
                  x2="19"
                  y2="12"
                ></line></svg
              > Add Header
            </button>
          </div>
        {/if}
      </div>

      <!-- Body Toggle -->
      <div class={apiForm.method === "GET" ? "opacity-50" : ""}>
        <div class="flex items-center justify-between mb-2">
          <label class="block text-sm font-semibold text-cyan-50"
            >Body / Payload</label
          >
          <div
            class="flex bg-slate-800 p-0.5 rounded-lg border border-slate-700/50"
          >
            <button
              type="button"
              disabled={apiForm.method === "GET"}
              class="px-3 py-1 text-xs font-semibold rounded-md transition-all {bodyMode ===
              'json'
                ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                : 'text-cyan-500/80 hover:text-cyan-50'} disabled:cursor-not-allowed"
              on:click={() => toggleBodyMode("json")}>JSON</button
            >
            <button
              type="button"
              disabled={apiForm.method === "GET"}
              class="px-3 py-1 text-xs font-semibold rounded-md transition-all {bodyMode ===
              'kv'
                ? 'bg-slate-900/40 shadow-sm text-cyan-300'
                : 'text-cyan-500/80 hover:text-cyan-50'} disabled:cursor-not-allowed"
              on:click={() => toggleBodyMode("kv")}>Key-Value</button
            >
          </div>
        </div>
        {#if bodyMode === "json"}
          <textarea
            rows="3"
            bind:value={apiForm.body}
            disabled={apiForm.method === "GET"}
            class="w-full font-mono px-4 py-3 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all text-xs resize-y disabled:cursor-not-allowed"
            placeholder={`{ "key": "value" }`}
          ></textarea>
        {:else}
          <div class="space-y-2">
            {#each bodyKV as bdy, i}
              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={bdy.key}
                  disabled={apiForm.method === "GET"}
                  placeholder="Body Key"
                  class="flex-1 px-3 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-xs font-mono disabled:cursor-not-allowed"
                />
                <div
                  class="flex-1 bg-slate-900/50 rounded-lg border border-slate-700/50 focus-within:ring-2 focus-within:ring-blue-500/50 text-xs font-mono h-[34px] overflow-hidden relative {apiForm.method ===
                  'GET'
                    ? 'opacity-50 cursor-not-allowed hidden'
                    : ''}"
                >
                  <InputWithVariables
                    bind:value={bdy.value}
                    disabled={apiForm.method === "GET"}
                    placeholder="Value"
                    variables={envVarDict}
                  />
                </div>
                <!-- Fallback input when GET mode triggers disabling -->
                <input
                  type="text"
                  bind:value={bdy.value}
                  disabled={true}
                  placeholder="Value"
                  class="flex-1 px-3 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-xs font-mono disabled:cursor-not-allowed {apiForm.method !==
                  'GET'
                    ? 'hidden'
                    : ''}"
                />
                <button
                  type="button"
                  disabled={apiForm.method === "GET"}
                  on:click={() =>
                    (bodyKV = bodyKV.filter((_, idx) => idx !== i))}
                  class="p-2 text-slate-500 hover:text-red-500 bg-slate-900/50 hover:bg-red-950/30 border border-slate-700/50 rounded-lg transition-colors disabled:cursor-not-allowed disabled:hover:bg-slate-900/50 disabled:hover:text-slate-500"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    ><line x1="18" y1="6" x2="6" y2="18"></line><line
                      x1="6"
                      y1="6"
                      x2="18"
                      y2="18"
                    ></line></svg
                  >
                </button>
              </div>
            {/each}
            <button
              type="button"
              disabled={apiForm.method === "GET"}
              on:click={() => (bodyKV = [...bodyKV, { key: "", value: "" }])}
              class="text-xs font-medium text-cyan-400 hover:text-cyan-400 flex items-center gap-1 mt-1 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:text-cyan-400"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                ><line x1="12" y1="5" x2="12" y2="19"></line><line
                  x1="5"
                  y1="12"
                  x2="19"
                  y2="12"
                ></line></svg
              > Add Body Field
            </button>
          </div>
        {/if}
        {#if apiForm.method === "GET"}
          <p class="text-xs text-orange-500 mt-1">
            Request body is disabled for GET requests.
          </p>
        {/if}
      </div>

      <div class="w-full md:w-1/2">
        <label
          for="api_expected_status_edit"
          class="block text-sm font-semibold text-cyan-50 mb-1"
          >Expected HTTP Status Code</label
        >
        <input
          id="api_expected_status_edit"
          type="number"
          min="100"
          max="599"
          bind:value={apiForm.expected_status_code}
          required
          class="w-full px-4 py-2 bg-slate-900/50 rounded-lg border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 transition-all text-sm font-mono"
        />
      </div>
    </div>

    <div class="pb-12"></div>

    <div
      class="pt-3 flex justify-end gap-3 border-t border-slate-800 sticky bottom-0 bg-slate-900/40 z-10 -mx-6 px-6 -mb-6 pb-4 mt-2 backdrop-blur-md"
    >
      <button
        type="button"
        on:click={() => (showEditApiModal = false)}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-slate-500 bg-slate-800 hover:bg-slate-700 transition-colors"
        >Cancel</button
      >
      <button
        type="submit"
        class="px-4 py-2 text-xs rounded-xl font-semibold text-white bg-cyan-600 hover:bg-cyan-700 transition-colors shadow-sm shadow-cyan-500/20"
        >Save Changes</button
      >
    </div>
  </form>
</Modal>

<!-- 5. Delete API Modal -->
<Modal bind:open={showDeleteApiModal} title="Remove Endpoint">
  <div class="space-y-4">
    <div
      class="bg-amber-50 text-amber-800 p-4 rounded-xl border border-amber-100 flex items-start gap-3"
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
        class="shrink-0 mt-0.5"
        ><path
          d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"
        ></path><line x1="12" y1="9" x2="12" y2="13"></line><line
          x1="12"
          y1="17"
          x2="12.01"
          y2="17"
        ></line></svg
      >
      <div>
        <p class="font-bold text-sm">Warning: Removing API Endpoint</p>
        <p class="text-xs mt-1 text-amber-400">
          Are you sure you want to remove <span
            class="font-semibold px-1 rounded bg-amber-900/30"
            >{selectedApi?.name}</span
          >? This endpoint will become inactive and hidden from the dashboard,
          but monitoring history is preserved in the database for recovery.
        </p>
      </div>
    </div>
    <div class="flex justify-end gap-3 pt-2">
      <button
        on:click={() => (showDeleteApiModal = false)}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-slate-500 bg-slate-800 hover:bg-slate-700 transition-colors"
        >Cancel</button
      >
      <button
        on:click={handleDeleteApiSubmit}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-white bg-red-600 hover:bg-red-700 transition-colors shadow-sm shadow-red-500/20"
        >Yes, Remove</button
      >
    </div>
  </div>
</Modal>

<!-- 5a. Bulk Delete API Modal -->
<Modal bind:open={showBulkDeleteModal} title="Delete Selected APIs">
  <div class="space-y-4">
    <div
      class="bg-red-950/30 text-red-800 p-4 rounded-xl border border-red-100 flex items-start gap-3"
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
        class="shrink-0 mt-0.5"
        ><circle cx="12" cy="12" r="10"></circle><line
          x1="15"
          y1="9"
          x2="9"
          y2="15"
        ></line><line x1="9" y1="9" x2="15" y2="15"></line></svg
      >
      <div>
        <p class="font-bold text-sm">Warning: Bulk Endpoint Deletion</p>
        <p class="text-xs mt-1 text-red-400">
          Are you sure you want to delete <span class="font-bold"
            >{selectedApiIds.length}</span
          > selected endpoints? This action cannot be undone and will stop monitoring
          for all selected items immediately.
        </p>
      </div>
    </div>
    <div class="flex justify-end gap-3 pt-2">
      <button
        on:click={() => (showBulkDeleteModal = false)}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-slate-500 bg-slate-800 hover:bg-slate-700 transition-colors"
        >Cancel</button
      >
      <button
        on:click={handleBulkDeleteSubmit}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-white bg-red-600 hover:bg-red-700 transition-colors shadow-sm shadow-red-500/20 flex items-center gap-2"
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
          ><path d="M3 6h18"></path><path
            d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"
          ></path><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"></path></svg
        >
        Confirm Delete</button
      >
    </div>
  </div>
</Modal>

<!-- 6. Schedule / Alert Config Modal -->
<Modal
  bind:open={showScheduleModal}
  title="Schedule Trigger ({selectedApi?.name})"
>
  <div class="space-y-4">
    <p class="text-sm text-slate-500 mb-2">
      Set up when this API endpoint should be checked.
    </p>

    <div>
      <select
        bind:value={scheduleConfig.mode}
        class="w-full bg-slate-900/50 border border-slate-600 text-cyan-50 rounded-xl focus:ring-cyan-500 focus:border-cyan-500 block p-2.5 outline-none transition-all"
      >
        {#each scheduleModes as mode}
          <option value={mode}>{mode}</option>
        {/each}
      </select>
    </div>

    {#if scheduleConfig.mode !== "Week timer"}
      <!-- Minute & Hour Timers -->
      <div
        class="flex items-center justify-between border-b border-slate-700/50 pb-3"
      >
        <label class="text-sm font-medium text-cyan-50">Check interval:</label>
      </div>
      <div class="flex items-center gap-3">
        <span class="text-cyan-50 font-medium text-sm">Every</span>
        <input
          type="number"
          min="1"
          bind:value={scheduleConfig.value}
          class="w-24 bg-slate-900/40 border border-slate-600 text-cyan-50 rounded-xl focus:ring-cyan-500 focus:border-cyan-500 block p-2 outline-none transition-all text-center"
        />
        <span class="text-cyan-50 font-medium"
          >{scheduleConfig.mode === "Minute timer" ? "minutes" : "hours"}</span
        >
      </div>
      <p class="text-xs text-cyan-500/80">
        The endpoint will be constantly monitored every {scheduleConfig.value}
        {scheduleConfig.mode === "Minute timer" ? "minutes" : "hours"} based on this
        setting.
      </p>
    {:else}
      <!-- Week Timer -->
      <div class="space-y-3 pt-2">
        <div>
          <select
            bind:value={scheduleConfig.day}
            class="w-full bg-slate-900/50 border border-slate-600 text-cyan-50 rounded-xl focus:ring-cyan-500 focus:border-cyan-500 block p-2.5 outline-none transition-all"
          >
            {#each weekDays as day}
              <option value={day}>{day}</option>
            {/each}
          </select>
        </div>
        <div>
          <select
            bind:value={scheduleConfig.time}
            class="w-full bg-slate-900/50 border border-slate-600 text-cyan-50 rounded-xl focus:ring-cyan-500 focus:border-cyan-500 block p-2.5 outline-none transition-all"
          >
            {#each timeOptions as time}
              <option value={time}>{time}</option>
            {/each}
          </select>
        </div>
        <p class="text-xs text-cyan-500/80 mt-2">
          The health checker will initiate an automatic check {scheduleConfig.day.toLowerCase()}
          at {scheduleConfig.time}.
        </p>
      </div>
    {/if}

    <div class="flex gap-3 justify-end pt-5 mt-2 border-t border-slate-800">
      <button
        on:click={() => (showScheduleModal = false)}
        class="px-4 py-2 text-xs text-cyan-50 bg-slate-900/40 border border-slate-600 rounded-xl hover:bg-slate-900/50 font-medium transition-colors"
        >Cancel</button
      >
      <button
        on:click={handleScheduleSubmit}
        class="px-4 py-2 text-xs bg-indigo-600 text-white rounded-xl hover:bg-indigo-700 font-medium transition-colors shadow-sm"
      >
        Save Schedule
      </button>
    </div>
  </div>
</Modal>

<!-- 8. Edit Folder Modal -->
<Modal bind:open={showEditFolderModal} title="Rename Folder">
  <div class="space-y-4">
    <div>
      <label class="block text-sm font-semibold text-cyan-50 mb-1.5"
        >New Folder Name</label
      >
      <input
        type="text"
        bind:value={editFolderName}
        class="w-full bg-slate-900/50 border border-slate-600 text-cyan-50 rounded-xl focus:ring-cyan-500 focus:border-cyan-500 block p-2.5 outline-none transition-all"
      />
    </div>

    <div class="flex justify-end gap-3 pt-3 border-t border-slate-800">
      <button
        on:click={() => (showEditFolderModal = false)}
        class="px-4 py-2 text-xs text-cyan-50 bg-slate-900/40 border border-slate-600 rounded-xl hover:bg-slate-900/50 font-medium transition-colors"
        >Cancel</button
      >
      <button
        on:click={handleEditFolderSubmit}
        disabled={!editFolderName.trim() ||
          editFolderName.trim() === selectedFolderToEdit}
        class="px-4 py-2 text-xs bg-cyan-600 text-white rounded-xl hover:bg-cyan-700 font-medium transition-colors shadow-sm text-xs disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Save Changes
      </button>
    </div>
  </div>
</Modal>

<!-- 9. Delete Folder Modal -->
<Modal bind:open={showDeleteFolderModal} title="Delete Folder">
  <div class="space-y-4">
    <div
      class="bg-amber-50 text-amber-800 p-4 rounded-xl border border-amber-100 flex items-start gap-3"
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
        class="shrink-0 mt-0.5"
        ><path
          d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"
        ></path><line x1="12" y1="9" x2="12" y2="13"></line><line
          x1="12"
          y1="17"
          x2="12.01"
          y2="17"
        ></line></svg
      >
      <div>
        <p class="font-bold text-sm">Warning: Removing Folder</p>
        <p class="text-xs mt-1 text-amber-400">
          Are you sure you want to delete <span
            class="font-semibold px-1 rounded bg-amber-900/30"
            >{selectedFolderToDelete}</span
          >? All endpoints inside will be moved to "Uncategorized".
        </p>
      </div>
    </div>
    <div class="flex justify-end gap-3 pt-2">
      <button
        on:click={() => (showDeleteFolderModal = false)}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-slate-500 bg-slate-800 hover:bg-slate-700 transition-colors"
        >Cancel</button
      >
      <button
        on:click={handleDeleteFolderSubmit}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-white bg-red-600 hover:bg-red-700 transition-colors shadow-sm shadow-red-500/20"
        >Yes, Delete</button
      >
    </div>
  </div>
</Modal>

<!-- Env Vars Modal -->
<Modal
  bind:open={showEnvVarsModal}
  title="Environment Variables"
  maxWidth="max-w-2xl"
>
  <div class="space-y-4 relative">
    <p class="text-sm text-slate-500 mb-2">
      Define variables that can be shared across all API endpoints in this
      project. Use <code>{`{{VariableName}}`}</code> in your API URLs, Headers, or
      Body.
    </p>

    <div class="space-y-3 pb-20">
      {#each envVarsKV as pair, i}
        <div class="flex gap-2 items-center">
          <input
            type="text"
            bind:value={pair.key}
            placeholder="Variable Key (e.g. base_url)"
            class="flex-1 bg-slate-900/50 border border-slate-700/50 text-cyan-50 text-sm rounded-lg focus:ring-cyan-500 focus:border-cyan-500 block p-2 outline-none font-mono"
          />
          <input
            type="text"
            bind:value={pair.value}
            placeholder="Value"
            class="flex-1 bg-slate-900/50 border border-slate-700/50 text-cyan-50 text-sm rounded-lg focus:ring-cyan-500 focus:border-cyan-500 block p-2 outline-none font-mono"
          />
          <button
            on:click={() => removeEnvVarRow(i)}
            class="p-2 text-cyan-500/80 hover:text-red-400 hover:bg-slate-800 rounded-lg transition-colors border border-transparent hover:border-red-500/30"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="18"
              height="18"
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
      {/each}

      <button
        on:click={addEnvVarRow}
        class="text-cyan-500 hover:text-cyan-400 text-sm font-semibold flex items-center gap-1.5 mt-2"
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
          ><line x1="12" y1="5" x2="12" y2="19"></line><line
            x1="5"
            y1="12"
            x2="19"
            y2="12"
          ></line></svg
        >
        Add Variable
      </button>
    </div>

    <div
      class="flex justify-end gap-3 pt-3 border-t border-slate-800 absolute bottom-0 left-0 w-full bg-slate-900/95 px-6 pb-4 mt-2"
    >
      <button
        on:click={() => (showEnvVarsModal = false)}
        class="px-4 py-2 text-slate-500 bg-slate-800 border border-slate-700 rounded-xl hover:bg-slate-700 font-medium transition-colors text-xs"
        >Cancel</button
      >
      <button
        on:click={saveEnvVars}
        disabled={isSavingEnvVars}
        class="px-4 py-2 bg-cyan-600 text-cyan-50 rounded-xl hover:bg-cyan-700 font-medium transition-colors shadow-[0_0_15px_rgba(6,182,212,0.3)] text-xs disabled:opacity-50 flex items-center gap-2"
      >
        {#if isSavingEnvVars}
          <svg
            class="animate-spin h-4 w-4 text-cyan-50"
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
          Saving...
        {:else}
          Save Variables
        {/if}
      </button>
    </div>
  </div>
</Modal>

<!-- 7. Create Folder Modal -->
<Modal bind:open={showFolderModal} title="Create New Folder">
  <div class="space-y-4">
    <p class="text-xs text-slate-500 mb-2">
      Folders help you organize your endpoints logically.
    </p>

    <div>
      <label class="block text-xs font-semibold text-slate-500 mb-1.5" for="new_folder_name_input"
        >Folder Name</label
      >
      <input
        id="new_folder_name_input"
        type="text"
        bind:value={newFolderName}
        placeholder="e.g. Authentication APIs"
        class="w-full bg-slate-900/50 border border-slate-700/50 text-cyan-50 rounded-xl focus:ring-cyan-500 focus:border-cyan-500 block p-2.5 outline-none transition-all text-sm"
      />
    </div>

    <div class="flex justify-end gap-3 pt-3 border-t border-slate-800">
      <button
        on:click={() => (showFolderModal = false)}
        class="px-4 py-2 text-slate-500 bg-slate-800 border border-slate-700 rounded-xl hover:bg-slate-700 font-medium transition-colors text-xs"
        >Cancel</button
      >
      <button
        on:click={handleAddFolder}
        disabled={!newFolderName.trim()}
        class="px-4 py-2 bg-cyan-600 text-cyan-50 rounded-xl hover:bg-cyan-700 font-medium transition-colors shadow-[0_0_15px_rgba(6,182,212,0.3)] text-xs disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Create Folder
      </button>
    </div>
  </div>
</Modal>

<!-- Pause API Modal -->
<Modal bind:open={showPauseApiModal} title="Pause Monitor ({selectedApi?.name})">
  <div class="space-y-4">

    <!-- Type selector tabs -->
    <div class="flex gap-2 bg-slate-900/60 border border-slate-700/50 rounded-xl p-1">
      <button
        type="button"
        on:click={() => (pauseType = 'duration')}
        class="flex-1 px-3 py-1.5 text-xs font-semibold rounded-lg transition-all {pauseType === 'duration' ? 'bg-amber-600 text-white shadow' : 'text-slate-400 hover:text-cyan-50'}"
      >Pause for duration</button>
      <button
        type="button"
        on:click={() => (pauseType = 'indefinite')}
        class="flex-1 px-3 py-1.5 text-xs font-semibold rounded-lg transition-all {pauseType === 'indefinite' ? 'bg-amber-600 text-white shadow' : 'text-slate-400 hover:text-cyan-50'}"
      >Indefinite</button>
      <button
        type="button"
        on:click={() => (pauseType = 'resume')}
        class="flex-1 px-3 py-1.5 text-xs font-semibold rounded-lg transition-all {pauseType === 'resume' ? 'bg-emerald-600 text-white shadow' : 'text-slate-400 hover:text-cyan-50'}"
      >Resume Now</button>
    </div>

    {#if pauseType === 'duration'}
      <!-- Preset buttons -->
      <div>
        <p class="text-xs font-semibold text-slate-400 mb-2">Quick presets</p>
        <div class="flex flex-wrap gap-2">
          {#each [15, 30, 60, 180, 360, 720, 1440] as min}
            <button
              type="button"
              on:click={() => (pauseMinutes = min)}
              class="px-3 py-1 text-xs rounded-lg border transition-all {pauseMinutes === min ? 'bg-amber-600 border-amber-500 text-white' : 'border-slate-700 bg-slate-900/50 text-slate-300 hover:border-amber-500/50 hover:text-amber-300'}"
            >
              {min < 60 ? `${min}m` : min === 60 ? '1h' : min < 1440 ? `${min/60}h` : '24h'}
            </button>
          {/each}
        </div>
      </div>

      <!-- Custom minute input -->
      <div>
        <label for="pause_minutes_input" class="block text-xs font-semibold text-cyan-50 mb-1.5">
          Custom duration (minutes)
        </label>
        <div class="flex items-center gap-3">
          <input
            id="pause_minutes_input"
            type="number"
            min="1"
            max="43200"
            bind:value={pauseMinutes}
            class="w-32 px-3 py-2 bg-slate-900/50 border border-slate-700/50 text-cyan-50 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500/50 transition-all text-sm font-mono"
          />
          <span class="text-sm text-slate-400">
            {#if pauseMinutes < 60}
              {pauseMinutes} minute(s)
            {:else}
              {(pauseMinutes / 60).toFixed(1)} hour(s)
            {/if}
          </span>
        </div>
      </div>
    {:else if pauseType === 'indefinite'}
      <div class="bg-amber-950/30 border border-amber-500/30 rounded-xl p-3 flex items-start gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0 mt-0.5 text-amber-400"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        <p class="text-xs text-amber-300">Monitor จะถูกหยุดแบบไม่มีกำหนด จนกว่าจะ Resume ด้วยตนเอง</p>
      </div>
    {:else}
      <div class="bg-emerald-950/30 border border-emerald-500/30 rounded-xl p-3 flex items-start gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0 mt-0.5 text-emerald-400"><polyline points="20 6 9 17 4 12"/></svg>
        <p class="text-xs text-emerald-300">Monitor จะกลับมา Check ตาม Schedule ที่กำหนดไว้ทันที</p>
      </div>
    {/if}

    <div class="flex justify-end gap-3 pt-3 border-t border-slate-800">
      <button
        type="button"
        on:click={() => (showPauseApiModal = false)}
        class="px-4 py-2 text-xs text-cyan-50 bg-slate-900/40 border border-slate-600 rounded-xl hover:bg-slate-900/50 font-medium transition-colors"
      >
        Cancel
      </button>
      <button
        type="button"
        on:click={handlePauseApiSubmit}
        class="px-4 py-2 text-xs font-semibold text-white rounded-xl transition-colors shadow-sm {pauseType === 'resume' ? 'bg-emerald-600 hover:bg-emerald-700' : 'bg-amber-600 hover:bg-amber-700'}"
      >
        {pauseType === 'resume' ? 'Resume Monitor' : 'Confirm Pause'}
      </button>
    </div>
  </div>
</Modal>
