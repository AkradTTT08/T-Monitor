<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import Modal from "$lib/components/Modal.svelte";
  import { API_BASE_URL } from "$lib/config";
  import Swal from "sweetalert2";
  import { systemAlert, systemToast } from "$lib/swal-design";

  let projects: any[] = [];
  let selectedProjectId = "";
  let selectedCompanyId = ""; // Level 1: must select a company first

  $: currentPath = $page.url.pathname;

  // Listen to page changes to auto-select the right project/company ID
  $: {
    if ($page.params.id) {
      // Company route: /dashboard/companies/[id]
      if (currentPath.startsWith("/dashboard/companies/")) {
        selectedCompanyId = $page.params.id;
        if (typeof window !== "undefined") {
          localStorage.setItem("monitor_selected_company", selectedCompanyId);
        }
      } else {
        // Project route: /dashboard/projects/[id]
        selectedProjectId = $page.params.id;
        if (typeof window !== "undefined") {
          localStorage.setItem("monitor_selected_project", selectedProjectId);
        }
      }
    } else if (currentPath === "/dashboard/companies") {
      selectedCompanyId = "";
      selectedProjectId = "";
      if (typeof window !== "undefined") {
        localStorage.removeItem("monitor_selected_company");
        localStorage.removeItem("monitor_selected_project");
      }
    }
  }

  $: selectedProject = projects.find(
    (p) => p.id.toString() === selectedProjectId,
  );
  $: apiCount = selectedProject?.apis?.length || 0;

  // Derive the most accurate company ID: from URL param, localStorage, or the selected project's company
  $: effectiveCompanyId = (() => {
    // If we're on a project route, the project's company context should lead
    if (selectedProject?.company_id) return selectedProject.company_id.toString();
    // Fallback to explicitly selected company
    if (selectedCompanyId && selectedCompanyId !== "undefined") return selectedCompanyId;
    return "";
  })();

  // Keep selectedCompanyId in sync when a project is active
  $: if (selectedProject?.company_id) {
    const cid = selectedProject.company_id.toString();
    if (selectedCompanyId !== cid) {
      selectedCompanyId = cid;
      if (typeof window !== "undefined") {
        localStorage.setItem("monitor_selected_company", cid);
      }
    }
  }

  $: filteredProjects = effectiveCompanyId
    ? projects.filter((p) => p.company_id?.toString() === effectiveCompanyId)
    : [];

  function handleProjectChange(e: Event) {
    const target = e.target as HTMLSelectElement;
    const newId = target.value;
    if (newId) {
      selectedProjectId = newId;
      localStorage.setItem("monitor_selected_project", newId);
      goto(`/dashboard/projects/${newId}`);
    } else {
      goto("/dashboard");
    }
  }

  let user: any = null;
  let isMobileMenuOpen = false;
  let isSidebarCollapsed = false;
  let showHelpModal = false;

  let showProfileMenu = false;

  function toggleProfileMenu(e: Event) {
    e.stopPropagation();
    showProfileMenu = !showProfileMenu;
  }

  // Session Management State
  let lastActivityTimestamp = Date.now();
  let sessionInterval: any;
  const IDLE_TIMEOUT_MS = 60 * 60 * 1000; // 1 hour
  const TOKEN_REFRESH_THRESHOLD_MS = 30 * 60 * 1000; // 30 minutes
  let lastTokenRefresh = Date.now();

  // Notification State
  let unreadNotifications: any[] = [];
  let showNotificationCenter = false;
  $: unreadCount = unreadNotifications.length;

  function updateActivity() {
    lastActivityTimestamp = Date.now();
  }

  // Dashboard Notification Polling
  async function checkNotifications() {
    try {
      const token = localStorage.getItem("monitor_token");
      if (!token) return;
      
      const res = await fetch(`${API_BASE_URL}/api/v1/notifications/unread`, {
        headers: { Authorization: `Bearer ${token}` }
      });
      
      if (res.ok) {
        const data = await res.json();
        
        // Find truly new notifications to show Toast
        const newOnes = data.filter((n: any) => !unreadNotifications.find((un: any) => un.id === n.id));
        
        for (const note of newOnes) {
          systemToast.fire({
            icon: note.type === 'api_fail' ? 'error' : 'info',
            title: note.title,
            text: note.message,
            timer: 8000,
          });
        }
        
        unreadNotifications = data;
      }
    } catch (err) {
      // Silent error for polling
    }
  }

  async function markAllNotificationsRead() {
    try {
      const token = localStorage.getItem("monitor_token");
      for (const note of unreadNotifications) {
        await fetch(`${API_BASE_URL}/api/v1/notifications/${note.id}/read`, {
          method: 'PUT',
          headers: { Authorization: `Bearer ${token}` }
        });
      }
      unreadNotifications = [];
      showNotificationCenter = false;
    } catch (err) {
      console.error(err);
    }
  }

  async function handleAcceptInvitation(invitationId: string) {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/companies/invitations/${invitationId}/accept`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.ok) {
        systemToast.fire({ icon: "success", title: "Accepted", text: "You have joined the company successfully." });
        // Refresh data
        const pRes = await fetch(`${API_BASE_URL}/api/v1/projects`, {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (pRes.ok) projects = await pRes.json();
        
        unreadNotifications = unreadNotifications.filter(n => n.invitation_id !== invitationId);
        if (unreadNotifications.length === 0) showNotificationCenter = false;
      } else {
        const data = await res.json();
        systemAlert.fire({ icon: "error", title: "Error", text: data.error || "Failed to accept invitation" });
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function handleDeclineInvitation(invitationId: string) {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/companies/invitations/${invitationId}/decline`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.ok) {
        systemToast.fire({ icon: "info", title: "Declined", text: "Invitation declined." });
        unreadNotifications = unreadNotifications.filter(n => n.invitation_id !== invitationId);
        if (unreadNotifications.length === 0) showNotificationCenter = false;
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function fetchSidebarProjects() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        projects = await res.json();
        if (!selectedProjectId) {
          const saved = localStorage.getItem("monitor_selected_project");
          if (saved) selectedProjectId = saved;
          else if (projects.length > 0)
            selectedProjectId = projects[0].id.toString();
        }
      }
    } catch (err) {
      console.error("Failed to load projects for sidebar", err);
    }
  }

  // Separate mount call for project fetching
  onMount(() => {
    // Restore company selection
    const savedCompany = localStorage.getItem("monitor_selected_company");
    if (savedCompany) selectedCompanyId = savedCompany;

    fetchSidebarProjects();
    window.addEventListener("projects-updated", fetchSidebarProjects);
    
    return () => {
      window.removeEventListener("projects-updated", fetchSidebarProjects);
    };
  });

  // Computed access levels
  $: hasCompany = !!selectedCompanyId && selectedCompanyId !== "undefined";
  $: hasProject = !!selectedProjectId && selectedProjectId !== "undefined";

  function handleRequireCompany(e: Event) {
    if (!hasCompany) {
      e.preventDefault();
      systemAlert.fire({
        icon: 'info',
        title: 'Select a Company First',
        text: 'Please select a company from Company Monitor before accessing Project APIs.',
      });
    }
  }

  function handleRequireProject(e: Event) {
    if (!hasProject) {
      e.preventDefault();
      systemAlert.fire({
        icon: 'info',
        title: 'Select a Project First',
        text: 'Please open a project from Project APIs before accessing this section.',
      });
    }
  }

  // Session management mount (returns cleanup fn)
  onMount(() => {
    const token = localStorage.getItem("monitor_token");
    const userData = localStorage.getItem("monitor_user");

    if (!token || !userData) {
      window.location.href = "/login";
      return;
    }

    user = JSON.parse(userData);

    // Global Fetch Interceptor to catch 401/403 anywhere in the dashboard
    const originalFetch = window.fetch;
    window.fetch = async (...args) => {
      const response = await originalFetch(...args);
      if (response.status === 401 || response.status === 403) {
        console.warn("Global fetch caught unauthorized response. Logging out.");
        handleLogout();
      }
      return response;
    };

    const handleUserUpdate = (e: any) => {
      if (e.detail) {
        user = { ...e.detail };
      } else {
        const userData = localStorage.getItem("monitor_user");
        if (userData) user = { ...JSON.parse(userData) };
      }
    };

    // Setup Activity Listeners
    window.addEventListener("mousemove", updateActivity);
    window.addEventListener("keydown", updateActivity);
    window.addEventListener("click", updateActivity);
    window.addEventListener("scroll", updateActivity);
    window.addEventListener("user-updated", handleUserUpdate);

    // Setup Session Manager Interval (Runs every 5 minutes)
    sessionInterval = setInterval(checkSessionStatus, 5 * 60 * 1000);

    // Setup Notification Polling (Runs every 10 seconds)
    checkNotifications();
    const notificationInterval = setInterval(checkNotifications, 10 * 1000);

    return () => {
      window.fetch = originalFetch; // Restore original fetch
      window.removeEventListener("mousemove", updateActivity);
      window.removeEventListener("keydown", updateActivity);
      window.removeEventListener("click", updateActivity);
      window.removeEventListener("scroll", updateActivity);
      window.removeEventListener("user-updated", handleUserUpdate);
      if (sessionInterval) clearInterval(sessionInterval);
      if (notificationInterval) clearInterval(notificationInterval);
    };
  });

  async function checkSessionStatus() {
    const now = Date.now();

    // Check Idle Timeout (1 Hour)
    if (now - lastActivityTimestamp > IDLE_TIMEOUT_MS) {
      console.warn("Session expired due to inactivity.");
      handleLogout();
      return;
    }

    // Check Token Keep-Alive (Refresh every 30 mins if active)
    if (now - lastTokenRefresh > TOKEN_REFRESH_THRESHOLD_MS) {
      await performTokenRefresh();
    }
  }

  async function performTokenRefresh() {
    try {
      const token = localStorage.getItem("monitor_token");
      if (!token) return;

      const res = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
      });

      if (res.ok) {
        const data = await res.json();
        if (data.token) {
          localStorage.setItem("monitor_token", data.token);
          lastTokenRefresh = Date.now();
          console.log("Session token refreshed successfully.");
        }
      } else if (res.status === 401 || res.status === 403) {
        console.warn("Token refresh rejected by backend. Forcing logout.");
        handleLogout();
      }
    } catch (err) {
      console.error("Failed to refresh token:", err);
    }
  }

  function handleLogout() {
    localStorage.removeItem("monitor_token");
    localStorage.removeItem("monitor_user");
    window.location.href = "/login";
  }

  function handleAlertsClick(e: Event) {
    if (!selectedProjectId || selectedProjectId === "undefined") {
      e.preventDefault();
      systemAlert.fire({
        icon: "info",
        title: "Please Create a Project First",
        text: "Notification channels are configured on a per-project basis. Please create a project and select it from the sidebar to manage alerts.",
      });
    }
  }
</script>

<svelte:window on:click={() => (showProfileMenu = false)} />

{#if user}
  <div
    class="h-screen w-full bg-slate-950 flex overflow-hidden font-mono text-slate-300 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-slate-900 via-slate-950 to-black"
  >
    <!-- Mobile Menu Button -->
    <button
      class="md:hidden fixed top-4 right-4 z-50 p-2 rounded-lg bg-slate-800 shadow-[0_0_15px_rgba(6,182,212,0.3)] border border-slate-700 text-cyan-400"
      on:click={() => (isMobileMenuOpen = !isMobileMenuOpen)}
    >
      {#if isMobileMenuOpen}
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
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
      {:else}
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><line x1="3" y1="12" x2="21" y2="12"></line><line
            x1="3"
            y1="6"
            x2="21"
            y2="6"
          ></line><line x1="3" y1="18" x2="21" y2="18"></line></svg
        >
      {/if}
    </button>

    <!-- Sidebar Overlay for Mobile -->
    {#if isMobileMenuOpen}
      <div
        class="fixed inset-0 bg-slate-900/50 backdrop-blur-sm z-30 md:hidden"
        on:click={() => (isMobileMenuOpen = false)}
      ></div>
    {/if}

    <!-- Sidebar Area (Single-column layout) -->
    <div
      class="fixed md:static inset-y-0 left-0 flex h-full bg-slate-900/80 backdrop-blur-xl z-40 transition-transform duration-300 transform {isMobileMenuOpen
        ? 'translate-x-0'
        : '-translate-x-full md:translate-x-0'} border-r border-slate-800 relative group/sidebar overflow-visible"
    >
      <!-- Floating sidebar toggle button on right border -->
      <button
        on:click={() => (isSidebarCollapsed = !isSidebarCollapsed)}
        title={isSidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
        class="absolute right-0 translate-x-1/2 top-[40px] z-50
          w-6 h-6 flex items-center justify-center rounded-full
          bg-slate-800 border border-cyan-500/50 shadow-[0_0_10px_rgba(6,182,212,0.2)] text-cyan-500
          hover:text-cyan-300 hover:border-cyan-400 hover:shadow-[0_0_15px_rgba(6,182,212,0.5)]
          opacity-0 hover:opacity-100 group-hover/sidebar:opacity-100
          transition-all duration-300"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="12"
          height="12"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          {#if isSidebarCollapsed}
            <polyline points="9 18 15 12 9 6"></polyline>
          {:else}
            <polyline points="15 18 9 12 15 6"></polyline>
          {/if}
        </svg>
      </button>
      <aside
        class="{isSidebarCollapsed
          ? 'w-[80px] px-2'
          : 'w-[260px] px-4'} transition-all duration-300 h-full flex flex-col shrink-0 relative z-10 py-6"
      >
        <!-- Top Profile/Org block -->
        <div
          class="flex items-center p-2 mb-6 {isSidebarCollapsed
            ? 'justify-center'
            : 'gap-3'}"
        >
          <div
            class="w-10 h-10 flex shrink-0 items-center justify-center relative overflow-hidden"
          >
            <img
              src="/t-monitor-logo.svg"
              alt="T-Monitor Logo"
              class="w-full h-full object-contain relative z-10"
            />
          </div>
          {#if !isSidebarCollapsed}
            <div class="flex flex-col overflow-hidden min-w-0 flex-1">
              <span
                class="font-bold text-cyan-50 text-sm leading-tight truncate tracking-wide"
                >T-Monitor</span
              >
              <span
                class="text-[10px] text-cyan-500/80 truncate font-mono tracking-wider uppercase"
                >{user.role || "UNIT COMMAND"}</span
              >
            </div>
          {/if}
        </div>

        <!-- Main Navigation -->
        <div
          class="flex-1 overflow-y-auto space-y-1.5 {isSidebarCollapsed
            ? 'px-0'
            : 'pr-1'} hide-scrollbar"
        >
          <!-- Global Pulse Link -->
          <a
            href="/dashboard"
            on:click={() => (isMobileMenuOpen = false)}
            title="Global Pulse"
            class="w-full flex items-center group/navitem {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath === '/dashboard'
              ? 'bg-blue-900/30 border border-blue-500/50 text-blue-300 shadow-[0_0_15px_rgba(59,130,246,0.15)]'
              : 'text-slate-400 hover:bg-slate-800/80 hover:text-blue-400 border border-transparent'}"
          >
            <div class="flex items-center gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none"
                class="transition-colors {currentPath === '/dashboard' ? 'text-blue-400' : 'text-slate-500 group-hover/navitem:text-blue-400'}"
                stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
              ><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
              {#if !isSidebarCollapsed}<span>Global Pulse</span>{/if}
            </div>
            {#if !isSidebarCollapsed}
               <span class="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse shadow-[0_0_8px_rgba(59,130,246,0.8)]"></span>
            {/if}
          </a>

          <a
            href="/dashboard/companies"
            on:click={() => (isMobileMenuOpen = false)}
            title="Company Monitor"
            class="w-full flex items-center group/navitem {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath.startsWith(
            '/dashboard/companies',
          )
              ? 'bg-cyan-900/30 border border-cyan-500/50 text-cyan-300 shadow-[0_0_15px_rgba(6,182,212,0.15)]'
              : 'text-slate-400 hover:bg-slate-800/80 hover:text-cyan-400 border border-transparent'}"
          >
            <div class="flex items-center gap-3">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                class="transition-colors {currentPath.startsWith('/dashboard/companies')
                  ? 'text-cyan-400'
                  : 'text-slate-500 group-hover/navitem:text-cyan-400'}"
                stroke="currentColor"
                stroke-width="2.5"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg
              >
              {#if !isSidebarCollapsed}<span>Company Monitor</span>{/if}
            </div>
          </a>

          <!-- Project APIs Link — Level 2: requires Company -->
          <a
            href={effectiveCompanyId ? `/dashboard/companies/${effectiveCompanyId}` : '#'}
            on:click={(e) => { isMobileMenuOpen = false; if (!effectiveCompanyId) { e.preventDefault(); handleRequireCompany(e); } }}
            title="Project APIs"
            class="w-full flex items-center group/navitem {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all
              {!effectiveCompanyId
                ? 'opacity-40 cursor-not-allowed border border-transparent text-slate-500'
                : currentPath.startsWith('/dashboard/projects') || currentPath.startsWith('/dashboard/companies/')
                  ? 'bg-cyan-900/30 border border-cyan-500/50 text-cyan-300 shadow-[0_0_15px_rgba(6,182,212,0.15)]'
                  : 'text-slate-400 hover:bg-slate-800/80 hover:text-cyan-400 border border-transparent'}"
          >
            <div class="flex items-center gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none"
                class="transition-colors {currentPath.startsWith('/dashboard/projects') || currentPath.startsWith('/dashboard/companies/')
                  ? 'text-cyan-400' : 'text-slate-500 group-hover/navitem:text-cyan-400'}"
                stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
              ><rect x="3" y="3" width="7" height="9" rx="1" /><rect x="14" y="3" width="7" height="5" rx="1" /><rect x="14" y="12" width="7" height="9" rx="1" /><rect x="3" y="16" width="7" height="5" rx="1" /></svg>
              {#if !isSidebarCollapsed}<span>Project APIs</span>{/if}
            </div>
            {#if !isSidebarCollapsed && !effectiveCompanyId}
              <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-600 shrink-0">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            {/if}
          </a>

          <!-- Indent marker for Level 3 items -->
          <div class="{isSidebarCollapsed ? '' : 'ml-3 pl-3 border-l border-slate-700/60'} space-y-0.5">

          <!-- Open APIs Link — Level 3: requires Project -->
          <a
            href={hasProject ? `/dashboard/apis?project_id=${selectedProjectId}` : '#'}
            on:click={(e) => { isMobileMenuOpen = false; handleRequireProject(e); }}
            title="Open APIs"
            class="w-full flex items-center group/navitem {isSidebarCollapsed
              ? 'justify-center px-0 relative'
              : 'justify-between px-3'} py-2.5 rounded-xl text-sm font-semibold transition-all
              {!hasProject
                ? 'opacity-40 cursor-not-allowed border border-transparent text-slate-500'
                : currentPath === '/dashboard/apis'
                  ? 'bg-cyan-900/30 border border-cyan-500/50 text-cyan-300 shadow-[0_0_15px_rgba(6,182,212,0.15)]'
                  : 'text-slate-400 hover:bg-slate-800/80 hover:text-cyan-400 border border-transparent'}"
          >
            <div class="flex items-center gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
                class="transition-colors {currentPath === '/dashboard/apis' ? 'text-cyan-400' : 'text-slate-500 group-hover/navitem:text-cyan-400'}"
                stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
              ><polyline points="16 18 22 12 16 6" /><polyline points="8 6 2 12 8 18" /></svg>
              {#if !isSidebarCollapsed}<span>Open APIs</span>{/if}
            </div>
            {#if !isSidebarCollapsed}
              {#if !hasProject}
                <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-600 shrink-0">
                  <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                </svg>
              {:else}
                <span class="bg-cyan-500/20 border border-cyan-500/50 text-cyan-300 text-[10px] font-bold px-2 py-0.5 rounded-md">{apiCount}</span>
              {/if}
            {:else}
              {#if hasProject}
                <span class="absolute top-1.5 right-1.5 w-2.5 h-2.5 bg-cyan-400 border border-slate-900 rounded-full shadow-[0_0_10px_rgba(6,182,212,0.8)]"></span>
              {/if}
            {/if}
          </a>

          <!-- Status Live Link — Level 3: requires Project -->
          <a
            href={hasProject ? `/dashboard/status?project_id=${selectedProjectId}` : '#'}
            on:click={(e) => { isMobileMenuOpen = false; handleRequireProject(e); }}
            title="Status Live"
            class="w-full flex items-center group/navitem {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'justify-between px-3'} py-2.5 rounded-xl text-sm font-semibold transition-all
              {!hasProject
                ? 'opacity-40 cursor-not-allowed border border-transparent text-slate-500'
                : currentPath === '/dashboard/status'
                  ? 'bg-cyan-900/30 border border-cyan-500/50 text-cyan-300 shadow-[0_0_15px_rgba(6,182,212,0.15)]'
                  : 'text-slate-400 hover:bg-slate-800/80 hover:text-cyan-400 border border-transparent'}"
          >
            <div class="flex items-center gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
                class="transition-colors {currentPath === '/dashboard/status' ? 'text-cyan-400' : 'text-slate-500 group-hover/navitem:text-cyan-400'}"
                stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
              ><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
              {#if !isSidebarCollapsed}<span>Status Live</span>{/if}
            </div>
            {#if !isSidebarCollapsed && !hasProject}
              <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-600 shrink-0">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            {/if}
          </a>

          <!-- Analytics Link — Level 3: requires Project -->
          <a
            href={hasProject ? `/dashboard/analytics?project_id=${selectedProjectId}` : '#'}
            on:click={(e) => { isMobileMenuOpen = false; handleRequireProject(e); }}
            title="Analytics"
            class="w-full flex items-center group/navitem {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'justify-between px-3'} py-2.5 rounded-xl text-sm font-semibold transition-all
              {!hasProject
                ? 'opacity-40 cursor-not-allowed border border-transparent text-slate-500'
                : currentPath === '/dashboard/analytics'
                  ? 'bg-indigo-900/30 border border-indigo-500/50 text-indigo-300 shadow-[0_0_15px_rgba(99,102,241,0.15)]'
                  : 'text-slate-400 hover:bg-slate-800/80 hover:text-indigo-400 border border-transparent'}"
          >
            <div class="flex items-center gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
                class="transition-colors {currentPath === '/dashboard/analytics' ? 'text-indigo-400' : 'text-slate-500 group-hover/navitem:text-indigo-400'}"
                stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
              ><path d="M21 12V7H5a2 2 0 0 1 0-4h14v4"/><path d="M3 5v14a2 2 0 0 0 2 2h16v-5"/><path d="M18 12a2 2 0 0 0 0 4h4v-4Z"/></svg>
              {#if !isSidebarCollapsed}<span>Analytics</span>{/if}
            </div>
            {#if !isSidebarCollapsed && !hasProject}
              <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-600 shrink-0">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            {/if}
          </a>

          <!-- Repair API Link — Level 3: requires Project -->
          <a
            href={hasProject ? `/dashboard/projects/${selectedProjectId}/repair` : '#'}
            on:click={(e) => { isMobileMenuOpen = false; handleRequireProject(e); }}
            title="Repair API"
            class="w-full flex items-center group/navitem {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'justify-between px-3'} py-2.5 rounded-xl text-sm font-semibold transition-all
              {!hasProject
                ? 'opacity-40 cursor-not-allowed border border-transparent text-slate-500'
                : currentPath.includes('/repair')
                  ? 'bg-rose-900/30 border border-rose-500/50 text-rose-300 shadow-[0_0_15px_rgba(244,63,94,0.15)]'
                  : 'text-slate-400 hover:bg-slate-800/80 hover:text-rose-400 border border-transparent'}"
          >
            <div class="flex items-center gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
                class="transition-colors {currentPath.includes('/repair') ? 'text-rose-400' : 'text-slate-500 group-hover/navitem:text-rose-400'}"
                stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
              ><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/></svg>
              {#if !isSidebarCollapsed}<span>Repair API</span>{/if}
            </div>
            {#if !isSidebarCollapsed && !hasProject}
              <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-600 shrink-0">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            {/if}
          </a>

          <!-- Notification Channels Link — Level 3: requires Project -->
          <a
            href={hasProject ? `/dashboard/projects/${selectedProjectId}/notifications` : '#'}
            on:click={(e) => { isMobileMenuOpen = false; handleRequireProject(e); }}
            title="Alerts & Channels"
            class="w-full flex items-center group/navitem {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'justify-between px-3'} py-2.5 rounded-xl text-sm font-semibold transition-all
              {!hasProject
                ? 'opacity-40 cursor-not-allowed border border-transparent text-slate-500'
                : currentPath.includes('notifications')
                  ? 'bg-amber-900/30 border border-amber-500/50 text-amber-300 shadow-[0_0_15px_rgba(245,158,11,0.15)]'
                  : 'text-slate-400 hover:bg-slate-800/80 hover:text-amber-400 border border-transparent'}"
          >
            <div class="flex items-center gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
                class="transition-colors {currentPath.includes('notifications') ? 'text-amber-400' : 'text-slate-500 group-hover/navitem:text-amber-400'}"
                stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
              ><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
              {#if !isSidebarCollapsed}<span>Alerts & Channels</span>{/if}
            </div>
            {#if !isSidebarCollapsed && !hasProject}
              <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-600 shrink-0">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            {/if}
          </a>

          </div><!-- end Level 3 indent -->


          {#if user.role === "admin"}
            <div class="pt-4 pb-1">
              {#if !isSidebarCollapsed}
                <div
                  class="px-3 flex items-center justify-between text-[10px] uppercase font-bold text-slate-500 tracking-wider mb-2 font-mono"
                >
                  <span>Administration</span>
                </div>
              {:else}
                <div class="w-full h-px bg-slate-800 mb-2"></div>
              {/if}

              <!-- Manage Users Link -->
              <a
                href="/dashboard/users"
                on:click={() => (isMobileMenuOpen = false)}
                title="Users & Roles"
                class="w-full flex items-center group/navitem {isSidebarCollapsed
                  ? 'justify-center px-0'
                  : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath ===
                '/dashboard/users'
                  ? 'bg-purple-900/30 border border-purple-500/50 text-purple-300 shadow-[0_0_15px_rgba(168,85,247,0.15)]'
                  : 'text-slate-400 hover:bg-slate-800/80 hover:text-purple-400 border border-transparent'}"
              >
                <div class="flex items-center gap-3">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="20"
                    height="20"
                    viewBox="0 0 24 24"
                    fill="none"
                    class="transition-colors {currentPath === '/dashboard/users'
                      ? 'text-purple-400'
                      : 'text-slate-500 group-hover/navitem:text-purple-400'}"
                    stroke="currentColor"
                    stroke-width="2.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    ><path
                      d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"
                    /><circle cx="9" cy="7" r="4" /><path
                      d="M22 21v-2a4 4 0 0 0-3-3.87"
                    /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></svg
                  >
                  {#if !isSidebarCollapsed}<span>Users & Roles</span>{/if}
                </div>
              </a>

              <!-- Projects Section Header -->
              {#if !isSidebarCollapsed}
                <div
                  class="px-3 pt-4 pb-1 text-[10px] uppercase font-bold text-slate-500 tracking-wider font-mono"
                >
                  Projects
                </div>
              {:else}
                <div class="w-full h-px bg-slate-800 mt-3 mb-1"></div>
              {/if}

              <!-- Dynamic Project List -->
              {#if filteredProjects.length > 0}
                <div class="space-y-0.5">
                  {#each filteredProjects as project}
                    {@const isActive =
                      currentPath.startsWith(
                        `/dashboard/projects/${project.id}`,
                      ) ||
                      (!currentPath.startsWith("/dashboard/projects") &&
                        selectedProjectId === project.id.toString())}
                    <a
                      href={`/dashboard/projects/${project.id}`}
                      on:click={() => (isMobileMenuOpen = false)}
                      title={project.name}
                      class="w-full flex items-center group/project {isSidebarCollapsed
                        ? 'justify-center py-2'
                        : 'px-3 py-2'} rounded-xl text-sm font-medium transition-all border {isActive
                        ? 'bg-slate-800 border-cyan-500/30 text-cyan-300 shadow-[0_0_10px_rgba(6,182,212,0.1)]'
                        : 'text-slate-400 border-transparent hover:bg-slate-800/50 hover:text-cyan-400'}"
                    >
                      {#if isSidebarCollapsed}
                        <!-- Collapsed: small circle avatar -->
                        <div
                          class="relative w-7 h-7 rounded-lg flex items-center justify-center text-[11px] font-bold uppercase shrink-0
                            {isActive
                            ? 'bg-cyan-900 border border-cyan-400/50 text-cyan-300'
                            : 'bg-slate-800 text-slate-500 group-hover/project:text-cyan-400'}"
                        >
                          {project.name.charAt(0)}
                        </div>
                      {:else}
                        <!-- Expanded: dot + name + active badge -->
                        <div class="flex items-center gap-2.5 truncate w-full">
                          <div
                            class="w-2 h-2 rounded-full shrink-0 transition-colors {isActive
                              ? 'bg-cyan-400 shadow-[0_0_8px_rgba(6,182,212,0.8)]'
                              : 'bg-slate-600 group-hover/project:bg-cyan-500/50'}"
                          ></div>
                          <span class="truncate text-sm">{project.name}</span>
                          {#if isActive}
                            <span
                              class="ml-auto text-[10px] font-semibold px-1.5 py-0.5 rounded-full bg-cyan-950 border border-cyan-500/40 text-cyan-400 shrink-0"
                              >Active</span
                            >
                          {/if}
                        </div>
                      {/if}
                    </a>
                  {/each}
                </div>
              {:else if !isSidebarCollapsed}
                <p class="px-3 text-xs text-slate-400 italic">
                  No projects yet.
                </p>
              {/if}
            </div>
          {/if}
        </div>

        <!-- Bottom User Section -->
        <div class="mt-auto pt-4 border-t border-slate-800 flex flex-col gap-2">
          <!-- Notifications Bell -->
          <button
            on:click={() => (showNotificationCenter = true)}
            class="w-full flex items-center {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'gap-3 px-3'} py-2 rounded-xl text-sm font-semibold transition-all text-slate-500 hover:bg-slate-800/80 hover:text-rose-400 border border-transparent relative group/bell"
            title="Notifications"
          >
            <div class="relative">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2.5"
                stroke-linecap="round"
                stroke-linejoin="round"
                class="transition-colors group-hover/bell:text-rose-400"
                ><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path><path
                  d="M13.73 21a2 2 0 0 1-3.46 0"
                ></path></svg
              >
              {#if unreadCount > 0}
                <span
                  class="absolute -top-1.5 -right-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-rose-500 text-[10px] font-bold text-white shadow-lg ring-2 ring-slate-900 animate-pulse"
                >
                  {unreadCount}
                </span>
              {/if}
            </div>
            {#if !isSidebarCollapsed}<span>Notifications</span>{/if}
          </button>

          <!-- Help center -->
          <button
            on:click={() => (showHelpModal = true)}
            class="w-full flex items-center {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'gap-3 px-3'} py-2 rounded-xl text-sm font-semibold transition-all text-slate-500 hover:bg-slate-800/80 hover:text-cyan-400 border border-transparent"
            title="Help center"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><circle cx="12" cy="12" r="10"></circle><path
                d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"
              ></path><line x1="12" y1="17" x2="12.01" y2="17"></line></svg
            >
            {#if !isSidebarCollapsed}<span>Help center</span>{/if}
          </button>
        </div>

        <!-- User Info -->
        <div
          class="w-full flex flex-col {isSidebarCollapsed
            ? 'pt-0'
            : 'pt-2'} relative"
        >
          <button
            on:click={toggleProfileMenu}
            title={user?.email || "Profile options"}
            class="w-full flex items-center {isSidebarCollapsed
              ? 'justify-center p-0'
              : 'justify-between p-2'} rounded-xl hover:bg-slate-800 border border-transparent hover:border-slate-700 transition-all text-left group relative"
          >
            <div class="flex items-center gap-3 overflow-hidden min-w-0">
              <div
                class="w-10 h-10 rounded-full bg-slate-900 flex shrink-0 items-center justify-center text-cyan-400 font-bold text-sm shadow-[0_0_10px_rgba(6,182,212,0.2)] uppercase relative border-2 border-slate-700 overflow-hidden"
              >
                {#if user?.profile_image_url}
                  <img
                    src={user.profile_image_url.startsWith('http') || user.profile_image_url.startsWith('data:')
                      ? user.profile_image_url
                      : `${API_BASE_URL}${user.profile_image_url}`}
                    alt="Profile"
                    class="w-full h-full object-cover"
                  />
                  {:else}
                    {user?.email?.charAt(0) || "U"}
                  {/if}
                </div>
                {#if !isSidebarCollapsed}
                  <div
                    class="flex flex-col overflow-hidden min-w-0 flex-1 pr-2"
                  >
                    <span
                      class="font-bold text-slate-200 text-sm leading-tight truncate capitalize"
                      >{user?.name ||
                        user?.email?.split("@")[0] ||
                        "User"}</span
                    >
                    <span
                      class="text-xs text-slate-500 font-medium truncate font-mono"
                      >{user?.email || "user@example.com"}</span
                    >
                  </div>
                {/if}
              </div>
              {#if !isSidebarCollapsed}
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
                  class="text-slate-600 group-hover:text-cyan-500 transition-colors shrink-0"
                  ><polyline points="18 15 12 9 6 15"></polyline></svg
                >
              {/if}
            </button>

            <!-- Profile Popup Menu -->
            {#if showProfileMenu}
              <div
                class="absolute bottom-full left-0 {isSidebarCollapsed
                  ? 'ml-12'
                  : 'w-full'} mb-2 bg-slate-800 rounded-xl shadow-xl border border-slate-700 overflow-hidden z-50 animate-in slide-in-from-bottom-2 duration-200 min-w-[160px]"
              >
                <div class="p-1">
                  <a
                    href="/dashboard/settings/profile"
                    class="w-full text-left px-3 py-2.5 text-sm font-semibold text-slate-300 hover:text-cyan-400 hover:bg-slate-700/50 rounded-lg transition-colors flex items-center gap-2"
                  >
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
                      ><path
                        d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"
                      /><circle cx="12" cy="12" r="3" /></svg
                    >
                    Setting Profile
                  </a>
                  <button
                    on:click={handleLogout}
                    class="w-full text-left px-3 py-2.5 text-sm font-semibold text-red-500 hover:bg-red-500/10 rounded-lg transition-colors flex items-center gap-2"
                  >
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
                      ><path
                        d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"
                      /><polyline points="16 17 21 12 16 7" /><line
                        x1="21"
                        x2="9"
                        y1="12"
                        y2="12"
                      /></svg
                    >
                    Logout
                  </button>
                </div>
              </div>
            {/if}

            <!-- Copyright & Contact Icons -->
            {#if !isSidebarCollapsed}
              <div
                class="px-2 mt-4 animate-in fade-in duration-300 border-t border-slate-800/50 pt-4"
              >
                <p
                  class="text-[9px] font-bold text-slate-500 mb-3 font-mono tracking-tighter uppercase text-center"
                >
                  © 2024 TTT BROTHER CO., LTD.
                </p>
                <div class="flex items-center justify-center gap-3">
                  <a
                    href="https://www.facebook.com/TTTBrother/"
                    target="_blank"
                    title="Facebook"
                    class="group/link no-underline"
                  >
                    <div
                      class="w-8 h-8 rounded-lg flex items-center justify-center bg-slate-950 border border-slate-800 group-hover/link:border-blue-500/50 group-hover/link:bg-blue-500/5 transition-all"
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
                        class="text-slate-500 group-hover/link:text-blue-400"
                        ><path
                          d="M18 2h-3a5 5 0 0 0-5 5v3H7v4h3v8h4v-8h3l1-4h-4V7a1 1 0 0 1 1-1h3z"
                        /></svg
                      >
                    </div>
                  </a>
                  <a
                    href="https://tttbrother.com/"
                    target="_blank"
                    title="Website"
                    class="group/link no-underline"
                  >
                    <div
                      class="w-8 h-8 rounded-lg flex items-center justify-center bg-slate-950 border border-slate-800 group-hover/link:border-cyan-500/50 group-hover/link:bg-cyan-500/5 transition-all"
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
                        class="text-slate-500 group-hover/link:text-cyan-400"
                        ><circle cx="12" cy="12" r="10" /><line
                          x1="2"
                          y1="12"
                          x2="22"
                          y2="12"
                        /><path
                          d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"
                        /></svg
                      >
                    </div>
                  </a>
                  <div
                    title="Call: 085 818 8910"
                    class="group/link cursor-help"
                  >
                    <div
                      class="w-8 h-8 rounded-lg flex items-center justify-center bg-slate-950 border border-slate-800 hover:border-emerald-500/50 hover:bg-emerald-500/5 transition-all"
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
                        class="text-slate-500 group-hover/link:text-emerald-400"
                        ><path
                          d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"
                        /></svg
                      >
                    </div>
                  </div>
                </div>
              </div>
            {/if}
          </div>
        </aside>
      </div>

    <!-- Main Content Area -->
    <main class="flex-1 h-screen overflow-y-auto relative w-full pt-16 md:pt-0">
      <!-- Header decoration -->
      <div
        class="fixed top-0 left-0 w-full h-[500px] bg-gradient-to-b from-cyan-900/10 via-slate-900/5 to-transparent pointer-events-none -z-10"
      ></div>

      <div class="min-h-full p-6 md:p-8 max-w-7xl mx-auto w-full">
        <slot />
      </div>
    </main>
  </div>
{/if}

<!-- Help Center Modal -->
<Modal
  bind:open={showHelpModal}
  title="T-MONITOR HELP CENTER"
  maxWidth="max-w-4xl"
>
  <div class="space-y-8 py-2">
    <!-- Welcome Header -->
    <div class="text-center space-y-2">
      <div
        class="inline-flex items-center justify-center w-12 h-12 rounded-2xl bg-cyan-500/10 border border-cyan-500/20 text-cyan-400 mb-2"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><circle cx="12" cy="12" r="10" /><path
            d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"
          /><line x1="12" y1="17" x2="12.01" y2="17" /></svg
        >
      </div>
      <h3 class="text-xl font-bold text-white tracking-tight">
        How T-Monitor Works
      </h3>
      <p class="text-sm text-slate-400 max-w-sm mx-auto">
        Learn how to set up your API monitoring ecosystem in 4 simple steps.
      </p>
    </div>

    <!-- Visual Flow Diagram (CSS Grid) -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 relative">
      <!-- Connection Lines (Desktop only) -->
      <div
        class="hidden md:block absolute top-8 left-[12%] right-[12%] h-px bg-gradient-to-r from-transparent via-slate-700 to-transparent z-0"
      ></div>

      <!-- Step 1 -->
      <div class="relative z-10 flex flex-col items-center text-center group">
        <div
          class="w-10 h-10 rounded-full bg-slate-900 border-2 border-slate-800 flex items-center justify-center text-xs font-bold text-slate-500 group-hover:border-purple-500/50 group-hover:text-purple-400 transition-all duration-300 mb-3 bg-clip-padding"
        >
          1
        </div>
        <div
          class="p-4 rounded-2xl bg-slate-900/50 border border-slate-800 group-hover:bg-purple-500/5 group-hover:border-purple-500/30 transition-all duration-300 w-full"
        >
          <div class="text-purple-400 mb-2 flex justify-center">
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
              ><rect x="3" y="3" width="18" height="18" rx="2" ry="2" /><line
                x1="12"
                y1="8"
                x2="12"
                y2="16"
              /><line x1="8" y1="12" x2="16" y2="12" /></svg
            >
          </div>
          <p
            class="text-[11px] font-bold text-slate-200 uppercase tracking-widest mb-1"
          >
            PROJET
          </p>
          <p class="text-[10px] text-slate-500 leading-relaxed font-medium">
            Create workspace to group your APIs.
          </p>
        </div>
      </div>

      <!-- Step 2 -->
      <div class="relative z-10 flex flex-col items-center text-center group">
        <div
          class="w-10 h-10 rounded-full bg-slate-900 border-2 border-slate-800 flex items-center justify-center text-xs font-bold text-slate-500 group-hover:border-cyan-500/50 group-hover:text-cyan-400 transition-all duration-300 mb-3 bg-clip-padding"
        >
          2
        </div>
        <div
          class="p-4 rounded-2xl bg-slate-900/50 border border-slate-800 group-hover:bg-cyan-500/5 group-hover:border-cyan-500/30 transition-all duration-300 w-full"
        >
          <div class="text-cyan-400 mb-2 flex justify-center">
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
              ><path
                d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"
              /><path
                d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"
              /></svg
            >
          </div>
          <p
            class="text-[11px] font-bold text-slate-200 uppercase tracking-widest mb-1"
          >
            API REG
          </p>
          <p class="text-[10px] text-slate-500 leading-relaxed font-medium">
            Add endpoints and expected status.
          </p>
        </div>
      </div>

      <!-- Step 3 -->
      <div class="relative z-10 flex flex-col items-center text-center group">
        <div
          class="w-10 h-10 rounded-full bg-slate-900 border-2 border-slate-800 flex items-center justify-center text-xs font-bold text-slate-500 group-hover:border-amber-500/50 group-hover:text-amber-400 transition-all duration-300 mb-3 bg-clip-padding"
        >
          3
        </div>
        <div
          class="p-4 rounded-2xl bg-slate-900/50 border border-slate-800 group-hover:bg-amber-500/5 group-hover:border-amber-500/30 transition-all duration-300 w-full"
        >
          <div class="text-amber-400 mb-2 flex justify-center">
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
              ><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" /><path
                d="M13.73 21a2 2 0 0 1-3.46 0"
              /></svg
            >
          </div>
          <p
            class="text-[11px] font-bold text-slate-200 uppercase tracking-widest mb-1"
          >
            NOTIFY
          </p>
          <p class="text-[10px] text-slate-500 leading-relaxed font-medium">
            Set Telegram/Gmail for instant alerts.
          </p>
        </div>
      </div>

      <!-- Step 4 -->
      <div class="relative z-10 flex flex-col items-center text-center group">
        <div
          class="w-10 h-10 rounded-full bg-slate-900 border-2 border-slate-800 flex items-center justify-center text-xs font-bold text-slate-500 group-hover:border-emerald-500/50 group-hover:text-emerald-400 transition-all duration-300 mb-3 bg-clip-padding"
        >
          4
        </div>
        <div
          class="p-4 rounded-2xl bg-slate-900/50 border border-slate-800 group-hover:bg-emerald-500/5 group-hover:border-emerald-500/30 transition-all duration-300 w-full"
        >
          <div class="text-emerald-400 mb-2 flex justify-center">
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
              ><polyline points="22 12 18 12 15 21 9 3 6 12 2 12" /></svg
            >
          </div>
          <p
            class="text-[11px] font-bold text-slate-200 uppercase tracking-widest mb-1"
          >
            LIVE
          </p>
          <p class="text-[10px] text-slate-500 leading-relaxed font-medium">
            Monitor health via Status Live screen.
          </p>
        </div>
      </div>
    </div>

    <!-- Additional Help Sections -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div
        class="p-5 rounded-2xl bg-slate-950/50 border border-slate-800/50 hover:border-slate-700 transition-colors"
      >
        <h4 class="text-sm font-bold text-white mb-3 flex items-center gap-2">
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
            ><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline
              points="22 4 12 14.01 9 11.01"
            /></svg
          >
          What is "System Integrity"?
        </h4>
        <p class="text-[11px] text-slate-400 leading-relaxed">
          It represents the overall health of your API ecosystem. It calculates
          the percentage of healthy APIs across all your projects. A score below
          100% means some services are failing.
        </p>
      </div>
      <div
        class="p-5 rounded-2xl bg-slate-950/50 border border-slate-800/50 hover:border-slate-700 transition-colors"
      >
        <h4 class="text-sm font-bold text-white mb-3 flex items-center gap-2">
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
            class="text-amber-400"
            ><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path
              d="M7 11V7a5 5 0 0 1 10 0v4"
            /></svg
          >
          Gmail App Password?
        </h4>
        <p class="text-[11px] text-slate-400 leading-relaxed">
          To send alerts via Gmail, you must use an <strong>App Password</strong
          >. Enable 2FA in your Google Account settings, then search for "App
          Passwords" to generate a unique 16-character code for this app.
        </p>
      </div>
    </div>

    <!-- Contact Support -->
    <div
      class="pt-4 border-t border-slate-800 flex flex-col md:flex-row items-center justify-between gap-4"
    >
      <div class="flex items-center gap-3">
        <div
          class="w-8 h-8 rounded-full bg-cyan-500/10 flex items-center justify-center text-cyan-400"
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
              d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"
            /></svg
          >
        </div>
        <p class="text-xs font-medium text-slate-400">
          Need more help? Contact our support team.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <a
          href="https://tttbrother.com/"
          target="_blank"
          class="px-4 py-2 rounded-xl bg-slate-800 text-xs font-bold text-white hover:bg-slate-700 transition-colors"
          >Support Portal</a
        >
      </div>
    </div>
  </div>
</Modal>

<!-- Notification Center Modal -->
<Modal bind:open={showNotificationCenter} title="NOTIFICATION CENTER" maxWidth="max-w-lg">
  <div class="space-y-6">
    <!-- Account Indicator to prevent session confusion -->
    <div class="px-4 py-3 bg-slate-950/50 border border-slate-800 rounded-2xl flex items-center justify-between gap-3 shadow-inner">
        <div class="flex items-center gap-2">
            <div class="w-2 h-2 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]"></div>
            <p class="text-[10px] font-black text-slate-500 uppercase tracking-widest leading-none">Session Identity</p>
        </div>
        <div class="flex flex-col items-end">
            <p class="text-[11px] font-bold text-cyan-400 truncate max-w-[200px]">{user?.email}</p>
            <p class="text-[9px] text-slate-600 font-bold uppercase tracking-tighter">Current active account</p>
        </div>
    </div>

    {#if unreadNotifications.length === 0}
      <div class="py-12 text-center space-y-4">
        <div class="w-16 h-16 bg-slate-800/50 rounded-full flex items-center justify-center mx-auto text-slate-600">
           <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path><path d="M13.73 21a2 2 0 0 1-3.46 0"></path></svg>
        </div>
        <p class="text-slate-500 font-medium font-mono text-sm tracking-widest uppercase">No unread notifications</p>
      </div>
    {:else}
      <div class="space-y-3 max-h-[400px] overflow-y-auto pr-2 custom-scrollbar">
        {#each unreadNotifications as note}
          <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800 flex items-start gap-4 transition-all hover:bg-slate-800/50">
             <div class="mt-1 w-8 h-8 rounded-lg flex items-center justify-center shrink-0 {note.type === 'api_fail' ? 'bg-rose-500/10 text-rose-500' : 'bg-blue-500/10 text-blue-500'}">
                {#if note.type === 'api_fail'}
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                {:else}
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/></svg>
                {/if}
             </div>
             <div class="space-y-1 flex-1">
                <h4 class="text-sm font-bold text-slate-100">{note.title}</h4>
                <p class="text-xs text-slate-400 leading-relaxed">{note.message}</p>
                {#if note.type === 'company_invite' && note.invitation_id}
                  <div class="flex gap-2 pt-3">
                    <button 
                      on:click={() => handleAcceptInvitation(note.invitation_id)}
                      class="px-4 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white text-[10px] font-bold rounded-lg transition-all"
                    >
                      ACCEPT
                    </button>
                    <button 
                      on:click={() => handleDeclineInvitation(note.invitation_id)}
                      class="px-4 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-400 text-[10px] font-bold rounded-lg transition-all border border-slate-700"
                    >
                      DECLINE
                    </button>
                  </div>
                {/if}
                <p class="text-[10px] text-slate-600 font-bold uppercase tracking-tighter pt-1">{new Date(note.created_at).toLocaleString()}</p>
             </div>
          </div>
        {/each}
      </div>
      
      <div class="flex justify-between items-center pt-4 border-t border-slate-800">
        <p class="text-[10px] text-slate-500 font-bold uppercase tracking-widest leading-none">Showing {unreadNotifications.length} alerts</p>
        <button 
          on:click={markAllNotificationsRead}
          class="px-5 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-xl text-[10px] font-black uppercase tracking-widest transition-all shadow-lg shadow-rose-900/20"
        >
          MARK ALL AS READ
        </button>
      </div>
    {/if}
  </div>
</Modal>

<style>
  /* Custom glassmorphism and subtle animations */
  :global(.help-step-number) {
    box-shadow: 0 0 15px rgba(0, 0, 0, 0.4);
  }
</style>
