<script lang="ts">
  import { onMount } from "svelte";
  import Modal from "$lib/components/Modal.svelte";
  import { API_BASE_URL } from "$lib/config";

  let projects: any[] = [];
  let isLoading = true;

  // Create modal state
  let showCreateModal = false;
  let newProjectName = "";
  let newProjectDesc = "";

  // Edit modal state
  let showEditModal = false;
  let editingProjectId = 0;
  let editProjectName = "";
  let editProjectDesc = "";
  let editProjectEnvVars = "{}";

  // Delete modal state
  let showDeleteModal = false;
  let deletingProjectId = 0;
  let deletingProjectName = "";

  // Cover image upload state
  let coverFile: File | null = null;
  let editCoverFile: File | null = null;
  let newProjectCoverPos = 50;
  let editProjectCoverPos = 50;

  // Dropdown UI state
  let activeDropdownId: number | null = null;

  function toggleDropdown(id: number, event: Event) {
    event.stopPropagation();
    activeDropdownId = activeDropdownId === id ? null : id;
  }

  function handleWindowClick() {
    activeDropdownId = null;
  }

  onMount(async () => {
    await fetchProjects();
  });

  async function fetchProjects() {
    isLoading = true;
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
    } finally {
      isLoading = false;
    }
  }

  async function uploadCoverImage(projectId: number, file: File) {
    const formData = new FormData();
    formData.append("cover", file);
    const token = localStorage.getItem("monitor_token");
    const res = await fetch(`${API_BASE_URL}/api/v1/projects/${projectId}/cover`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: formData,
    });
    return res.ok;
  }

  async function handleCreateProject() {
    if (!newProjectName) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ name: newProjectName, description: newProjectDesc, environment_variables: "{}" }),
      });
      if (res.ok) {
        const project = await res.json();
        if (coverFile) await uploadCoverImage(project.id, coverFile);
        showCreateModal = false;
        newProjectName = ""; newProjectDesc = ""; coverFile = null;
        window.location.reload();
      }
    } catch (err) { console.error(err); }
  }

  function openEditModal(project: any) {
    editingProjectId = project.id;
    editProjectName = project.name;
    editProjectDesc = project.description || "";
    editProjectEnvVars = project.environment_variables || "{}";
    editProjectCoverPos = project.cover_position ?? 50;
    showEditModal = true;
    activeDropdownId = null;
  }

  function openDeleteModal(project: any) {
    deletingProjectId = project.id;
    deletingProjectName = project.name;
    showDeleteModal = true;
    activeDropdownId = null;
  }

  async function handleEditProject() {
    if (!editProjectName) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects/${editingProjectId}`, {
        method: "PUT",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ name: editProjectName, description: editProjectDesc, environment_variables: editProjectEnvVars, cover_position: editProjectCoverPos }),
      });
      if (res.ok) {
        if (editCoverFile) await uploadCoverImage(editingProjectId, editCoverFile);
        showEditModal = false; editCoverFile = null;
        window.location.reload();
      }
    } catch (err) { console.error(err); }
  }

  async function handleDeleteProject() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects/${deletingProjectId}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) { showDeleteModal = false; window.location.reload(); }
    } catch (err) { console.error(err); }
  }
</script>

<svelte:window on:click={handleWindowClick} />

<div class="fade-in">
  <div class="flex flex-col sm:flex-row justify-between items-start sm:items-end mb-10 gap-4 relative z-10">
    <div>
      <h1 class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase">
        Project API
      </h1>
      <p class="text-cyan-500/80 mt-2 font-mono text-sm tracking-wide">
        MANAGE YOUR API WORKSPACES AND CONFIGURATIONS.
      </p>
    </div>
    <button
      on:click={() => (showCreateModal = true)}
      class="bg-slate-900 border border-cyan-500/50 text-cyan-400 hover:bg-cyan-950/50 hover:border-cyan-400 hover:text-cyan-300 font-bold py-2.5 px-6 rounded-xl shadow-[0_0_15px_rgba(6,182,212,0.3)] hover:shadow-[0_0_25px_rgba(6,182,212,0.5)] transition-all flex items-center gap-2 group transform hover:-translate-y-0.5 font-mono tracking-wider overflow-hidden relative"
    >
      <div class="absolute inset-0 w-full h-full bg-cyan-400/10 -translate-x-full group-hover:animate-[shimmer_1.5s_infinite] skew-x-12"></div>
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="relative z-10 group-hover:rotate-90 transition-transform duration-300">
        <line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line>
      </svg>
      <span class="relative z-10">NEW_PROJECT</span>
    </button>
  </div>

  {#if isLoading}
    <div class="flex justify-center p-12">
      <svg class="animate-spin h-8 w-8 text-cyan-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
    </div>
  {:else if projects.length === 0}
    <div class="bg-slate-800/40 backdrop-blur-xl border border-slate-700/50 text-center rounded-3xl p-16 shadow-[0_8px_30px_rgb(0,0,0,0.5)] relative overflow-hidden group/empty">
      <div class="absolute inset-0 bg-cyan-900/5 opacity-0 group-hover/empty:opacity-100 transition-opacity duration-500"></div>
      <div class="inline-flex items-center justify-center w-24 h-24 rounded-full bg-slate-900 border border-cyan-500/30 mb-6 shadow-[0_0_15px_rgba(6,182,212,0.2)] relative z-10">
        <svg xmlns="http://www.w3.org/2000/svg" class="text-cyan-400 h-10 w-10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
        </svg>
      </div>
      <h3 class="text-2xl font-bold text-cyan-50 mb-3 font-mono tracking-wide relative z-10">NO_WORKSPACES_FOUND</h3>
      <p class="text-slate-400/80 max-w-md mx-auto mb-10 font-mono text-sm relative z-10">INITIALIZE A PROJECT WORKSPACE TO START GROUPING YOUR APIS FOR DIAGNOSTIC MONITORING.</p>
      <button on:click={() => (showCreateModal = true)} class="bg-cyan-950/40 border border-cyan-500/50 text-cyan-400 font-bold py-3 px-8 rounded-xl hover:bg-cyan-900/60 hover:text-cyan-300 transition-colors font-mono tracking-widest relative z-10 hover:shadow-[0_0_15px_rgba(6,182,212,0.3)]">
        INITIALIZE_FIRST_PROJECT
      </button>
    </div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 relative z-10">
      {#each projects as project}
        <div class="bg-slate-800/40 backdrop-blur-xl rounded-3xl border border-slate-700/50 p-6 shadow-[0_8px_30px_rgb(0,0,0,0.5)] transition-all duration-500 hover:shadow-[0_0px_30px_rgba(6,182,212,0.15)] hover:border-cyan-500/40 hover:-translate-y-1 flex flex-col group relative overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-cyan-900/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"></div>

          <!-- Project Cover Image -->
          <div class="h-32 -mx-6 -mt-6 mb-4 relative overflow-hidden bg-slate-900/50">
            {#if project.cover_image_url}
              <img src={project.cover_image_url.startsWith("http") ? project.cover_image_url : `${API_BASE_URL}${project.cover_image_url}`} alt={project.name}
                style={`object-position: 50% ${project.cover_position ?? 50}%`}
                class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110" />
            {:else}
              <div class="w-full h-full flex items-center justify-center opacity-20">
                <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" class="text-cyan-400">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline>
                </svg>
              </div>
            {/if}
            <div class="absolute inset-0 bg-gradient-to-t from-slate-800/80 to-transparent"></div>
          </div>

          <div class="flex justify-between items-start mb-5 relative z-10">
            <div class="p-3 bg-slate-900 border border-slate-700/80 text-cyan-500/80 rounded-xl group-hover:border-cyan-500/50 group-hover:text-cyan-400 group-hover:shadow-[0_0_15px_rgba(6,182,212,0.3)] transition-all duration-300 relative overflow-hidden">
              <div class="absolute inset-0 bg-cyan-400/20 mix-blend-screen filter blur-md opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="relative z-10">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
              </svg>
            </div>
            <button on:click={(e) => toggleDropdown(project.id, e)} class="text-slate-500 hover:text-cyan-400 p-2 rounded-lg hover:bg-slate-800 transition-colors">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="1.5"></circle><circle cx="19" cy="12" r="1.5"></circle><circle cx="5" cy="12" r="1.5"></circle>
              </svg>
            </button>
            {#if activeDropdownId === project.id}
              <div class="absolute right-0 top-10 mt-1 w-36 bg-slate-800 rounded-xl shadow-xl border border-slate-700 py-1 z-20 animate-in slide-in-from-top-2 duration-150" on:click|stopPropagation role="menu" tabindex="-1" on:keydown>
                <button on:click={() => openEditModal(project)} class="w-full text-left px-4 py-2 text-sm font-medium text-slate-300 hover:text-cyan-400 hover:bg-slate-700/50 transition-colors font-mono tracking-wide">EDIT</button>
                <div class="h-px w-full bg-slate-700/50 my-1"></div>
                <button on:click={() => openDeleteModal(project)} class="w-full text-left px-4 py-2 text-sm font-medium text-red-400 hover:text-red-300 hover:bg-red-500/10 transition-colors font-mono tracking-wide">TERMINATE</button>
              </div>
            {/if}
          </div>
          <h3 class="text-xl font-bold text-cyan-50 mb-2 truncate font-mono tracking-wide relative z-10">{project.name}</h3>
          <p class="text-slate-400/80 text-sm mb-6 flex-1 line-clamp-2 font-mono relative z-10">{project.description || "NO_DESCRIPTION_PROVIDED"}</p>
          <div class="pt-5 border-t border-slate-700/50 flex justify-between items-center relative z-10">
            <span class="text-[10px] font-bold bg-slate-900 border border-slate-700 group-hover:border-cyan-500/30 text-slate-400 group-hover:text-cyan-400 px-3 py-1.5 rounded-md tracking-wider font-mono transition-colors">
              {project.apis?.length || 0} APIs
            </span>
            <a href={`/dashboard/projects/${project.id}`} class="text-sm font-bold text-cyan-500 hover:text-cyan-300 flex items-center gap-1.5 group-hover:underline font-mono tracking-wider transition-colors">
              ACCESS
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="group-hover:translate-x-1 transition-transform">
                <line x1="5" y1="12" x2="19" y2="12"></line><polyline points="12 5 19 12 12 19"></polyline>
              </svg>
            </a>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Create Project Modal -->
<Modal bind:open={showCreateModal} title="Create Project">
  <form on:submit|preventDefault={handleCreateProject} class="space-y-5">
    <div>
      <label for="name" class="block text-sm font-medium text-cyan-50 mb-1">Project Name</label>
      <input id="name" type="text" bind:value={newProjectName} required class="w-full px-4 py-3 rounded-xl border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all text-sm" placeholder="e.g. E-Commerce Core API" />
    </div>
    <div>
      <label for="desc" class="block text-sm font-medium text-cyan-50 mb-1">Description</label>
      <textarea id="desc" rows="3" bind:value={newProjectDesc} class="w-full px-4 py-3 rounded-xl border border-slate-700/50 bg-slate-900/50 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all text-sm resize-none" placeholder="Brief description of this workspace..."></textarea>
    </div>
    <div>
      <label for="cover" class="block text-sm font-medium text-cyan-50 mb-1">Cover Image</label>
      <input id="cover" type="file" accept="image/*"
        on:change={(e) => (coverFile = (e.target as HTMLInputElement).files?.[0] || null)}
        class="w-full px-4 py-2 rounded-xl border border-slate-700/50 bg-slate-900/50 text-cyan-50/70 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all" />
    </div>
    <div class="pt-2 flex gap-3">
      <button type="button" on:click={() => (showCreateModal = false)} class="flex-1 px-4 py-3 rounded-xl font-medium text-slate-400 bg-slate-800 hover:bg-slate-200 transition-colors">Cancel</button>
      <button type="submit" class="flex-1 px-4 py-3 rounded-xl font-medium text-cyan-50 bg-cyan-600 hover:bg-blue-700 transition-colors">Create</button>
    </div>
  </form>
</Modal>

<!-- Edit Project Modal -->
<Modal bind:open={showEditModal} title="Edit Project">
  <form on:submit|preventDefault={handleEditProject} class="space-y-5">
    <div>
      <label for="edit_name" class="block text-sm font-medium text-cyan-50 mb-1">Project Name</label>
      <input id="edit_name" type="text" bind:value={editProjectName} required class="w-full px-4 py-3 rounded-xl border border-slate-700/50 focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all text-sm" />
    </div>
    <div>
      <label for="edit_desc" class="block text-sm font-medium text-cyan-50 mb-1">Description</label>
      <textarea id="edit_desc" rows="3" bind:value={editProjectDesc} class="w-full px-4 py-3 rounded-xl border border-slate-700/50 bg-slate-900/50 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all text-sm resize-none"></textarea>
    </div>
    <div>
      <label for="edit_cover" class="block text-sm font-medium text-cyan-50 mb-1">Update Cover Image</label>
      <input id="edit_cover" type="file" accept="image/*"
        on:change={(e) => (editCoverFile = (e.target as HTMLInputElement).files?.[0] || null)}
        class="w-full px-4 py-2 rounded-xl border border-slate-700/50 bg-slate-900/50 text-cyan-50/70 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all" />
    </div>
    <div class="space-y-1">
      <label for="edit_pos" class="block text-sm font-medium text-cyan-50">Position Adjustment (Vertical: {editProjectCoverPos}%)</label>
      <input id="edit_pos" type="range" min="0" max="100" bind:value={editProjectCoverPos} class="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-cyan-500" />
    </div>
    <div class="pt-2 flex gap-3">
      <button type="button" on:click={() => (showEditModal = false)} class="flex-1 px-4 py-3 rounded-xl font-medium text-slate-400 bg-slate-800 hover:bg-slate-200 transition-colors">Cancel</button>
      <button type="submit" class="flex-1 px-4 py-3 rounded-xl font-medium text-cyan-50 bg-cyan-600 hover:bg-blue-700 transition-colors">Save Changes</button>
    </div>
  </form>
</Modal>

<!-- Delete Project Modal -->
<Modal bind:open={showDeleteModal} title="Delete Project">
  <div class="space-y-5">
    <p class="text-slate-400 text-sm">Are you sure you want to delete <span class="font-bold text-cyan-50">{deletingProjectName}</span>? This action cannot be undone and will remove all associated APIs and monitoring logs.</p>
    <div class="pt-2 flex gap-3">
      <button type="button" on:click={() => (showDeleteModal = false)} class="flex-1 px-4 py-3 rounded-xl font-medium text-slate-400 bg-slate-800 hover:bg-slate-200 transition-colors">Cancel</button>
      <button type="button" on:click={handleDeleteProject} class="flex-1 px-4 py-3 rounded-xl font-medium text-cyan-50 bg-red-500/20 text-red-400 border border-red-500/30 hover:bg-red-700 transition-colors">Delete Workspace</button>
    </div>
  </div>
</Modal>
