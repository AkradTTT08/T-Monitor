<script lang="ts">
  import { page } from "$app/stores";
  import { onMount, tick } from "svelte";
  import Swal from "sweetalert2";
  import { systemAlert, systemToast } from "$lib/swal-design";
  import { API_BASE_URL } from "$lib/config";
  import Modal from "$lib/components/Modal.svelte";
  import InputWithVariables from "$lib/components/InputWithVariables.svelte";
  import TextareaWithVariables from "$lib/components/TextareaWithVariables.svelte";

  $: projectId = $page.params.id;
  $: backUrl = $page.url.searchParams.get('back') || '/dashboard';

  let project: any = null;
  let apis: any[] = [];
  let now = new Date();

  onMount(() => {
    const interval = setInterval(() => {
      now = new Date();
    }, 1000);
    return () => clearInterval(interval);
  });

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
  let selectedApiIds: string[] = [];
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
  
  // Project Members State
  let projectMembers: any[] = [];
  let companyMembers: any[] = [];
  let showMembersModal = false;
  let isAddingMember = false;
  let selectedMemberId: string | null = null;

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
    response_script: "",
    recovery_script: "",
  };

  // ===== Script Templates =====
  const responseScriptTemplates = [
    {
      label: "-- เลือก Template --",
      value: "",
      script: "",
    },
    {
      label: "✅ ตรวจสอบ Status 200",
      value: "check_status_200",
      script:
`// ตรวจสอบว่า Response Status เป็น 200
if (response.status === 200) {
    console.log("✅ API ตอบกลับสำเร็จ Status:", response.status);
} else {
    console.error("❌ Status ไม่ถูกต้อง:", response.status);
}`,
    },
    {
      label: "🔑 บันทึก Token ลง ENV",
      value: "save_token_env",
      script:
`// บันทึก Token จาก Response Body ลง Environment Variable
const data = JSON.parse(response.body);
if (data.token) {
    setEnv("AUTH_TOKEN", data.token);
    console.log("✅ Token บันทึกลง ENV สำเร็จ");
} else {
    console.error("❌ ไม่พบ token ใน response");
}`,
    },
    {
      label: "🔑 บันทึก AccessToken + RefreshToken",
      value: "save_access_refresh_token",
      script:
`// บันทึก AccessToken และ RefreshToken ลง ENV
const data = JSON.parse(response.body);
if (data.accessToken) setEnv("ACCESS_TOKEN", data.accessToken);
if (data.refreshToken) setEnv("REFRESH_TOKEN", data.refreshToken);
console.log("✅ Tokens อัปเดตแล้ว");`,
    },
    {
      label: "📦 บันทึก Field จาก JSON ลง ENV",
      value: "save_field_env",
      script:
`// บันทึก Field ที่ต้องการจาก JSON Response ลง ENV
const data = JSON.parse(response.body);
// เปลี่ยน "fieldName" และ "ENV_KEY" ตามที่ต้องการ
const value = data.fieldName;
if (value !== undefined) {
    setEnv("ENV_KEY", String(value));
    console.log("✅ บันทึก ENV_KEY =", value);
}`,
    },
    {
      label: "🆔 บันทึก ID จาก Array แรก",
      value: "save_first_id",
      script:
`// บันทึก ID จาก item แรกของ Array ใน Response
const data = JSON.parse(response.body);
if (Array.isArray(data) && data.length > 0) {
    setEnv("FIRST_ID", String(data[0].id));
    console.log("✅ FIRST_ID =", data[0].id);
} else {
    console.warn("⚠️ Array ว่างหรือ Response ไม่ใช่ Array");
}`,
    },
    {
      label: "📋 Log Response ทั้งหมด",
      value: "log_response",
      script:
`// แสดง Response ใน Console เพื่อ Debug
console.log("📋 Status:", response.status);
console.log("📋 Headers:", JSON.stringify(response.headers));
console.log("📋 Body:", response.body);`,
    },
    {
      label: "🔀 เงื่อนไข: บันทึก Token ตาม Status",
      value: "conditional_token",
      script:
`// บันทึก Token เฉพาะเมื่อ Status 200 หรือ 201
if (response.status === 200 || response.status === 201) {
    const data = JSON.parse(response.body);
    if (data.token) {
        setEnv("AUTH_TOKEN", data.token);
        console.log("✅ Token อัปเดตแล้ว");
    }
} else {
    console.warn("⚠️ ข้ามการบันทึก Token เนื่องจาก Status:", response.status);
}`,
    },
  ];

  const recoveryScriptTemplates = [
    {
      label: "-- เลือก Template --",
      value: "",
      script: "",
    },
    {
      label: "🔄 Refresh Token อัตโนมัติ",
      value: "refresh_token",
      script:
`// Auto-Refresh Token เมื่อ API พัง
console.log("⚠️ API พังเนื่องจาก:", errorReason);

try {
    const res = await fetch("https://your-api.com/auth/refresh", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken: getEnv("REFRESH_TOKEN") })
    });
    const data = await res.json();
    if (data.token) {
        setEnv("AUTH_TOKEN", data.token);
        console.log("✅ Token Refresh สำเร็จ");
    }
} catch (e) {
    console.error("❌ Refresh Token ล้มเหลว:", e);
}`,
    },
    {
      label: "🔑 Login ใหม่ด้วย Credentials",
      value: "relogin",
      script:
`// Login ใหม่เพื่อขอ Token ใหม่
console.log("⚠️ พยายาม Re-Login เนื่องจาก:", errorReason);

try {
    const res = await fetch("https://your-api.com/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            username: getEnv("USERNAME"),
            password: getEnv("PASSWORD")
        })
    });
    const data = await res.json();
    if (data.token) {
        setEnv("AUTH_TOKEN", data.token);
        console.log("✅ Re-Login สำเร็จ");
    }
} catch (e) {
    console.error("❌ Re-Login ล้มเหลว:", e);
}`,
    },
    {
      label: "📋 Log Error แล้ว Skip",
      value: "log_and_skip",
      script:
`// บันทึก Error และข้ามไป (ไม่ทำ Recovery)
console.error("❌ API พัง:", errorReason);
console.log("⏭️ ข้ามการ Retry ครั้งนี้");`,
    },
    {
      label: "⏳ รอแล้วลองใหม่ (Delay Retry)",
      value: "delay_retry",
      script:
`// รอก่อนที่ระบบจะ Retry
console.log("⏳ รอ 5 วินาทีก่อน Retry...", "Error:", errorReason);
await new Promise(resolve => setTimeout(resolve, 5000));
console.log("✅ พร้อม Retry แล้ว");`,
    },
    {
      label: "🔀 Refresh เฉพาะเมื่อ 401",
      value: "refresh_on_401",
      script:
`// Refresh Token เฉพาะเมื่อได้ 401 Unauthorized
if (errorReason && errorReason.includes("401")) {
    console.log("🔒 401 Detected - กำลัง Refresh Token...");
    const res = await fetch("https://your-api.com/auth/refresh", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken: getEnv("REFRESH_TOKEN") })
    });
    const data = await res.json();
    if (data.token) {
        setEnv("AUTH_TOKEN", data.token);
        console.log("✅ Token Refresh สำเร็จ");
    }
} else {
    console.warn("⚠️ ข้าม Recovery เนื่องจาก Error ไม่ใช่ 401:", errorReason);
}`,
    },
  ];

  function applyResponseTemplate(e: Event) {
    const sel = (e.target as HTMLSelectElement).value;
    const tpl = responseScriptTemplates.find(t => t.value === sel);
    if (tpl && tpl.script) {
      apiForm.response_script = tpl.script;
    }
    (e.target as HTMLSelectElement).value = "";
  }

  function applyRecoveryTemplate(e: Event) {
    const sel = (e.target as HTMLSelectElement).value;
    const tpl = recoveryScriptTemplates.find(t => t.value === sel);
    if (tpl && tpl.script) {
      apiForm.recovery_script = tpl.script;
    }
    (e.target as HTMLSelectElement).value = "";
  }

  
  // API Search and Pagination State
  let apiSearchQuery = "";
  let apiPage = 1;
  const apiLimit = 10;

  // Reactively filter APIs based on search query
  $: filteredApisForDisplay = apis.filter(api => {
    if (!apiSearchQuery.trim()) return true;
    const q = apiSearchQuery.toLowerCase();
    return (
      (api.name || "").toLowerCase().includes(q) ||
      (api.url || "").toLowerCase().includes(q) ||
      (api.folder || "").toLowerCase().includes(q)
    );
  });

  // Calculate total pages for APIs
  $: totalApiPages = Math.ceil(filteredApisForDisplay.length / apiLimit);

  // Paginate filtered APIs
  $: paginatedApisForDisplay = filteredApisForDisplay.slice(
    (apiPage - 1) * apiLimit,
    apiPage * apiLimit
  );

  // Reset to first page when search changes
  $: if (apiSearchQuery !== undefined) {
    apiPage = 1;
  }

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
  let draggingApiId: string | null = null;
  let dragOverItem: { folder: string; index: number } | null = null;

  // Derived state to group APIs by Folder
  $: groupedApis = (() => {
    const groups: Record<string, any[]> = {};

    // Initialize custom folders empty
    customFolders.forEach((f) => (groups[f] = []));

    // Use paginated list instead of full apis list
    paginatedApisForDisplay
      .sort((a, b) => a.order_index - b.order_index)
      .forEach((api) => {
        const folder = api.folder || "Uncategorized";
        if (!groups[folder]) groups[folder] = [];
        groups[folder].push(api);
      });

    // Make sure 'Uncategorized' defaults first or last
    if (!groups["Uncategorized"] && Object.keys(groups).length === 0 && paginatedApisForDisplay.length > 0) {
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
        systemAlert.fire({
          icon: "error",
          title: "Import Failed",
          text: "Could not parse the pasted cURL command.",
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

      systemToast.fire({
        icon: "success",
        title: "cURL Imported",
        text: "Successfully parsed cURL command.",
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
  function handleDragStart(e: DragEvent, apiId: string) {
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

  // ===== Centralized auth-error handler =====
  // 401 = token expired → logout and go to login
  // 403 = forbidden (e.g. not a member) → show alert, stay on page
  async function handleAuthError(res: Response, action = "perform this action") {
    if (res.status === 401) {
      localStorage.removeItem("monitor_token");
      localStorage.removeItem("monitor_selected_project");
      window.location.href = "/login";
      return true;
    }
    if (res.status === 403) {
      let msg = `You don't have permission to ${action}.`;
      try {
        const data = await res.clone().json();
        if (data.error) msg = data.error;
      } catch {}
      systemAlert.fire({
        icon: "error",
        title: "Permission Denied",
        text: msg,
      });
      return true;
    }
    return false;
  }

  // Re-fetch whenever projectId changes
  $: if (projectId) {
    console.log("Project ID changed to:", projectId);
    fetchProjectDetails(projectId);
    fetchMembersData();
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

      // 401 = token expired → redirect to login
      if (projRes.status === 401) {
        localStorage.removeItem("monitor_token");
        window.location.href = "/login";
        return;
      }

      if (projRes.ok) {
        project = await projRes.json();
      } else {
        // 403/404 = not a member or not found → go back to dashboard
        localStorage.removeItem("monitor_selected_project");
        window.location.href = "/dashboard";
        return;
      }
      if (apisRes.ok) {
        const data = await apisRes.json();
        apis = Array.isArray(data) ? data : [];
      } else {
        apis = [];
      }
    } catch (err) {
      console.error(err);
    } finally {
      isLoading = false;
    }
  }

  async function fetchMembersData() {
    if (!projectId) return;
    try {
      const token = localStorage.getItem("monitor_token");
      
      // Fetch project members
      const pmRes = await fetch(`${API_BASE_URL}/api/v1/projects/${projectId}/members`, {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (pmRes.ok) {
        projectMembers = await pmRes.json();
        console.log("Project Members fetched:", projectMembers);
      }

      // Fetch company members (candidates)
      if (project && project.company_id) {
        const cmRes = await fetch(`${API_BASE_URL}/api/v1/companies/${project.company_id}`, {
          headers: { Authorization: `Bearer ${token}` }
        });
        if (cmRes.ok) {
          const companyData = await cmRes.json();
          companyMembers = companyData.members || [];
          console.log("Company Members fetched:", companyMembers);
        }
      }
    } catch (err) {
      console.error("Failed to fetch members data:", err);
    }
  }

  async function addProjectMember() {
    console.log("addProjectMember called. selectedMemberId:", selectedMemberId);
    if (!selectedMemberId) {
      systemAlert.fire("Error", "Please select a member first", "error");
      return;
    }
    isAddingMember = true;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects/${projectId}/members`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          user_id: selectedMemberId,
          role: "member"
        })
      });

      if (res.ok) {
        systemToast.fire({ icon: 'success', title: 'Member added' });
        await fetchMembersData();
        selectedMemberId = null;
      } else {
        const error = await res.json();
        console.error("Add member error response:", error);
        systemAlert.fire('Error', error.error || 'Failed to add member', 'error');
      }
    } catch (err) {
      console.error("Network/System error adding member:", err);
      systemAlert.fire('Error', 'A system error occurred', 'error');
    } finally {
      isAddingMember = false;
    }
  }

  async function removeProjectMember(userId: string) {
    const confirm = await systemAlert.fire({
      title: 'Are you sure?',
      text: "This user will lose access to this project.",
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText: 'Yes, remove'
    });

    if (!confirm.isConfirmed) return;

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects/${projectId}/members/${userId}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` }
      });

      if (res.ok) {
        systemToast.fire({ icon: 'success', title: 'Member removed' });
        await fetchMembersData();
      }
    } catch (err) {
      console.error(err);
    }
  }

  function openMembersModal() {
    fetchMembersData();
    showMembersModal = true;
  }

  // --- Bulk Selection Logic --- //
  $: allApisList = apis; // Flatted list of all APIs for the project
  $: allSelected = apis.length > 0 && selectedApiIds.length === apis.length;
  $: indeterminate =
    selectedApiIds.length > 0 && selectedApiIds.length < apis.length;

  let isAnalyzingRCA = false;
  async function analyzeIncident(logId: string) {
    if (isAnalyzingRCA) return;
    isAnalyzingRCA = true;
    
    Swal.fire({
      title: 'AI is analyzing...',
      text: 'Please wait while Ollama processes the error logs.',
      allowOutsideClick: false,
      background: '#0f172a',
      color: '#f8fafc',
      didOpen: () => { Swal.showLoading(); }
    });

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/ai/analyze-incident`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ log_id: logId })
      });

      const data = await res.json();
      if (res.ok) {
        Swal.fire({
          title: '<span class="text-indigo-400 font-bold flex items-center justify-center gap-2"><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/></svg>AI Analysis (RCA)</span>',
          html: `<pre style="white-space: pre-wrap; font-family: inherit; text-align: left; font-size: 14px; color: #cbd5e1; max-height: 60vh; overflow-y: auto;">${data.reason}</pre>`,
          width: 800,
          background: '#0f172a',
          confirmButtonColor: '#6366f1'
        });
      } else {
        systemAlert.fire('Failed', data.error || 'Failed to analyze incident', 'error');
      }
    } catch (err) {
      console.error(err);
      systemAlert.fire('Error', 'Network error occurred while contacting AI.', 'error');
    } finally {
      isAnalyzingRCA = false;
    }
  }

  function toggleAllSelection() {
    if (allSelected) {
      selectedApiIds = [];
    } else {
      selectedApiIds = apis.map((a) => a.id);
    }
  }

  function toggleSelection(id: string) {
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
      response_script: "",
      recovery_script: "",
    };
    headerMode = "kv";
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
      response_script: api.response_script || "",
      recovery_script: api.recovery_script || "",
    };

    // Parse into KV states
    headersKV = parseToKVArray(apiForm.headers);
    bodyKV = parseToKVArray(apiForm.body);
    paramsKV = parseToKVArray(apiForm.parameters);

    headerMode = headersKV.length > 0 ? "kv" : "json";
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

  let isBulkSchedule = false;

  function openScheduleModal(api: any) {
    isBulkSchedule = false;
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

  function openBulkScheduleModal() {
    isBulkSchedule = true;
    scheduleConfig = {
      mode: "Minute timer",
      value: 1,
      day: "Every day",
      time: "4:00 PM",
    };
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

      if (isBulkSchedule) {
        systemToast.fire({
          icon: "info",
          title: "Processing",
          text: `Applying schedule to ${selectedApiIds.length} APIs...`,
        });

        const promises = selectedApiIds.map((id) => {
          const api = apis.find(a => a.id === id);
          if (!api) return Promise.resolve(null);
          return fetch(`${API_BASE_URL}/api/v1/apis/${id}`, {
            method: "PUT",
            headers: {
              Authorization: `Bearer ${token}`,
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              project_id: projectId,
              folder: api.folder,
              name: api.name,
              method: api.method,
              url: api.url,
              headers: api.headers || "[]",
              body: api.body || "{}",
              parameters: api.parameters || "[]",
              expected_status_code: api.expected_status_code,
              interval: intervalSeconds,
              schedule_config: JSON.stringify(scheduleConfig),
              response_script: api.response_script || "",
              recovery_script: api.recovery_script || "",
            }),
          }).then(res => {
            if (res.ok) {
              return fetch(`${API_BASE_URL}/api/v1/apis/${id}/pause`, {
                method: "POST",
                headers: {
                  Authorization: `Bearer ${token}`,
                  "Content-Type": "application/json",
                },
                body: JSON.stringify({ pause_hours: 0 }),
              });
            }
            return res;
          });
        });

        await Promise.all(promises);
        
        systemToast.fire({
          icon: "success",
          title: "Schedules Updated",
          text: "All selected APIs have been scheduled and resumed.",
        });
        
        isBulkSchedule = false;
        selectedApiIds = [];
        await fetchProjectDetails();

      } else {
        const res = await fetch(
          `${API_BASE_URL}/api/v1/apis/${selectedApi.id}`,
          {
            method: "PUT",
            headers: {
              Authorization: `Bearer ${token}`,
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              project_id: projectId,
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
              response_script: selectedApi.response_script || "",
              recovery_script: selectedApi.recovery_script || "",
            }),
          },
        );
        
        if (res.ok) {
          // Unpause
          await fetch(`${API_BASE_URL}/api/v1/apis/${selectedApi.id}/pause`, {
            method: "POST",
            headers: {
              Authorization: `Bearer ${token}`,
              "Content-Type": "application/json",
            },
            body: JSON.stringify({ pause_hours: 0 }),
          });
          
          await fetchProjectDetails();
        }
      }
    } catch (err) {
      console.error(err);
      systemAlert.fire({
        icon: "error",
        title: "Error",
        text: "Failed to apply schedule.",
      });
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
            project_id: projectId,
            folder: apiForm.folder,
            name: apiForm.name,
            method: apiForm.method,
            url: apiForm.url,
            parameters: apiForm.parameters,
            headers: apiForm.headers,
            body: apiForm.body,
            expected_status_code: apiForm.expected_status_code,
            interval: apiForm.interval,
            response_script: apiForm.response_script,
            recovery_script: apiForm.recovery_script,
          }),
        },
      );

      if (await handleAuthError(res, "edit this API")) return;

      if (res.ok) {
        await fetchProjectDetails();
        systemToast.fire({
          icon: "success",
          title: "Saved",
          text: "API endpoint updated successfully!",
        });
      } else {
        const data = await res.json().catch(() => ({}));
        systemAlert.fire({ icon: "error", title: "Error", text: data.error || "Failed to save changes." });
      }
    } catch (err) {
      console.error(err);
      systemAlert.fire({ icon: "error", title: "Network Error", text: "Could not reach the server." });
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

      if (await handleAuthError(res, "delete this API")) return;

      if (res.ok) {
        apis = apis.filter((a) => a.id !== selectedApi.id);
        systemToast.fire({ icon: "success", title: "Deleted", text: "API endpoint removed." });
        await fetchProjectDetails();
      } else {
        const data = await res.json().catch(() => ({}));
        systemAlert.fire({ icon: "error", title: "Error", text: data.error || "Failed to delete API." });
      }
    } catch (err) {
      console.error(err);
      systemAlert.fire({ icon: "error", title: "Network Error", text: "Could not reach the server." });
    }
  }

  function getStatusInfo(api: any, currentTime: Date) {
    let statusObj: any = { status: 'LIVE', detail: 'Ready for next check', latency: 0, tls: null, securityScore: 0 };
    
    // Evaluate Logs for dynamic real-time data
    if (api.logs && api.logs.length > 0) {
      const latestLog = api.logs[0];
      statusObj.status = latestLog.is_success ? 'ONLINE' : 'DOWN';
      statusObj.latency = latestLog.response_time;
      statusObj.detail = latestLog.is_success ? 'Healthy' : 'Failing';

      if (latestLog.tls_status) {
         try { statusObj.tls = JSON.parse(latestLog.tls_status); } catch(e){}
      }
      if (latestLog.security_headers) {
         try {
           const headers = JSON.parse(latestLog.security_headers);
           let score = 0;
           Object.values(headers).forEach(v => { if (v === 'Present') score += 25; });
           statusObj.securityScore = score;
         } catch(e){}
      }
    }

    if (!api.paused_until) return statusObj;
    const pausedUntil = new Date(api.paused_until);
    if (pausedUntil <= currentTime) return statusObj;

    statusObj.status = 'PAUSED';
    if (pausedUntil.getFullYear() > 9000) {
      statusObj.detail = 'Indefinite';
      return statusObj;
    }

    const diffMs = pausedUntil.getTime() - currentTime.getTime();
    const diffSecs = Math.floor(diffMs / 1000);
    const diffMins = Math.round(diffMs / 60000);
    
    if (diffMins < 60) {
      if (diffSecs < 60) {
        statusObj.detail = `${diffSecs}s remaining`;
      } else {
        statusObj.detail = `${diffMins}m remaining`;
      }
    } else {
      statusObj.detail = `until ${pausedUntil.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
    }
    
    return statusObj;
  }

  function openPauseApiModal(api: any) {
    selectedApi = api;
    
    if (api.paused_until && new Date(api.paused_until) > new Date()) {
      const pausedUntil = new Date(api.paused_until);
      const now = new Date();
      const diffMs = pausedUntil.getTime() - now.getTime();
      const diffMins = Math.round(diffMs / 60000);
      
      if (pausedUntil.getFullYear() > 9000) {
        pauseType = 'indefinite';
        pauseMinutes = 60;
      } else {
        pauseType = 'duration';
        pauseMinutes = diffMins > 0 ? diffMins : 1;
      }
    } else {
      pauseMinutes = 60;
      pauseType = 'duration';
    }
    
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

      if (await handleAuthError(res, "pause/resume this API")) return;

      if (res.ok) {
        await fetchProjectDetails();
        systemToast.fire({
          icon: "success",
          title: pause_hours !== 0 ? "Monitor Paused" : "Monitor Resumed",
          text: label,
        });
      } else {
        const data = await res.json().catch(() => ({}));
        systemAlert.fire({ icon: "error", title: "Error", text: data.error || "Failed to update pause status." });
      }
    } catch (err) {
      console.error(err);
      systemAlert.fire({ icon: "error", title: "Network Error", text: "Could not reach the server." });
    }
  }

  async function handleBulkDeleteSubmit() {
    try {
      const token = localStorage.getItem("monitor_token");
      const deletePromises = selectedApiIds.map((id) =>
        fetch(`${API_BASE_URL}/api/v1/apis/${id}`, {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
        }),
      );

      const results = await Promise.all(deletePromises);

      // Check if any returned 401 (expired token)
      const hasUnauth = results.find((r) => r.status === 401);
      if (hasUnauth) {
        await handleAuthError(hasUnauth, "delete APIs");
        return;
      }

      // Check if any returned 403 (permission)
      const hasForbidden = results.find((r) => r.status === 403);
      if (hasForbidden) {
        await handleAuthError(hasForbidden, "delete APIs");
        return;
      }

      const allOk = results.every((res) => res.ok);
      const someOk = results.some((res) => res.ok);

      if (allOk || someOk) {
        showBulkDeleteModal = false;
        selectedApiIds = [];
        await fetchProjectDetails();
        systemToast.fire({ icon: "success", title: "Deleted", text: allOk ? "All selected APIs removed." : "Some APIs were removed." });
      } else {
        systemAlert.fire({ icon: "error", title: "Error", text: "Failed to delete the selected APIs." });
      }
    } catch (err) {
      console.error("Bulk delete failed:", err);
      systemAlert.fire({ icon: "error", title: "Network Error", text: "Could not reach the server." });
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
            project_id: projectId,
            folder: apiForm.folder,
            name: apiForm.name,
            method: apiForm.method,
            url: apiForm.url,
            parameters: apiForm.parameters,
            headers: apiForm.headers,
            body: apiForm.body,
            expected_status_code: apiForm.expected_status_code,
            interval: apiForm.interval,
            response_script: apiForm.response_script,
            recovery_script: apiForm.recovery_script,
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
            company_id: project.company_id,
            cover_position: project.cover_position,
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
              href={backUrl}
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
            onclick={openMembersModal}
            class="bg-blue-950/30 border border-blue-500/40 text-blue-400 hover:bg-blue-900/50 hover:border-blue-400 font-bold py-2 px-4 rounded-lg shadow-[0_0_10px_rgba(59,130,246,0.15)] transition-all flex items-center gap-2 text-sm w-full md:w-auto justify-center font-mono tracking-wide"
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
              ><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle
                cx="9"
                cy="7"
                r="4"
              ></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path
                d="M16 3.13a4 4 0 0 1 0 7.75"
              ></path></svg
            >
            MEMBERS
          </button>
          <button
            onclick={openEnvVarsModal}
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
            onclick={() => (showFolderModal = true)}
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
            onclick={openAddApiModal}
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
              onchange={handleFileSelect}
              disabled={isUploading}
            />
          </label>
        </div>
      </div>
    </div>

    <!-- API List Header -->
    <div class="mb-6 flex flex-col md:flex-row md:items-center justify-between gap-4 mt-8">
      <div class="flex flex-col md:flex-row md:items-center gap-4 w-full md:w-auto">
        <h2
          class="text-xl md:text-2xl font-bold text-cyan-50 font-mono tracking-wide whitespace-nowrap"
        >
          MONITORED_ENDPOINTS
        </h2>
        <div class="flex items-center gap-3">
          <span
            class="bg-cyan-900 border border-cyan-500/50 text-cyan-300 text-xs font-bold px-3 py-1 rounded-md shadow-[0_0_10px_rgba(6,182,212,0.2)] font-mono tracking-wider w-fit"
            >TOTAL: {filteredApisForDisplay.length}</span
          >
          <div class="relative group/search flex-1 md:w-72">
            <input 
              type="text" 
              bind:value={apiSearchQuery}
              placeholder="ค้นหา API..." 
              class="w-full bg-slate-900/50 border border-slate-700/50 hover:border-cyan-500/50 focus:border-cyan-500 rounded-xl px-10 py-2 text-xs text-cyan-400 font-mono outline-none transition-all placeholder:text-slate-600 shadow-inner"
            />
            <div class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-600 group-hover/search:text-cyan-500 transition-colors">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
            </div>
          </div>
        </div>
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
            onclick={openBulkScheduleModal}
            class="bg-indigo-500/10 text-indigo-400 hover:bg-indigo-500/20 hover:text-indigo-300 border border-indigo-500/30 font-bold py-1.5 px-4 rounded-lg shadow-[0_0_10px_rgba(99,102,241,0.1)] transition-colors text-sm flex items-center gap-2 font-mono tracking-wide"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path><path d="M13.73 21a2 2 0 0 1-3.46 0"></path></svg>
            Set Schedule
          </button>
          <button
            onclick={() => (showBulkDeleteModal = true)}
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
                  onchange={toggleAllSelection}
                  class="w-4 h-4 text-cyan-500 bg-slate-900 border border-slate-600 rounded focus:ring-cyan-500/50 focus:ring-offset-slate-900 cursor-pointer appearance-none checked:bg-cyan-500 checked:border-cyan-500 transition-colors shadow-[0_0_10px_rgba(6,182,212,0.2)]"
                />
              </th>
              <th class="p-3 md:p-4">Method</th>
              <th class="p-3 md:p-4">Endpoint Name</th>
              <th class="p-3 md:p-4">URL</th>
              <th class="p-3 md:p-4">Expected</th>
              <th class="p-3 md:p-4">Status</th>
              <th class="p-3 md:p-4">Security</th>
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
                ondragover={(e) => handleDragOver(e, folderName, -1)}
                ondragleave={handleDragLeave}
                ondrop={(e) => handleDrop(e, folderName, -1)}
              >
                <td colspan="8" class="px-4 py-2">
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
                          onclick={() => openEditFolder(folderName)}
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
                          onclick={() => openDeleteFolder(folderName)}
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
                {@const info = getStatusInfo(api, now)}
                <tr
                  draggable="true"
                  ondragstart={(e) => handleDragStart(e, api.id)}
                  ondragover={(e) => handleDragOver(e, folderName, i)}
                  ondragleave={handleDragLeave}
                  ondrop={(e) => handleDrop(e, folderName, i)}
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
                      onchange={() => toggleSelection(api.id)}
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
                    <span class="font-bold">{api.name}</span>
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
                  <td class="p-3 md:p-4">
                    <div class="flex flex-col gap-1">
                      <div class="flex items-center gap-2">
                        <span
                          class="px-2 py-0.5 rounded border text-[9px] font-bold w-fit tracking-wider shadow-[0_0_5px_rgba(0,0,0,0.2)] flex items-center gap-1
                          {info.status === 'ONLINE'
                            ? 'bg-emerald-950/40 border-emerald-500/30 text-emerald-400'
                            : info.status === 'DOWN' 
                              ? 'bg-red-950/40 border-red-500/30 text-red-400'
                              : info.status === 'PAUSED'
                                ? 'bg-amber-950/40 border-amber-500/30 text-amber-400'
                                : 'bg-slate-800/40 border-slate-600/30 text-slate-400'}"
                        >
                          {#if info.status === 'ONLINE'}
                            <span class="relative flex h-2 w-2 mr-0.5">
                              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                              <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                            </span>
                          {/if}
                          {info.status}
                        </span>
                        
                        {#if info.latency > 0}
                          <span class="text-[10px] font-mono text-cyan-400/80 tracking-tighter">
                            {info.latency}ms
                          </span>
                        {/if}
                      </div>

                      {#if info.detail}
                        <span class="text-[10px] text-slate-500 font-medium italic lowercase">
                          {info.detail}
                        </span>
                      {/if}
                    </div>
                  </td>
                  
                  <td class="p-3 md:p-4">
                    <div class="flex flex-col gap-1 items-start">
                      {#if info.securityScore > 0}
                        <div class="flex items-center gap-1.5" title="Security Score">
                          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-indigo-400"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg>
                          <span class="text-[10px] font-bold {info.securityScore >= 75 ? 'text-emerald-400' : 'text-amber-400'}">{info.securityScore}/100</span>
                        </div>
                      {/if}
                      {#if info.tls}
                        <div class="flex items-center gap-1.5" title="SSL Certificate ({info.tls.issuer})">
                          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-cyan-400"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
                          <span class="text-[10px] font-medium {info.tls.valid ? 'text-cyan-300/80' : 'text-red-400'}">SSL {info.tls.valid ? 'Active' : 'Expired'}</span>
                        </div>
                      {/if}
                      {#if !info.securityScore && !info.tls}
                        <span class="text-[10px] text-slate-600 font-mono italic">N/A</span>
                      {/if}
                    </div>
                  </td>
                  <td
                    class="p-3 md:p-4 text-right flex items-center justify-end gap-1 sm:gap-2"
                  >
                    {#if info.status === 'DOWN' && api.logs && api.logs.length > 0}
                      <button
                        onclick={() => analyzeIncident(api.logs[0].id)}
                        class="text-indigo-400 hover:text-indigo-300 transition-colors p-1.5 rounded-lg hover:bg-slate-900 border border-transparent hover:border-indigo-500/30 hover:shadow-[0_0_10px_rgba(99,102,241,0.2)] animate-pulse"
                        title="Ask AI to Analyze Root Cause"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/></svg>
                      </button>
                    {/if}
                    <button
                      onclick={() => openEditApiModal(api)}
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
                      onclick={() => openScheduleModal(api)}
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
                      onclick={() => openPauseApiModal(api)}
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
                      onclick={() => openDeleteApiModal(api)}
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
                  ondragover={(e) =>
                    handleDragOver(e, folderName, folderApis.length)}
                  ondragleave={handleDragLeave}
                  ondrop={(e) => handleDrop(e, folderName, folderApis.length)}
                >
                  <td colspan="7" class="p-0"></td>
                </tr>
              {/if}
            {/each}
          </tbody>
        </table>
      </div>

      {#if totalApiPages > 1}
        <div class="mt-8 flex justify-center items-center gap-4 animate-fade-in mb-12">
          <button 
            onclick={() => { apiPage = Math.max(1, apiPage - 1); window.scrollTo({ top: 400, behavior: 'smooth' }); }}
            disabled={apiPage === 1}
            class="px-4 py-2 bg-slate-900/50 border border-slate-700 text-slate-400 hover:text-cyan-400 hover:border-cyan-500/30 rounded-xl font-mono text-xs disabled:opacity-50 transition-all flex items-center gap-2"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"></polyline></svg>
            ก่อนหน้า
          </button>
          
          <div class="flex items-center gap-2 text-white">
            <span class="text-xs font-mono text-slate-500 uppercase tracking-widest">หน้า <span class="text-cyan-400">{apiPage}</span> จาก {totalApiPages}</span>
          </div>

          <button 
            onclick={() => { apiPage = Math.min(totalApiPages, apiPage + 1); window.scrollTo({ top: 400, behavior: 'smooth' }); }}
            disabled={apiPage === totalApiPages}
            class="px-4 py-2 bg-slate-900/50 border border-slate-700 text-slate-400 hover:text-cyan-400 hover:border-cyan-500/30 rounded-xl font-mono text-xs disabled:opacity-50 transition-all flex items-center gap-2"
          >
            ถัดไป
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
          </button>
        </div>
      {/if}
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
        onclick={() => executePostmanUpload("append")}
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
        onclick={() => executePostmanUpload("replace")}
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
        onclick={() => executeAddApi("append")}
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
        onclick={() => executeAddApi("replace")}
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
        <span class="text-xs mt-1 text-red-500/80"
          >Clear project and use only this API.</span
        >
      </button>
    </div>
  </div>
</Modal>

<!-- 3. Add API Manual Input Modal -->
<Modal
  bind:open={showAddApiModal}
  title="Create API Endpoint"
  size="full"
>
  <form onsubmit={(e) => { e.preventDefault(); handleAddApiSubmit(); }} class="flex flex-col h-full bg-slate-900/10">
    <div class="flex-1 overflow-y-auto px-1 pt-2 space-y-6 custom-scrollbar pb-10">
      <!-- Section 1: Basic Information -->
      <div class="bg-slate-800/20 rounded-2xl p-6 border border-slate-700/30">
        <h4 class="text-xs font-bold text-cyan-500/80 uppercase tracking-widest mb-4 flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
          Basic Configuration
        </h4>
        <div class="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div class="md:col-span-1">
            <label for="api_folder" class="block text-xs font-bold text-slate-400 uppercase mb-2">Folder (Optional)</label>
            <input id="api_folder" type="text" bind:value={apiForm.folder} class="w-full px-4 py-2.5 bg-slate-950 border border-slate-700/50 rounded-xl focus:ring-2 focus:ring-cyan-500/30 outline-none text-sm font-medium transition-all" placeholder="e.g. Authentication" />
          </div>
          <div class="md:col-span-2">
            <label for="api_name" class="block text-xs font-bold text-slate-400 uppercase mb-2">Endpoint Name</label>
            <input id="api_name" type="text" bind:value={apiForm.name} required class="w-full px-4 py-2.5 bg-slate-950 border border-slate-700/50 rounded-xl focus:ring-2 focus:ring-cyan-500/30 outline-none text-sm font-medium transition-all" placeholder="e.g. Fetch User Data" />
          </div>
          <div class="md:col-span-1">
            <label for="api_method" class="block text-xs font-bold text-slate-400 uppercase mb-2">HTTP Method</label>
            <select id="api_method" bind:value={apiForm.method} class="w-full px-4 py-2.5 bg-slate-950 border border-slate-700/50 rounded-xl focus:ring-2 focus:ring-cyan-500/30 outline-none text-sm font-bold text-cyan-400 uppercase transition-all cursor-pointer">
              <optgroup label="Standard Methods" class="bg-slate-950">
                <option value="GET">GET</option>
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="PATCH">PATCH</option>
                <option value="DELETE">DELETE</option>
              </optgroup>
            </select>
          </div>
          <div class="md:col-span-4">
            <label for="api_url" class="block text-xs font-bold text-slate-400 uppercase mb-2">Target URL</label>
            <div class="flex group">
              <span class="inline-flex items-center px-4 rounded-l-xl border border-r-0 border-slate-700/50 bg-slate-950 text-cyan-500 text-xs font-bold tracking-widest transition-colors group-focus-within:border-cyan-500/30 group-focus-within:bg-cyan-950/20">URL</span>
              <div class="flex-1 w-full bg-slate-950 rounded-r-xl border border-slate-700/50 focus-within:ring-2 focus-within:ring-cyan-500/30 transition-all text-sm overflow-hidden h-[42px] relative">
                <InputWithVariables bind:value={apiForm.url} placeholder="&#123;&#123;base_url&#125;&#125;/api/v1/users" required={true} variables={envVarDict} onpaste={handleUrlPaste} />
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Section 2: Parameters, Headers, Body -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <!-- Left Column: Parameters & Headers -->
        <div class="space-y-6">
          <div class="bg-slate-800/20 rounded-2xl p-6 border border-slate-700/30">
            <div class="flex items-center justify-between mb-4">
              <h4 class="text-xs font-bold text-cyan-500/80 uppercase tracking-widest flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" y1="21" x2="4" y2="14"></line><line x1="4" y1="10" x2="4" y2="3"></line><line x1="12" y1="21" x2="12" y2="12"></line><line x1="12" y1="8" x2="12" y2="3"></line><line x1="20" y1="21" x2="20" y2="16"></line><line x1="20" y1="12" x2="20" y2="3"></line><line x1="1" y1="14" x2="7" y2="14"></line><line x1="9" y1="8" x2="15" y2="8"></line><line x1="17" y1="16" x2="23" y2="16"></line></svg>
                Query Parameters
              </h4>
              <div class="flex bg-slate-950 p-1 rounded-xl border border-slate-800 shadow-inner">
                <button type="button" class="px-3 py-1 text-[10px] uppercase font-bold rounded-lg transition-all {paramMode === 'json' ? 'bg-cyan-600 text-white shadow-lg shadow-cyan-500/20' : 'text-slate-500 hover:text-cyan-400'}" onclick={() => toggleParamMode("json")}>JSON</button>
                <button type="button" class="px-3 py-1 text-[10px] uppercase font-bold rounded-lg transition-all {paramMode === 'kv' ? 'bg-cyan-600 text-white shadow-lg shadow-cyan-500/20' : 'text-slate-500 hover:text-cyan-400'}" onclick={() => toggleParamMode("kv")}>Key-Value</button>
              </div>
            </div>
            {#if paramMode === "json"}
              <TextareaWithVariables rows={3} bind:value={apiForm.parameters} variables={envVarDict} placeholder={`[\n  {"key": "search", "value": "keyword"}\n]`} />
            {:else}
              <div class="space-y-2 max-h-[200px] overflow-y-auto custom-scrollbar pr-2">
                {#each paramsKV as param, i}
                  <div class="flex gap-2 group/kv">
                    <input type="text" bind:value={param.key} placeholder="Key" class="w-1/3 px-3 py-2 bg-slate-950 rounded-lg border border-slate-800 focus:border-cyan-500/50 outline-none text-xs font-mono" />
                    <div class="flex-1 bg-slate-950 rounded-lg border border-slate-800 focus-within:border-cyan-500/50 text-xs font-mono h-[34px] overflow-hidden relative">
                      <InputWithVariables bind:value={param.value} placeholder="Value" variables={envVarDict} />
                    </div>
                    <button type="button" onclick={() => (paramsKV = paramsKV.filter((_, idx) => idx !== i))} class="p-2 text-slate-600 hover:text-red-400 bg-slate-950 border border-slate-800 rounded-lg hover:border-red-500/30 transition-all opacity-40 group-hover/kv:opacity-100">
                      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                    </button>
                  </div>
                {/each}
                <button type="button" onclick={() => (paramsKV = [...paramsKV, { key: "", value: "" }])} class="text-[10px] font-bold text-cyan-500 hover:underline flex items-center gap-1.5 mt-2 transition-all">
                  <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                  ADD PARAMETER
                </button>
              </div>
            {/if}
          </div>

          <div class="bg-slate-800/20 rounded-2xl p-6 border border-slate-700/30">
            <div class="flex items-center justify-between mb-4">
              <h4 class="text-xs font-bold text-cyan-500/80 uppercase tracking-widest flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L2 7l10 5 10-5-10-5z"></path><path d="M2 17l10 5 10-5"></path><path d="M2 12l10 5 10-5"></path></svg>
                Headers
              </h4>
              <div class="flex bg-slate-950 p-1 rounded-xl border border-slate-800 shadow-inner">
                <button type="button" class="px-3 py-1 text-[10px] uppercase font-bold rounded-lg transition-all {headerMode === 'json' ? 'bg-cyan-600 text-white shadow-lg shadow-cyan-500/20' : 'text-slate-500 hover:text-cyan-400'}" onclick={() => toggleHeaderMode("json")}>JSON</button>
                <button type="button" class="px-3 py-1 text-[10px] uppercase font-bold rounded-lg transition-all {headerMode === 'kv' ? 'bg-cyan-600 text-white shadow-lg shadow-cyan-500/20' : 'text-slate-500 hover:text-cyan-400'}" onclick={() => toggleHeaderMode("kv")}>Key-Value</button>
              </div>
            </div>
            {#if headerMode === "json"}
              <TextareaWithVariables rows={3} bind:value={apiForm.headers} variables={envVarDict} placeholder={`[\n  {"key": "Authorization", "value": "Bearer token"}\n]`} />
            {:else}
              <div class="space-y-2 max-h-[200px] overflow-y-auto custom-scrollbar pr-2">
                {#each headersKV as hdr, i}
                  <div class="flex gap-2 group/kv items-center">
                    <input type="text" bind:value={hdr.key} placeholder="Header Name" class="w-[40%] px-3 py-2.5 bg-slate-950 rounded-lg border border-slate-800 focus:border-cyan-500/50 outline-none text-sm font-mono" />
                    <div class="flex-1 bg-slate-950 rounded-lg border border-slate-800 focus-within:border-cyan-500/50 text-sm font-mono h-[40px] overflow-hidden relative">
                      <InputWithVariables bind:value={hdr.value} placeholder="Value" variables={envVarDict} />
                    </div>
                    <button type="button" onclick={() => (headersKV = headersKV.filter((_, idx) => idx !== i))} class="p-2.5 text-slate-600 hover:text-red-400 bg-slate-950 border border-slate-800 rounded-lg hover:border-red-500/30 transition-all opacity-40 group-hover/kv:opacity-100">
                      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                    </button>
                  </div>
                {/each}
                <button type="button" onclick={() => (headersKV = [...headersKV, { key: "", value: "" }])} class="text-[10px] font-bold text-cyan-500 hover:underline flex items-center gap-1.5 mt-2 transition-all">
                  <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                  ADD HEADER
                </button>
              </div>
            {/if}
          </div>
        </div>

        <!-- Right Column: Body & Expected Status -->
        <div class="space-y-6">
          <div class="bg-slate-800/20 rounded-2xl p-6 border border-slate-700/30 {apiForm.method === 'GET' ? 'opacity-40 grayscale pointer-events-none' : ''}">
            <div class="flex items-center justify-between mb-4">
              <h4 class="text-xs font-bold text-cyan-500/80 uppercase tracking-widest flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path></svg>
                Request Body
              </h4>
              <div class="flex bg-slate-950 p-1 rounded-xl border border-slate-800 shadow-inner">
                <button type="button" class="px-3 py-1 text-[10px] uppercase font-bold rounded-lg transition-all {bodyMode === 'json' ? 'bg-cyan-600 text-white shadow-lg shadow-cyan-500/20' : 'text-slate-500 hover:text-cyan-400'}" onclick={() => toggleBodyMode("json")}>JSON</button>
                <button type="button" class="px-3 py-1 text-[10px] uppercase font-bold rounded-lg transition-all {bodyMode === 'kv' ? 'bg-cyan-600 text-white shadow-lg shadow-cyan-500/20' : 'text-slate-500 hover:text-cyan-400'}" onclick={() => toggleBodyMode("kv")}>Key-Value</button>
              </div>
            </div>
            {#if bodyMode === "json"}
              <TextareaWithVariables rows={7} bind:value={apiForm.body} variables={envVarDict} disabled={apiForm.method === "GET"} placeholder={`{ "key": "value" }`} />
            {:else}
              <div class="space-y-2 max-h-[300px] overflow-y-auto custom-scrollbar pr-2">
                {#each bodyKV as bdy, i}
                  <div class="flex gap-2 group/kv">
                    <input type="text" bind:value={bdy.key} disabled={apiForm.method === 'GET'} placeholder="Key" class="w-1/3 px-3 py-2 bg-slate-950 rounded-lg border border-slate-800 focus:border-cyan-500/50 outline-none text-xs font-mono" />
                    <div class="flex-1 bg-slate-950 rounded-lg border border-slate-800 focus-within:border-cyan-500/50 text-xs font-mono h-[34px] overflow-hidden relative">
                      <InputWithVariables bind:value={bdy.value} disabled={apiForm.method === 'GET'} placeholder="Value" variables={envVarDict} />
                    </div>
                    <button type="button" onclick={() => (bodyKV = bodyKV.filter((_, idx) => idx !== i))} class="p-2 text-slate-600 hover:text-red-400 bg-slate-950 border border-slate-800 rounded-lg hover:border-red-500/30 transition-all opacity-40 group-hover/kv:opacity-100">
                      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                    </button>
                  </div>
                {/each}
                <button type="button" onclick={() => (bodyKV = [...bodyKV, { key: "", value: "" }])} class="text-[10px] font-bold text-cyan-500 hover:underline flex items-center gap-1.5 mt-2 transition-all">
                  <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                  ADD BODY FIELD
                </button>
              </div>
            {/if}
            {#if apiForm.method === 'GET'}
              <p class="text-[10px] text-amber-500/80 mt-2 italic font-medium">Body is not supported for GET requests</p>
            {/if}
          </div>

          <div class="grid grid-cols-2 gap-4 bg-slate-800/20 rounded-2xl p-6 border border-slate-700/30">
            <div>
              <label for="api_expected_status" class="block text-xs font-bold text-slate-400 uppercase mb-2">Expected Status</label>
              <input id="api_expected_status" type="number" bind:value={apiForm.expected_status_code} required class="w-full px-4 py-2 bg-slate-950 border border-slate-700/50 rounded-xl focus:ring-2 focus:ring-emerald-500/30 outline-none text-sm font-bold text-emerald-400" />
            </div>
            <div>
              <label for="api_interval" class="block text-xs font-bold text-slate-400 uppercase mb-2">Interval (SEC)</label>
              <input id="api_interval" type="number" bind:value={apiForm.interval} required class="w-full px-4 py-2 bg-slate-950 border border-slate-700/50 rounded-xl focus:ring-2 focus:ring-indigo-500/30 outline-none text-sm font-bold text-indigo-400" />
            </div>
          </div>
        </div>
      </div>

      <!-- Section 3: Response Extraction Script (JS) -->
      <div class="bg-slate-800/30 rounded-2xl p-6 border border-cyan-500/20 shadow-[0_0_20px_rgba(6,182,212,0.05)]">
        <div class="flex items-center justify-between mb-4">
          <h4 class="text-xs font-bold text-cyan-400 uppercase tracking-widest flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>
            Post-Response Script (JavaScript)
          </h4>
          <span class="text-[10px] bg-cyan-950/40 text-cyan-500 px-3 py-1 rounded-full border border-cyan-500/30 font-bold">BETA</span>
        </div>
        <p class="text-[10px] text-slate-500 mb-3 italic">
          Use JavaScript to extract data from the response and update Environment Variables. 
          Use <code>pm.environment.set("key", "value")</code> or <code>setEnv("key", "value")</code>.
          The <code>response</code> object is available with <code>body</code>, <code>status</code>, and <code>headers</code>.
        </p>
        <!-- Response Script Template Dropdown -->
        <div class="flex items-center gap-2 mb-3">
          <span class="text-[10px] text-slate-500 uppercase tracking-widest font-mono shrink-0">📋 ใส่ Template:</span>
          <div class="relative flex-1">
            <select
              onchange={applyResponseTemplate}
              class="w-full appearance-none bg-slate-900 border border-slate-700/60 hover:border-cyan-500/50 text-slate-300 text-xs font-mono rounded-lg px-3 py-1.5 pr-7 outline-none transition-all cursor-pointer focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/20"
            >
              {#each responseScriptTemplates as tpl}
                <option value={tpl.value} class="bg-slate-900">{tpl.label}</option>
              {/each}
            </select>
            <div class="pointer-events-none absolute inset-y-0 right-2 flex items-center">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-slate-500"><polyline points="6 9 12 15 18 9"></polyline></svg>
            </div>
          </div>
        </div>
        <textarea 
          bind:value={apiForm.response_script} 
          rows={6}
          spellcheck={false}
          class="w-full font-mono text-xs px-4 py-3 bg-slate-950 rounded-xl border border-slate-800 focus:border-cyan-500/50 outline-none resize-none transition-all placeholder:text-slate-700 leading-relaxed shadow-inner mb-6"
          placeholder={`// Example: Extract token from JSON\nconst data = JSON.parse(response.body);\nif (data.token) {\n    pm.environment.set("AUTH_TOKEN", data.token);\n    console.log("Token updated!");\n}`}
        ></textarea>

        <div class="flex items-center justify-between mb-4">
          <h4 class="text-xs font-bold text-amber-400 uppercase tracking-widest flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>
            Auto-Recovery / Self-Healing Script
            <!-- Info Icon with Thai Tooltip (Tailwind inline) -->
            <span class="relative inline-flex items-center group/tip ml-1">
              <span class="inline-flex items-center justify-center w-[18px] h-[18px] rounded-full bg-amber-500/15 border border-amber-500/40 text-amber-400 cursor-help transition-all duration-200 group-hover/tip:bg-amber-500/30 group-hover/tip:border-amber-500/70 group-hover/tip:shadow-[0_0_8px_rgba(245,158,11,0.4)] group-hover/tip:scale-110">
                <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
              </span>
              <!-- Tooltip panel -->
              <span class="pointer-events-none invisible opacity-0 group-hover/tip:visible group-hover/tip:opacity-100 group-hover/tip:pointer-events-auto absolute bottom-[calc(100%+10px)] left-1/2 -translate-x-1/2 translate-y-1 group-hover/tip:translate-y-0 w-[320px] bg-slate-950/97 backdrop-blur-md border border-amber-500/30 rounded-xl p-[14px_16px] text-[11px] leading-[1.7] text-slate-300 font-normal normal-case tracking-normal shadow-[0_20px_40px_rgba(0,0,0,0.5),0_0_0_1px_rgba(245,158,11,0.1),inset_0_1px_0_rgba(255,255,255,0.05)] transition-all duration-200 z-[9999]" role="tooltip">
                <strong class="text-amber-400 font-bold">📌 คืออะไร?</strong><br/>
                สคริปต์นี้จะทำงาน<em class="text-orange-400 italic">เฉพาะเมื่อ API เกิดข้อผิดพลาด</em> (เช่น Timeout หรือได้ Status Code ที่ไม่ถูกต้อง) โดยจะรันก่อนที่ระบบจะลองเรียก API ซ้ำ (Retry)<br/><br/>
                <strong class="text-amber-400 font-bold">🔧 ใช้ทำอะไร?</strong><br/>
                ใช้เพื่อ &quot;ซ่อมแซม&quot; สถานการณ์ก่อน Retry เช่น ต่ออายุ Token ที่หมดอายุ<br/><br/>
                <strong class="text-amber-400 font-bold">💡 ตัวแปรที่ใช้ได้:</strong><br/>
                • <code class="bg-amber-500/15 border border-amber-500/25 rounded px-1 py-px font-mono text-[10px] text-yellow-300">errorReason</code> — สาเหตุที่ API พัง<br/>
                • <code class="bg-amber-500/15 border border-amber-500/25 rounded px-1 py-px font-mono text-[10px] text-yellow-300">setEnv(&quot;KEY&quot;, &quot;VAL&quot;)</code> — อัปเดต Environment Variable<br/>
                • <code class="bg-amber-500/15 border border-amber-500/25 rounded px-1 py-px font-mono text-[10px] text-yellow-300">fetch(...)</code> — เรียก API อื่นเพื่อขอ Token ใหม่<br/><br/>
                <strong class="text-amber-400 font-bold">📝 ตัวอย่าง:</strong><br/>
                ถ้า API พังเพราะ Token หมดอายุ ให้เขียนโค้ด fetch ไปขอ Token ใหม่ แล้วใช้ setEnv เก็บไว้ ระบบจะใช้ Token ใหม่ในการ Retry อัตโนมัติ
                <!-- Arrow -->
                <span class="absolute top-full left-1/2 -translate-x-1/2 border-[6px] border-transparent border-t-amber-500/30"></span>
              </span>
            </span>
          </h4>
          <span class="text-[10px] bg-amber-950/40 text-amber-500 px-3 py-1 rounded-full border border-amber-500/30 font-bold">EXPERIMENTAL</span>
        </div>
        <p class="text-[10px] text-slate-500 mb-3 italic">
          If the API fails (timeout or invalid status), this script runs before retrying. Use logic like <code>fetch</code> to refresh a token and <code>setEnv("KEY", "VAL")</code> to store it. The error message is available as <code>errorReason</code>.
        </p>
        <!-- Recovery Script Template Dropdown -->
        <div class="flex items-center gap-2 mb-3">
          <span class="text-[10px] text-slate-500 uppercase tracking-widest font-mono shrink-0">📋 ใส่ Template:</span>
          <div class="relative flex-1">
            <select
              onchange={applyRecoveryTemplate}
              class="w-full appearance-none bg-slate-900 border border-slate-700/60 hover:border-amber-500/50 text-slate-300 text-xs font-mono rounded-lg px-3 py-1.5 pr-7 outline-none transition-all cursor-pointer focus:border-amber-500/50 focus:ring-1 focus:ring-amber-500/20"
            >
              {#each recoveryScriptTemplates as tpl}
                <option value={tpl.value} class="bg-slate-900">{tpl.label}</option>
              {/each}
            </select>
            <div class="pointer-events-none absolute inset-y-0 right-2 flex items-center">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-amber-600"><polyline points="6 9 12 15 18 9"></polyline></svg>
            </div>
          </div>
        </div>
        <textarea 
          bind:value={apiForm.recovery_script} 
          rows={6}
          spellcheck={false}
          class="w-full font-mono text-xs px-4 py-3 bg-slate-950 rounded-xl border border-slate-800 focus:border-amber-500/50 outline-none resize-none transition-all placeholder:text-slate-700 leading-relaxed shadow-inner"
          placeholder={`// Example: Refresh token logic\nconsole.log("API Failed because: " + errorReason);\n// perform a fetch call to refresh token...`}
        ></textarea>
      </div>
    </div>

    <!-- Modal Footer -->
    <div class="flex justify-end gap-3 p-6 bg-slate-900 border-t border-slate-700/50 sticky bottom-0 z-10">
      <button type="button" onclick={() => (showAddApiModal = false)} class="px-6 py-2.5 text-xs font-bold uppercase tracking-wider text-slate-400 hover:text-white transition-colors">Cancel</button>
      <button type="submit" class="px-8 py-2.5 bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white text-xs font-bold uppercase tracking-wider rounded-xl transition-all shadow-lg shadow-cyan-900/40 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path><polyline points="17 21 17 13 7 13 7 21"></polyline><polyline points="7 3 7 8 15 8"></polyline></svg>
        Save Endpoint
      </button>
    </div>
  </form>
</Modal>

<!-- 4. Edit API Modal -->
<Modal
  bind:open={showEditApiModal}
  title="Edit API Endpoint"
  size="full"
>
  <form onsubmit={(e) => { e.preventDefault(); handleEditApiSubmit(); }} class="flex flex-col h-full bg-slate-900/10">
    <div class="flex-1 overflow-y-auto px-1 pt-2 space-y-6 custom-scrollbar pb-24">
      <!-- Section 1: Basic Information -->
      <div class="bg-slate-800/20 rounded-2xl p-6 border border-slate-700/30">
        <h4 class="text-xs font-bold text-cyan-500/80 uppercase tracking-widest mb-4 flex items-center gap-2">
          Basic Configuration (Editing)
        </h4>
        <div class="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div class="md:col-span-1">
            <label for="api_folder_edit" class="block text-xs font-bold text-slate-400 uppercase mb-2">Folder</label>
            <input id="api_folder_edit" type="text" bind:value={apiForm.folder} class="w-full px-4 py-2.5 bg-slate-950 border border-slate-700/50 rounded-xl outline-none text-sm font-medium" />
          </div>
          <div class="md:col-span-2">
            <label for="api_name_edit" class="block text-xs font-bold text-slate-400 uppercase mb-2">Name</label>
            <input id="api_name_edit" type="text" bind:value={apiForm.name} required class="w-full px-4 py-2.5 bg-slate-950 border border-slate-700/50 rounded-xl outline-none text-sm font-medium" />
          </div>
          <div class="md:col-span-1">
            <label for="api_method_edit" class="block text-xs font-bold text-slate-400 uppercase mb-2">Method</label>
            <select id="api_method_edit" bind:value={apiForm.method} class="w-full px-4 py-2.5 bg-slate-950 border border-slate-700/50 rounded-xl outline-none text-sm font-bold text-cyan-400 uppercase">
              <optgroup label="Standard" class="bg-slate-950">
                <option value="GET">GET</option><option value="POST">POST</option><option value="PUT">PUT</option><option value="PATCH">PATCH</option><option value="DELETE">DELETE</option>
              </optgroup>
            </select>
          </div>
          <div class="md:col-span-4">
            <label for="api_url_edit" class="block text-xs font-bold text-slate-400 uppercase mb-2">URL</label>
            <div class="flex group bg-slate-950 rounded-xl border border-slate-700/50 focus-within:ring-2 focus-within:ring-cyan-500/30 text-sm overflow-hidden h-[42px] relative">
              <InputWithVariables bind:value={apiForm.url} variables={envVarDict} onpaste={handleUrlPaste} />
            </div>
          </div>
        </div>
      </div>

      <!-- Section 2: Params, Headers, Body -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div class="space-y-6">
          <div class="bg-slate-800/20 rounded-2xl p-6 border border-slate-700/30">
            <div class="flex items-center justify-between mb-4">
              <h4 class="text-xs font-bold text-cyan-500/80 uppercase tracking-widest">Query Parameters</h4>
              <div class="flex bg-slate-950 p-1 rounded-xl">
                <button type="button" class="px-3 py-1 text-[10px] font-bold rounded-lg {paramMode === 'json' ? 'bg-cyan-600 text-white' : 'text-slate-500'}" onclick={() => toggleParamMode("json")}>JSON</button>
                <button type="button" class="px-3 py-1 text-[10px] font-bold rounded-lg {paramMode === 'kv' ? 'bg-cyan-600 text-white' : 'text-slate-500'}" onclick={() => toggleParamMode("kv")}>KV</button>
              </div>
            </div>
            {#if paramMode === "json"}
              <TextareaWithVariables rows={3} bind:value={apiForm.parameters} variables={envVarDict} />
            {:else}
              <div class="space-y-2 max-h-[200px] overflow-y-auto">
                {#each paramsKV as param, i}
                  <div class="flex gap-2">
                    <input type="text" bind:value={param.key} class="w-1/3 px-3 py-2 bg-slate-950 rounded-lg border border-slate-800 text-xs font-mono" />
                    <div class="flex-1 bg-slate-950 rounded-lg border border-slate-800 text-xs font-mono h-[34px] overflow-hidden relative">
                      <InputWithVariables bind:value={param.value} variables={envVarDict} />
                    </div>
                    <button type="button" onclick={() => (paramsKV = paramsKV.filter((_, idx) => idx !== i))} class="p-2 text-slate-600 hover:text-red-400"><svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg></button>
                  </div>
                {/each}
                <button type="button" onclick={() => (paramsKV = [...paramsKV, { key: "", value: "" }])} class="text-[10px] font-bold text-cyan-500 mt-2">+ ADD PARAM</button>
              </div>
            {/if}
          </div>

          <div class="bg-slate-800/20 rounded-2xl p-6 border border-slate-700/30">
            <div class="flex items-center justify-between mb-4">
              <h4 class="text-xs font-bold text-cyan-500/80 uppercase tracking-widest flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L2 7l10 5 10-5-10-5z"></path><path d="M2 17l10 5 10-5"></path><path d="M2 12l10 5 10-5"></path></svg>
                Headers
              </h4>
              <div class="flex bg-slate-950 p-1 rounded-xl border border-slate-800 shadow-inner">
                <button type="button" class="px-3 py-1 text-[10px] uppercase font-bold rounded-lg transition-all {headerMode === 'json' ? 'bg-cyan-600 text-white shadow-lg shadow-cyan-500/20' : 'text-slate-500 hover:text-cyan-400'}" onclick={() => toggleHeaderMode("json")}>JSON</button>
                <button type="button" class="px-3 py-1 text-[10px] uppercase font-bold rounded-lg transition-all {headerMode === 'kv' ? 'bg-cyan-600 text-white shadow-lg shadow-cyan-500/20' : 'text-slate-500 hover:text-cyan-400'}" onclick={() => toggleHeaderMode("kv")}>Key-Value</button>
              </div>
            </div>
            {#if headerMode === "json"}
              <TextareaWithVariables rows={3} bind:value={apiForm.headers} variables={envVarDict} />
            {:else}
              <div class="space-y-2 max-h-[200px] overflow-y-auto custom-scrollbar pr-2">
                {#each headersKV as hdr, i}
                  <div class="flex gap-2 group/kv items-center">
                    <input type="text" bind:value={hdr.key} placeholder="Header Name" class="w-[40%] px-3 py-2.5 bg-slate-950 rounded-lg border border-slate-800 focus:border-cyan-500/50 outline-none text-sm font-mono" />
                    <div class="flex-1 bg-slate-950 rounded-lg border border-slate-800 focus-within:border-cyan-500/50 text-sm font-mono h-[40px] overflow-hidden relative">
                      <InputWithVariables bind:value={hdr.value} variables={envVarDict} />
                    </div>
                    <button type="button" onclick={() => (headersKV = headersKV.filter((_, idx) => idx !== i))} class="p-2.5 text-slate-600 hover:text-red-400 bg-slate-950 border border-slate-800 rounded-lg hover:border-red-500/30 transition-all opacity-40 group-hover/kv:opacity-100"><svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg></button>
                  </div>
                {/each}
                <button type="button" onclick={() => (headersKV = [...headersKV, { key: "", value: "" }])} class="text-[10px] font-bold text-cyan-500 hover:underline flex items-center gap-1.5 mt-2 transition-all">
                  <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                  ADD HEADER
                </button>
              </div>
            {/if}
          </div>
        </div>

        <div class="space-y-6">
          <div class="bg-slate-800/20 rounded-2xl p-6 border border-slate-700/30 {apiForm.method === 'GET' ? 'opacity-40 grayscale pointer-events-none' : ''}">
            <h4 class="text-xs font-bold text-cyan-500/80 uppercase tracking-widest mb-4">Body</h4>
            {#if bodyMode === "json"}
              <TextareaWithVariables rows={7} bind:value={apiForm.body} variables={envVarDict} disabled={apiForm.method === "GET"} />
            {:else}
              <div class="space-y-2 max-h-[300px] overflow-y-auto">
                {#each bodyKV as bdy, i}
                  <div class="flex gap-2">
                    <input type="text" bind:value={bdy.key} class="w-1/3 px-3 py-2 bg-slate-950 rounded-lg border border-slate-800 text-xs font-mono" />
                    <div class="flex-1 bg-slate-950 rounded-lg border border-slate-800 text-xs font-mono h-[34px] overflow-hidden relative"><InputWithVariables bind:value={bdy.value} variables={envVarDict} /></div>
                    <button type="button" onclick={() => (bodyKV = bodyKV.filter((_, idx) => idx !== i))} class="p-2 text-slate-600 hover:text-red-400"><svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg></button>
                  </div>
                {/each}
                <button type="button" onclick={() => (bodyKV = [...bodyKV, { key: "", value: "" }])} class="text-[10px] font-bold text-cyan-500 mt-2">+ ADD FIELD</button>
              </div>
            {/if}
          </div>

          <div class="grid grid-cols-2 gap-4 bg-slate-800/20 rounded-2xl p-6 border border-slate-700/30">
            <div>
              <label for="api_expected_status_edit" class="block text-xs font-bold text-slate-400 uppercase mb-2">Status</label>
              <input id="api_expected_status_edit" type="number" bind:value={apiForm.expected_status_code} class="w-full px-4 py-2 bg-slate-950 border border-slate-700 rounded-xl text-emerald-400 font-bold" />
            </div>
            <div>
              <label for="api_interval_edit" class="block text-xs font-bold text-slate-400 uppercase mb-2">Interval</label>
              <input id="api_interval_edit" type="number" bind:value={apiForm.interval} class="w-full px-4 py-2 bg-slate-950 border border-slate-700 rounded-xl text-indigo-400 font-bold" />
            </div>
          </div>
        </div>
      </div>

      <!-- Section 3: Scripts (Post-Response + Auto-Recovery) -->
      <div class="bg-slate-800/30 rounded-2xl p-6 border border-cyan-500/20 shadow-[0_0_20px_rgba(6,182,212,0.05)]">
        <h4 class="text-xs font-bold text-cyan-400 uppercase tracking-widest mb-3">Post-Response Script (JavaScript)</h4>
        <!-- Response Script Template Dropdown (Edit Modal) -->
        <div class="flex items-center gap-2 mb-3">
          <span class="text-[10px] text-slate-500 uppercase tracking-widest font-mono shrink-0">📋 ใส่ Template:</span>
          <div class="relative flex-1">
            <select
              onchange={applyResponseTemplate}
              class="w-full appearance-none bg-slate-900 border border-slate-700/60 hover:border-cyan-500/50 text-slate-300 text-xs font-mono rounded-lg px-3 py-1.5 pr-7 outline-none transition-all cursor-pointer focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/20"
            >
              {#each responseScriptTemplates as tpl}
                <option value={tpl.value} class="bg-slate-900">{tpl.label}</option>
              {/each}
            </select>
            <div class="pointer-events-none absolute inset-y-0 right-2 flex items-center">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-slate-500"><polyline points="6 9 12 15 18 9"></polyline></svg>
            </div>
          </div>
        </div>
        <textarea bind:value={apiForm.response_script} rows={6} spellcheck={false} class="w-full font-mono text-xs px-4 py-3 bg-slate-950 rounded-xl border border-slate-800 focus:border-cyan-500/50 outline-none resize-none transition-all placeholder:text-slate-700 leading-relaxed shadow-inner" placeholder={`pm.environment.set("token", JSON.parse(response.body).token);`}></textarea>

        <div class="flex items-center justify-between mb-4 mt-6">
          <h4 class="text-xs font-bold text-amber-400 uppercase tracking-widest flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>
            Auto-Recovery / Self-Healing Script
            <!-- Info Icon with Thai Tooltip (Tailwind inline) -->
            <span class="relative inline-flex items-center group/tip ml-1">
              <span class="inline-flex items-center justify-center w-[18px] h-[18px] rounded-full bg-amber-500/15 border border-amber-500/40 text-amber-400 cursor-help transition-all duration-200 group-hover/tip:bg-amber-500/30 group-hover/tip:border-amber-500/70 group-hover/tip:shadow-[0_0_8px_rgba(245,158,11,0.4)] group-hover/tip:scale-110">
                <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
              </span>
              <span class="pointer-events-none invisible opacity-0 group-hover/tip:visible group-hover/tip:opacity-100 group-hover/tip:pointer-events-auto absolute bottom-[calc(100%+10px)] left-1/2 -translate-x-1/2 translate-y-1 group-hover/tip:translate-y-0 w-[320px] bg-slate-950/97 backdrop-blur-md border border-amber-500/30 rounded-xl p-[14px_16px] text-[11px] leading-[1.7] text-slate-300 font-normal normal-case tracking-normal shadow-[0_20px_40px_rgba(0,0,0,0.5),0_0_0_1px_rgba(245,158,11,0.1),inset_0_1px_0_rgba(255,255,255,0.05)] transition-all duration-200 z-[9999]" role="tooltip">
                <strong class="text-amber-400 font-bold">📌 คืออะไร?</strong><br/>
                สคริปต์นี้จะทำงาน<em class="text-orange-400 italic">เฉพาะเมื่อ API เกิดข้อผิดพลาด</em> (เช่น Timeout หรือได้ Status Code ที่ไม่ถูกต้อง) โดยจะรันก่อนที่ระบบจะลองเรียก API ซ้ำ (Retry)<br/><br/>
                <strong class="text-amber-400 font-bold">🔧 ใช้ทำอะไร?</strong><br/>
                ใช้เพื่อ &quot;ซ่อมแซม&quot; สถานการณ์ก่อน Retry เช่น ต่ออายุ Token ที่หมดอายุ<br/><br/>
                <strong class="text-amber-400 font-bold">💡 ตัวแปรที่ใช้ได้:</strong><br/>
                • <code class="bg-amber-500/15 border border-amber-500/25 rounded px-1 py-px font-mono text-[10px] text-yellow-300">errorReason</code> — สาเหตุที่ API พัง<br/>
                • <code class="bg-amber-500/15 border border-amber-500/25 rounded px-1 py-px font-mono text-[10px] text-yellow-300">setEnv(&quot;KEY&quot;, &quot;VAL&quot;)</code> — อัปเดต Environment Variable<br/>
                • <code class="bg-amber-500/15 border border-amber-500/25 rounded px-1 py-px font-mono text-[10px] text-yellow-300">fetch(...)</code> — เรียก API อื่นเพื่อขอ Token ใหม่<br/><br/>
                <strong class="text-amber-400 font-bold">📝 ตัวอย่าง:</strong><br/>
                ถ้า API พังเพราะ Token หมดอายุ ให้เขียนโค้ด fetch ไปขอ Token ใหม่ แล้วใช้ setEnv เก็บไว้ ระบบจะใช้ Token ใหม่ในการ Retry อัตโนมัติ
                <span class="absolute top-full left-1/2 -translate-x-1/2 border-[6px] border-transparent border-t-amber-500/30"></span>
              </span>
            </span>
          </h4>
          <span class="text-[10px] bg-amber-950/40 text-amber-500 px-3 py-1 rounded-full border border-amber-500/30 font-bold">EXPERIMENTAL</span>
        </div>
        <p class="text-[10px] text-slate-500 mb-3 italic">
          If the API fails (timeout or invalid status), this script runs before retrying. Use logic like <code>fetch</code> to refresh a token and <code>setEnv("KEY", "VAL")</code> to store it. The error message is available as <code>errorReason</code>.
        </p>
        <!-- Recovery Script Template Dropdown (Edit Modal) -->
        <div class="flex items-center gap-2 mb-3">
          <span class="text-[10px] text-slate-500 uppercase tracking-widest font-mono shrink-0">📋 ใส่ Template:</span>
          <div class="relative flex-1">
            <select
              onchange={applyRecoveryTemplate}
              class="w-full appearance-none bg-slate-900 border border-slate-700/60 hover:border-amber-500/50 text-slate-300 text-xs font-mono rounded-lg px-3 py-1.5 pr-7 outline-none transition-all cursor-pointer focus:border-amber-500/50 focus:ring-1 focus:ring-amber-500/20"
            >
              {#each recoveryScriptTemplates as tpl}
                <option value={tpl.value} class="bg-slate-900">{tpl.label}</option>
              {/each}
            </select>
            <div class="pointer-events-none absolute inset-y-0 right-2 flex items-center">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-amber-600"><polyline points="6 9 12 15 18 9"></polyline></svg>
            </div>
          </div>
        </div>
        <textarea
          bind:value={apiForm.recovery_script}
          rows={6}
          spellcheck={false}
          class="w-full font-mono text-xs px-4 py-3 bg-slate-950 rounded-xl border border-slate-800 focus:border-amber-500/50 outline-none resize-none transition-all placeholder:text-slate-700 leading-relaxed shadow-inner"
          placeholder={`// Example: Refresh token logic\nconsole.log("API Failed because: " + errorReason);\n// perform a fetch call to refresh token...`}
        ></textarea>
      </div>
    </div>

    <div class="flex justify-end gap-3 p-6 bg-slate-900 border-t border-slate-700/50 sticky bottom-0 z-10">
      <button type="button" onclick={() => (showEditApiModal = false)} class="px-6 py-2.5 text-xs font-bold uppercase tracking-wider text-slate-500">Cancel</button>
      <button type="submit" class="px-8 py-2.5 bg-gradient-to-r from-emerald-600 to-teal-600 text-white text-xs font-bold uppercase tracking-wider rounded-xl shadow-lg ring-1 ring-emerald-500/50">Save Changes</button>
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
        onclick={() => (showDeleteApiModal = false)}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-slate-500 bg-slate-800 hover:bg-slate-700 transition-colors"
        >Cancel</button
      >
      <button
        onclick={handleDeleteApiSubmit}
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
        onclick={() => (showBulkDeleteModal = false)}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-slate-500 bg-slate-800 hover:bg-slate-700 transition-colors"
        >Cancel</button
      >
      <button
        onclick={handleBulkDeleteSubmit}
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
        onclick={() => (showScheduleModal = false)}
        class="px-4 py-2 text-xs text-cyan-50 bg-slate-900/40 border border-slate-600 rounded-xl hover:bg-slate-900/50 font-medium transition-colors"
        >Cancel</button
      >
      <button
        onclick={handleScheduleSubmit}
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
        onclick={() => (showEditFolderModal = false)}
        class="px-4 py-2 text-xs text-cyan-50 bg-slate-900/40 border border-slate-600 rounded-xl hover:bg-slate-900/50 font-medium transition-colors"
        >Cancel</button
      >
      <button
        onclick={handleEditFolderSubmit}
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
        onclick={() => (showDeleteFolderModal = false)}
        class="px-4 py-2 text-xs rounded-xl font-semibold text-slate-500 bg-slate-800 hover:bg-slate-700 transition-colors"
        >Cancel</button
      >
      <button
        onclick={handleDeleteFolderSubmit}
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
            onclick={() => removeEnvVarRow(i)}
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
        onclick={addEnvVarRow}
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
        onclick={() => (showEnvVarsModal = false)}
        class="px-4 py-2 text-slate-500 bg-slate-800 border border-slate-700 rounded-xl hover:bg-slate-700 font-medium transition-colors text-xs"
        >Cancel</button
      >
      <button
        onclick={saveEnvVars}
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
        onclick={() => (showFolderModal = false)}
        class="px-4 py-2 text-slate-500 bg-slate-800 border border-slate-700 rounded-xl hover:bg-slate-700 font-medium transition-colors text-xs"
        >Cancel</button
      >
      <button
        onclick={handleAddFolder}
        disabled={!newFolderName.trim()}
        class="px-4 py-2 bg-cyan-600 text-cyan-50 rounded-xl hover:bg-cyan-700 font-medium transition-colors shadow-[0_0_15px_rgba(6,182,212,0.3)] text-xs disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Create Folder
      </button>
    </div>
  </div>
</Modal>

<!-- Pause API Modal -->
<Modal bind:open={showPauseApiModal} title="Pause Monitor ({selectedApi?.name})" maxWidth="max-w-md">
  <div class="space-y-4">

    <!-- Type selector tabs -->
    <div class="flex gap-2 bg-slate-900/60 border border-slate-700/50 rounded-xl p-1">
      <button
        type="button"
        onclick={() => (pauseType = 'duration')}
        class="flex-1 px-3 py-1.5 text-xs font-semibold rounded-lg transition-all {pauseType === 'duration' ? 'bg-amber-600 text-white shadow' : 'text-slate-400 hover:text-cyan-50'}"
      >Pause for duration</button>
      <button
        type="button"
        onclick={() => (pauseType = 'indefinite')}
        class="flex-1 px-3 py-1.5 text-xs font-semibold rounded-lg transition-all {pauseType === 'indefinite' ? 'bg-amber-600 text-white shadow' : 'text-slate-400 hover:text-cyan-50'}"
      >Indefinite</button>
      <button
        type="button"
        onclick={() => (pauseType = 'resume')}
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
              onclick={() => (pauseMinutes = min)}
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
        onclick={() => (showPauseApiModal = false)}
        class="px-4 py-2 text-xs text-cyan-50 bg-slate-900/40 border border-slate-600 rounded-xl hover:bg-slate-900/50 font-medium transition-colors"
      >
        Cancel
      </button>
      <button
        type="button"
        onclick={handlePauseApiSubmit}
        class="px-4 py-2 text-xs font-semibold text-white rounded-xl transition-colors shadow-sm {pauseType === 'resume' ? 'bg-emerald-600 hover:bg-emerald-700' : 'bg-amber-600 hover:bg-amber-700'}"
      >
        {pauseType === 'resume' ? 'Resume Monitor' : 'Confirm Pause'}
      </button>
    </div>
  </div>
</Modal>

<!-- Manage Project Members Modal -->
<Modal bind:open={showMembersModal} title="PROJECT_MEMBERS" maxWidth="max-w-xl">
  <div class="space-y-6 px-1">
    <!-- Add Member Section -->
    <div
      class="p-4 bg-slate-900/50 border border-slate-700/50 rounded-xl space-y-3"
    >
      <label
        class="block text-xs font-mono font-bold text-cyan-500/70 uppercase tracking-widest"
      >
        Add Company Member to Project
      </label>
      <div class="flex gap-3">
        <select
          bind:value={selectedMemberId}
          class="flex-1 bg-slate-800 border border-slate-700 text-cyan-50 rounded-lg px-3 py-2 focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500 outline-none transition-all font-mono text-sm"
        >
          <option value={null}>Select a member...</option>
          {#each companyMembers as cm}
            {#if !projectMembers.some((pm) => pm.user_id === cm.user_id)}
              <option value={cm.user_id}>{cm.user.name} ({cm.user.email})</option>
            {/if}
          {/each}
        </select>
        <button
          onclick={addProjectMember}
          disabled={!selectedMemberId || isAddingMember}
          class="bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed text-white px-4 py-2 rounded-lg font-bold transition-all flex items-center gap-2 whitespace-nowrap"
        >
          {#if isAddingMember}
            <div
              class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"
            ></div>
          {/if}
          ADD_MEMBER
        </button>
      </div>
    </div>

    <!-- Members List -->
    <div class="space-y-3">
      <h4
        class="text-xs font-mono font-bold text-slate-500 uppercase tracking-widest border-b border-slate-800 pb-2 flex justify-between items-center"
      >
        <span>Current Members</span>
        <span class="text-cyan-500/50">{projectMembers.length} ACTIVE</span>
      </h4>

      <div class="space-y-2 max-h-[300px] overflow-y-auto pr-2 custom-scrollbar">
        {#each projectMembers as pm}
          <div
            class="flex items-center justify-between p-3 bg-slate-800/30 border border-slate-700/30 rounded-xl hover:bg-slate-800/50 transition-all group"
          >
            <div class="flex items-center gap-3">
              <div
                class="w-10 h-10 rounded-full bg-slate-700 flex items-center justify-center overflow-hidden border border-slate-600 group-hover:border-cyan-500/50 transition-colors"
              >
                {#if pm.user?.profile_image_url}
                  <img
                    src={pm.user.profile_image_url.startsWith("http") ||
                    pm.user.profile_image_url.startsWith("data:")
                      ? pm.user.profile_image_url
                      : `${API_BASE_URL}${pm.user.profile_image_url}`}
                    alt={pm.user.name}
                    class="w-full h-full object-cover"
                  />
                {:else}
                  <span class="text-slate-400 font-bold"
                    >{pm.user?.name?.charAt(0) || '?'}</span
                  >
                {/if}
              </div>
              <div class="min-w-0">
                <p class="text-sm font-bold text-cyan-50 truncate">
                  {pm.user?.name || pm.user?.email || "Unnamed"}
                </p>
                <p class="text-xs text-slate-400 truncate">{pm.user?.email}</p>
              </div>
            </div>

            <button
              onclick={() => removeProjectMember(pm.user_id)}
              class="flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-bold text-red-400 bg-red-400/5 border border-red-400/20 rounded-lg hover:bg-red-400/10 transition-all flex items-center justify-center font-mono uppercase tracking-widest shrink-0"
              title="Remove from project"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><line x1="18" y1="6" x2="6" y2="18" /><line
                  x1="6"
                  y1="6"
                  x2="18"
                  y2="18"
                /></svg
              >
              REMOVE
            </button>
          </div>
        {:else}
          <div class="text-center py-8 text-slate-500 font-mono text-sm italic">
            No additional members in this project.
          </div>
        {/each}
      </div>
    </div>
  </div>
</Modal>

<style>
  .custom-scrollbar::-webkit-scrollbar {
    width: 6px;
  }
  .custom-scrollbar::-webkit-scrollbar-track {
    background: rgba(15, 23, 42, 0.3);
    border-radius: 10px;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb {
    background: rgba(71, 85, 105, 0.5);
    border-radius: 10px;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb:hover {
    background: rgba(100, 116, 139, 0.5);
  }

  /* ===== Recovery Info Tooltip ===== */
  .recovery-info-tooltip-wrapper {
    position: relative;
    display: inline-flex;
    align-items: center;
  }

  .recovery-info-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: rgba(245, 158, 11, 0.15);
    border: 1px solid rgba(245, 158, 11, 0.4);
    color: #f59e0b;
    cursor: help;
    transition: all 0.2s ease;
    flex-shrink: 0;
  }

  .recovery-info-icon:hover {
    background: rgba(245, 158, 11, 0.3);
    border-color: rgba(245, 158, 11, 0.7);
    box-shadow: 0 0 8px rgba(245, 158, 11, 0.4);
    transform: scale(1.1);
  }

  .recovery-tooltip {
    visibility: hidden;
    opacity: 0;
    pointer-events: none;
    position: absolute;
    bottom: calc(100% + 10px);
    left: 50%;
    transform: translateX(-50%) translateY(4px);
    width: 320px;
    background: rgba(15, 23, 42, 0.97);
    backdrop-filter: blur(12px);
    border: 1px solid rgba(245, 158, 11, 0.3);
    border-radius: 12px;
    padding: 14px 16px;
    font-size: 11px;
    line-height: 1.7;
    color: #cbd5e1;
    font-weight: 400;
    font-family: inherit;
    text-transform: none;
    letter-spacing: normal;
    box-shadow:
      0 20px 40px rgba(0, 0, 0, 0.5),
      0 0 0 1px rgba(245, 158, 11, 0.1),
      inset 0 1px 0 rgba(255, 255, 255, 0.05);
    transition: opacity 0.2s ease, transform 0.2s ease, visibility 0.2s ease;
    z-index: 9999;
  }

  /* Tooltip arrow */
  .recovery-tooltip::after {
    content: '';
    position: absolute;
    top: 100%;
    left: 50%;
    transform: translateX(-50%);
    border: 6px solid transparent;
    border-top-color: rgba(245, 158, 11, 0.3);
  }

  .recovery-info-tooltip-wrapper:hover .recovery-tooltip {
    visibility: visible;
    opacity: 1;
    transform: translateX(-50%) translateY(0);
    pointer-events: auto;
  }

  .recovery-tooltip strong {
    color: #fbbf24;
    font-weight: 700;
  }

  .recovery-tooltip em {
    color: #fb923c;
    font-style: italic;
  }

  .recovery-tooltip code {
    background: rgba(245, 158, 11, 0.15);
    border: 1px solid rgba(245, 158, 11, 0.25);
    border-radius: 4px;
    padding: 1px 5px;
    font-family: 'Fira Code', 'Cascadia Code', monospace;
    font-size: 10px;
    color: #fcd34d;
  }
</style>

