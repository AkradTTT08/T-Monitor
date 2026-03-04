<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";

  let projects: any[] = [];
  let selectedProjectId = "";

  // Listen to page changes to auto-select the right project ID
  $: {
    if ($page.params.id) {
      selectedProjectId = $page.params.id;
      if (typeof window !== "undefined") {
        localStorage.setItem("monitor_selected_project", selectedProjectId);
      }
    }
  }

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

  $: currentPath = $page.url.pathname;

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

  function updateActivity() {
    lastActivityTimestamp = Date.now();
  }

  // Separate mount call for project fetching (no cleanup needed)
  onMount(async () => {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch("http://localhost:5273/api/v1/projects", {
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
  });

  // Session management mount (returns cleanup fn)
  onMount(() => {
    const token = localStorage.getItem("monitor_token");
    const userData = localStorage.getItem("monitor_user");

    if (!token || !userData) {
      window.location.href = "/";
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

    return () => {
      window.fetch = originalFetch; // Restore original fetch
      window.removeEventListener("mousemove", updateActivity);
      window.removeEventListener("keydown", updateActivity);
      window.removeEventListener("click", updateActivity);
      window.removeEventListener("scroll", updateActivity);
      window.removeEventListener("user-updated", handleUserUpdate);
      if (sessionInterval) clearInterval(sessionInterval);
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

      const res = await fetch("http://localhost:5273/api/v1/auth/refresh", {
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
    window.location.href = "/";
  }
</script>

<svelte:window on:click={() => (showProfileMenu = false)} />

{#if user}
  <div
    class="h-screen w-full bg-slate-50 flex overflow-hidden font-sans text-slate-800"
  >
    <!-- Mobile Menu Button -->
    <button
      class="md:hidden fixed top-4 right-4 z-50 p-2 rounded-lg bg-white shadow-md border border-slate-200 text-slate-600"
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
      class="fixed md:static inset-y-0 left-0 flex h-full bg-[#fcfcfc] z-40 transition-transform duration-300 transform {isMobileMenuOpen
        ? 'translate-x-0'
        : '-translate-x-full md:translate-x-0'} border-r border-slate-200 relative group/sidebar overflow-visible"
    >
      <!-- Floating sidebar toggle button on right border -->
      <button
        on:click={() => (isSidebarCollapsed = !isSidebarCollapsed)}
        title={isSidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
        class="absolute right-0 translate-x-1/2 top-[40px] z-50
          w-6 h-6 flex items-center justify-center rounded-full
          bg-white border border-slate-200 shadow-md text-slate-500
          hover:text-slate-900 hover:border-slate-400 hover:shadow-lg
          opacity-0 hover:opacity-100 group-hover/sidebar:opacity-100
          transition-all duration-200"
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
            class="w-10 h-10 rounded-xl bg-slate-900 flex shrink-0 items-center justify-center text-white shadow-sm"
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
              ><path d="M20.2 7.8l-7.7 7.7-4-4-5.7 5.7" /><path
                d="M15 7h6v6"
              /></svg
            >
          </div>
          {#if !isSidebarCollapsed}
            <div class="flex flex-col overflow-hidden min-w-0 flex-1">
              <span
                class="font-bold text-slate-900 text-sm leading-tight truncate"
                >T-Monitor</span
              >
              <span class="text-xs text-slate-500 truncate">Enterprise</span>
            </div>
          {/if}
        </div>

        <!-- Main Navigation -->
        <div
          class="flex-1 overflow-y-auto space-y-1.5 {isSidebarCollapsed
            ? 'px-0'
            : 'pr-1'} hide-scrollbar"
        >
          <a
            href={selectedProjectId
              ? `/dashboard/projects/${selectedProjectId}`
              : "/dashboard"}
            on:click={() => (isMobileMenuOpen = false)}
            title="Project APIs"
            class="w-full flex items-center {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath ===
            '/dashboard'
              ? 'bg-[#ecff82] text-slate-900 shadow-sm'
              : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}"
          >
            <div class="flex items-center gap-3">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                class={currentPath === "/dashboard"
                  ? "text-slate-900"
                  : "text-slate-500"}
                stroke="currentColor"
                stroke-width="2.5"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><rect x="3" y="3" width="7" height="9" rx="1" /><rect
                  x="14"
                  y="3"
                  width="7"
                  height="5"
                  rx="1"
                /><rect x="14" y="12" width="7" height="9" rx="1" /><rect
                  x="3"
                  y="16"
                  width="7"
                  height="5"
                  rx="1"
                /></svg
              >
              {#if !isSidebarCollapsed}<span>Project APIs</span>{/if}
            </div>
          </a>

          <!-- Open APIs Link -->
          <a
            href="/dashboard/apis"
            on:click={() => (isMobileMenuOpen = false)}
            title="Open APIs"
            class="w-full flex items-center {isSidebarCollapsed
              ? 'justify-center px-0 relative'
              : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath ===
            '/dashboard/apis'
              ? 'bg-[#ecff82] text-slate-900 shadow-sm'
              : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}"
          >
            <div class="flex items-center gap-3">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                class={currentPath === "/dashboard/apis"
                  ? "text-slate-900"
                  : "text-slate-500"}
                stroke="currentColor"
                stroke-width="2.5"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><path d="M12 2v20" /><path
                  d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"
                /></svg
              >
              {#if !isSidebarCollapsed}<span>Open APIs</span>{/if}
            </div>
            {#if !isSidebarCollapsed}
              <span
                class="bg-slate-900 text-white text-[10px] font-bold px-2 py-0.5 rounded-md"
                >8</span
              >
            {:else}
              <span
                class="absolute top-1.5 right-1.5 w-2.5 h-2.5 bg-slate-900 border-2 border-slate-50 rounded-full"
              ></span>
            {/if}
          </a>

          <!-- Status Live Link -->
          <a
            href="/dashboard/status"
            on:click={() => (isMobileMenuOpen = false)}
            title="Status Live"
            class="w-full flex items-center {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath ===
            '/dashboard/status'
              ? 'bg-[#ecff82] text-slate-900 shadow-sm'
              : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}"
          >
            <div class="flex items-center gap-3">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                class={currentPath === "/dashboard/status"
                  ? "text-slate-900"
                  : "text-slate-500"}
                stroke="currentColor"
                stroke-width="2.5"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"
                ></polyline></svg
              >
              {#if !isSidebarCollapsed}<span>Status Live</span>{/if}
            </div>
          </a>

          <!-- Notification Channels Link -->
          <a
            href={`/dashboard/projects/${$page.params.id || "1"}/notifications`}
            on:click={() => (isMobileMenuOpen = false)}
            title="Alerts & Channels"
            class="w-full flex items-center {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath.includes(
              'notifications',
            )
              ? 'bg-amber-100 text-amber-900 shadow-sm'
              : 'text-slate-600 hover:bg-amber-50 hover:text-amber-700'}"
          >
            <div class="flex items-center gap-3">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                class={currentPath.includes("notifications")
                  ? "text-amber-700"
                  : "text-amber-500"}
                stroke="currentColor"
                stroke-width="2.5"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><path
                  d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"
                ></path><line x1="12" y1="9" x2="12" y2="13"></line><line
                  x1="12"
                  y1="17"
                  x2="12.01"
                  y2="17"
                ></line></svg
              >
              {#if !isSidebarCollapsed}<span>Alerts & Channels</span>{/if}
            </div>
          </a>

          {#if user.role === "admin"}
            <div class="pt-4 pb-1">
              {#if !isSidebarCollapsed}
                <div
                  class="px-3 flex items-center justify-between text-[10px] uppercase font-bold text-slate-400 tracking-wider mb-2"
                >
                  <span>Administration</span>
                </div>
              {:else}
                <div class="w-full h-px bg-slate-200 mb-2"></div>
              {/if}

              <!-- Manage Users Link -->
              <a
                href="/dashboard/users"
                on:click={() => (isMobileMenuOpen = false)}
                title="Users & Roles"
                class="w-full flex items-center {isSidebarCollapsed
                  ? 'justify-center px-0'
                  : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath ===
                '/dashboard/users'
                  ? 'bg-[#ecff82] text-slate-900 shadow-sm'
                  : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}"
              >
                <div class="flex items-center gap-3">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="20"
                    height="20"
                    viewBox="0 0 24 24"
                    fill="none"
                    class={currentPath === "/dashboard/users"
                      ? "text-slate-900"
                      : "text-slate-500"}
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
                  class="px-3 pt-4 pb-1 text-[10px] uppercase font-bold text-slate-400 tracking-wider"
                >
                  Projects
                </div>
              {:else}
                <div class="w-full h-px bg-slate-200 mt-3 mb-1"></div>
              {/if}

              <!-- Dynamic Project List -->
              {#if projects.length > 0}
                <div class="space-y-0.5">
                  {#each projects as project}
                    {@const isActive =
                      currentPath === `/dashboard/projects/${project.id}`}
                    <a
                      href={`/dashboard/projects/${project.id}`}
                      on:click={() => (isMobileMenuOpen = false)}
                      title={project.name}
                      class="w-full flex items-center {isSidebarCollapsed
                        ? 'justify-center py-2'
                        : 'px-3 py-2'} rounded-xl text-sm font-medium transition-all {isActive
                        ? 'bg-slate-100 text-slate-900'
                        : 'text-slate-500 hover:bg-slate-50 hover:text-slate-800'}"
                    >
                      {#if isSidebarCollapsed}
                        <!-- Collapsed: small circle avatar -->
                        <div
                          class="relative w-7 h-7 rounded-lg flex items-center justify-center text-[11px] font-bold uppercase shrink-0
                            {isActive
                            ? 'bg-slate-900 text-white'
                            : 'bg-slate-200 text-slate-600'}"
                        >
                          {project.name.charAt(0)}
                        </div>
                      {:else}
                        <!-- Expanded: dot + name + active badge -->
                        <div class="flex items-center gap-2.5 truncate w-full">
                          <div
                            class="w-2 h-2 rounded-full shrink-0 {isActive
                              ? 'bg-emerald-500'
                              : 'bg-slate-300'}"
                          ></div>
                          <span class="truncate text-sm">{project.name}</span>
                          {#if isActive}
                            <span
                              class="ml-auto text-[10px] font-semibold px-1.5 py-0.5 rounded-full bg-emerald-100 text-emerald-700 shrink-0"
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
        <div class="mt-auto pt-4 border-t border-slate-200 flex flex-col gap-4">
          <!-- Help Center -->
          <a
            href="#"
            class="w-full flex items-center {isSidebarCollapsed
              ? 'justify-center px-0'
              : 'gap-3 px-3'} py-2 rounded-xl text-sm font-semibold transition-all text-slate-500 hover:bg-slate-100 hover:text-slate-900"
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
          </a>

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
                : 'justify-between p-2'} rounded-xl hover:bg-slate-100 transition-all text-left group relative"
            >
              <div class="flex items-center gap-3 overflow-hidden min-w-0">
                <div
                  class="w-10 h-10 rounded-full bg-slate-900 flex shrink-0 items-center justify-center text-white font-bold text-sm shadow-sm uppercase relative border-2 border-slate-100 overflow-hidden"
                >
                  {#if user?.profile_image_url}
                    <img
                      src={user.profile_image_url}
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
                      class="font-bold text-slate-900 text-sm leading-tight truncate capitalize"
                      >{user?.name ||
                        user?.email?.split("@")[0] ||
                        "User"}</span
                    >
                    <span class="text-xs text-slate-500 font-medium truncate"
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
                  class="text-slate-400 group-hover:text-amber-500 transition-colors shrink-0"
                  ><polyline points="18 15 12 9 6 15"></polyline></svg
                >
              {/if}
            </button>

            <!-- Profile Popup Menu -->
            {#if showProfileMenu}
              <div
                class="absolute bottom-full left-0 {isSidebarCollapsed
                  ? 'ml-12'
                  : 'w-full'} mb-2 bg-white rounded-xl shadow-lg border border-slate-200 overflow-hidden z-50 animate-in slide-in-from-bottom-2 duration-200 min-w-[160px]"
              >
                <div class="p-1">
                  <a
                    href="/dashboard/settings/profile"
                    class="w-full text-left px-3 py-2.5 text-sm font-semibold text-slate-600 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors flex items-center gap-2"
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
                    class="w-full text-left px-3 py-2.5 text-sm font-semibold text-red-600 hover:bg-red-50 rounded-lg transition-colors flex items-center gap-2"
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

            <!-- Progress Bar (Static mock) -->
            {#if !isSidebarCollapsed}
              <div class="px-2 mt-4 animate-in fade-in duration-300">
                <div
                  class="flex items-center justify-between gap-3 text-[11px] font-bold text-slate-800 mb-1.5"
                >
                  <div
                    class="flex-1 h-1.5 bg-slate-200 rounded-full overflow-hidden"
                  >
                    <div
                      class="h-full bg-emerald-500 rounded-full w-[70%] shadow-sm"
                    ></div>
                  </div>
                  <span>70%</span>
                </div>
                <span class="text-[11px] font-medium text-slate-500"
                  >Complete your profile</span
                >
              </div>
            {/if}
          </div>
        </div>
      </aside>
    </div>

    <!-- Main Content Area -->
    <main class="flex-1 h-screen overflow-y-auto relative w-full pt-16 md:pt-0">
      <!-- Header decoration -->
      <div
        class="fixed top-0 left-0 w-full h-64 bg-gradient-to-b from-blue-50/80 to-transparent pointer-events-none -z-10"
      ></div>

      <div class="min-h-full p-6 md:p-8 max-w-7xl mx-auto w-full">
        <slot />
      </div>
    </main>
  </div>
{/if}
