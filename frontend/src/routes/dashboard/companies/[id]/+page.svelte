<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import Modal from "$lib/components/Modal.svelte";
  import { API_BASE_URL } from "$lib/config";
  import Swal from "sweetalert2";

  $: companyId = $page.params.id;

  let company: any = null;
  let projects: any[] = [];
  let isLoading = true;

  const systemAlert = Swal.mixin({
    customClass: {
      confirmButton: 'bg-cyan-600 hover:bg-cyan-700 text-white font-bold py-2 px-4 rounded-lg ml-3',
      cancelButton: 'bg-slate-700 hover:bg-slate-600 text-white font-bold py-2 px-4 rounded-lg'
    },
    buttonsStyling: false,
    background: '#0f172a',
    color: '#f1f5f9'
  });

  const systemToast = Swal.mixin({
    toast: true,
    position: 'top-end',
    showConfirmButton: false,
    timer: 3000,
    timerProgressBar: true,
    background: '#1e293b',
    color: '#f1f5f9'
  });

  // Create project modal
  let showCreateModal = false;
  let newProjectName = "";
  let newProjectDesc = "";
  let coverFile: File | null = null;

  // Edit project modal
  let showEditModal = false;
  let editingProjectId = "";
  let editProjectName = "";
  let editProjectDesc = "";
  let editProjectEnvVars = "{}";
  let editCoverFile: File | null = null;
  let editProjectCoverPos = 50;

  // Delete project modal
  let showDeleteModal = false;
  let deletingProjectId = "";
  let deletingProjectName = "";

  let activeDropdownId: string | null = null;
  
  // Project Members State
  let showMembersModal = false;
  let activeProject: any = null;
  let projectMembers: any[] = [];
  let isAddingMember = false;
  let selectedMemberId: string | null = null;

  function toggleDropdown(id: string, e: Event) {
    e.stopPropagation();
    activeDropdownId = activeDropdownId === id ? null : id;
  }

  onMount(async () => {
    await fetchData();
  });

  async function fetchData() {
    isLoading = true;
    try {
      const token = localStorage.getItem("monitor_token");
      // Fetch company info
      const companyRes = await fetch(`${API_BASE_URL}/api/v1/companies`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (companyRes.ok) {
        const companies = await companyRes.json();
        company = companies.find((c: any) => c.id.toString() === companyId);
      }
      // Fetch all projects and filter by company
      const projRes = await fetch(`${API_BASE_URL}/api/v1/projects`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (projRes.ok) {
        const allProjects = await projRes.json();
        projects = allProjects.filter((p: any) => p.company_id?.toString() === companyId);
      }
    } catch (err) {
      console.error(err);
    } finally {
      isLoading = false;
    }
  }

  async function uploadCover(projectId: string, file: File) {
    const formData = new FormData();
    formData.append("cover", file);
    const token = localStorage.getItem("monitor_token");
    const res = await fetch(`${API_BASE_URL}/api/v1/projects/${projectId}/cover`, {
      method: "POST", headers: { Authorization: `Bearer ${token}` }, body: formData,
    });
    return res.ok;
  }

  async function handleCreate() {
    if (!newProjectName.trim()) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ name: newProjectName, description: newProjectDesc, company_id: companyId, environment_variables: "{}" }),
      });
      if (res.ok) {
        const project = await res.json();
        if (coverFile) await uploadCover(project.id, coverFile);
        showCreateModal = false;
        newProjectName = ""; newProjectDesc = ""; coverFile = null;
        await fetchData();
        window.dispatchEvent(new CustomEvent("projects-updated"));
      }
    } catch (err) { console.error(err); }
  }

  function openEdit(project: any) {
    editingProjectId = project.id;
    editProjectName = project.name;
    editProjectDesc = project.description || "";
    editProjectEnvVars = project.environment_variables || "{}";
    editProjectCoverPos = project.cover_position ?? 50;
    editCoverFile = null;
    showEditModal = true;
    activeDropdownId = null;
  }

  async function handleEdit() {
    if (!editProjectName.trim()) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects/${editingProjectId}`, {
        method: "PUT",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ name: editProjectName, description: editProjectDesc, environment_variables: editProjectEnvVars, cover_position: editProjectCoverPos, company_id: companyId }),
      });
      if (res.ok) {
        if (editCoverFile) await uploadCover(editingProjectId, editCoverFile);
        showEditModal = false;
        await fetchData();
        window.dispatchEvent(new CustomEvent("projects-updated"));
      }
    } catch (err) { console.error(err); }
  }

  function openDelete(project: any) {
    deletingProjectId = project.id;
    deletingProjectName = project.name;
    showDeleteModal = true;
    activeDropdownId = null;
  }
  async function handleDelete() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects/${deletingProjectId}`, {
        method: "DELETE", headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        showDeleteModal = false;
        await fetchData();
        window.dispatchEvent(new CustomEvent("projects-updated"));
      }
    } catch (err) { console.error(err); }
  }

  async function openMembers(project: any) {
    activeProject = project;
    activeDropdownId = null;
    await fetchProjectMembers(project.id);
    showMembersModal = true;
  }

  async function fetchProjectMembers(projectId: string) {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects/${projectId}/members`, {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.ok) {
        projectMembers = await res.json();
      }
    } catch (err) {
      console.error("Failed to fetch project members:", err);
    }
  }

  async function addProjectMember() {
    if (!selectedMemberId || !activeProject) return;
    isAddingMember = true;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects/${activeProject.id}/members`, {
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
        systemToast.fire({ icon: 'success', title: 'Member added to project' });
        await fetchProjectMembers(activeProject.id);
        selectedMemberId = null;
      } else {
        const error = await res.json();
        systemAlert.fire('Error', error.error || 'Failed to add member', 'error');
      }
    } catch (err) {
      console.error(err);
    } finally {
      isAddingMember = false;
    }
  }

  async function removeProjectMember(userId: string) {
    if (!activeProject) return;
    const confirm = await Swal.fire({
      title: 'Are you sure?',
      text: "This user will lose access to this specific project.",
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText: 'Yes, remove',
      background: '#0f172a',
      color: '#f1f5f9'
    });

    if (!confirm.isConfirmed) return;

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/projects/${activeProject.id}/members/${userId}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` }
      });

      if (res.ok) {
        await fetchProjectMembers(activeProject.id);
      }
    } catch (err) {
      console.error(err);
    }
  }

  let user: any = null;
  onMount(async () => {
    const userData = localStorage.getItem("monitor_user");
    if (userData) user = JSON.parse(userData);
    await fetchData();
  });
</script>

<svelte:window onclick={() => (activeDropdownId = null)} />

<div class="fade-in">
  <!-- Breadcrumb -->
  <div class="flex items-center gap-2 text-xs font-mono text-slate-500 mb-6">
    <a href="/dashboard/companies" class="hover:text-cyan-400 transition-colors">COMPANIES</a>
    <span>›</span>
    <span class="text-cyan-400">{company?.name || "..."}</span>
  </div>

  <!-- Header -->
  <div class="flex flex-col sm:flex-row justify-between items-start sm:items-end mb-10 gap-4 relative z-10">
    <div>
      <h1 class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase">
        {company?.name || "Project API"}
      </h1>
      <p class="text-cyan-500/80 mt-2 font-mono text-sm tracking-wide">
        MANAGE PROJECT WORKSPACES FOR THIS COMPANY.
      </p>
    </div>
    {#if user && (user.id === company?.user_id || user.role === 'admin')}
      <button
        onclick={() => (showCreateModal = true)}
        class="bg-slate-900 border border-cyan-500/50 text-cyan-400 hover:bg-cyan-950/50 hover:border-cyan-400 hover:text-cyan-300 font-bold py-2.5 px-6 rounded-xl shadow-[0_0_15px_rgba(6,182,212,0.3)] hover:shadow-[0_0_25px_rgba(6,182,212,0.5)] transition-all flex items-center gap-2 group transform hover:-translate-y-0.5 font-mono tracking-wider overflow-hidden relative"
      >
        <div class="absolute inset-0 w-full h-full bg-cyan-400/10 -translate-x-full group-hover:animate-[shimmer_1.5s_infinite] skew-x-12"></div>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="relative z-10 group-hover:rotate-90 transition-transform duration-300">
          <line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        <span class="relative z-10">+ NEW_PROJECT</span>
      </button>
    {/if}
  </div>

  {#if isLoading}
    <div class="flex justify-center p-12">
      <svg class="animate-spin h-8 w-8 text-cyan-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
    </div>
  {:else if projects.length === 0}
    <div class="bg-slate-800/40 backdrop-blur-xl border border-slate-700/50 text-center rounded-3xl p-16 shadow-[0_8px_30px_rgb(0,0,0,0.5)]">
      <div class="inline-flex items-center justify-center w-20 h-20 rounded-full bg-slate-900 border border-cyan-500/30 mb-5">
        <svg xmlns="http://www.w3.org/2000/svg" class="text-cyan-400 h-9 w-9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
        </svg>
      </div>
      <h3 class="text-xl font-bold text-cyan-50 mb-2 font-mono">NO_PROJECTS_YET</h3>
      <p class="text-slate-400 text-sm mb-8 font-mono">Add the first project workspace for this company.</p>
      <button onclick={() => (showCreateModal = true)} class="bg-cyan-950/40 border border-cyan-500/50 text-cyan-400 font-bold py-2.5 px-6 rounded-xl hover:bg-cyan-900/60 transition-colors font-mono">
        CREATE_PROJECT
      </button>
    </div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 relative z-10">
      {#each projects as project}
        <div class="bg-slate-800/40 backdrop-blur-xl rounded-3xl border border-slate-700/50 p-6 shadow-[0_8px_30px_rgb(0,0,0,0.5)] transition-all duration-500 hover:shadow-[0_0px_30px_rgba(6,182,212,0.15)] hover:border-cyan-500/40 hover:-translate-y-1 flex flex-col group relative overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-cyan-900/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"></div>

          <!-- Cover image -->
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
            <div class="p-3 bg-slate-900 border border-slate-700/80 text-cyan-500/80 rounded-xl group-hover:border-cyan-500/50 group-hover:text-cyan-400 transition-all">
              <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
              </svg>
            </div>
            <div class="relative">
              <button onclick={(e) => toggleDropdown(project.id, e)} class="text-slate-500 hover:text-cyan-400 p-2 rounded-lg hover:bg-slate-800 transition-colors">
                <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="1.5"></circle><circle cx="19" cy="12" r="1.5"></circle><circle cx="5" cy="12" r="1.5"></circle>
                </svg>
              </button>
              {#if activeDropdownId === project.id}
                <div class="absolute right-0 top-10 mt-1 w-36 bg-slate-800 rounded-xl shadow-xl border border-slate-700 py-1 z-20 animate-in slide-in-from-top-2" onclick={(e) => e.stopPropagation()}>
                  {#if user && (user.id === (company?.user_id || project.user_id) || user.role === 'admin')}
                    <button onclick={() => openMembers(project)} class="w-full text-left px-4 py-2 text-sm font-medium text-blue-400 hover:text-blue-300 hover:bg-blue-500/10 transition-colors font-mono">MEMBERS</button>
                    <div class="h-px w-full bg-slate-700/50 my-1"></div>
                    <button onclick={() => openEdit(project)} class="w-full text-left px-4 py-2 text-sm font-medium text-slate-300 hover:text-cyan-400 hover:bg-slate-700/50 transition-colors font-mono">EDIT</button>
                    <div class="h-px w-full bg-slate-700/50 my-1"></div>
                    <button onclick={() => openDelete(project)} class="w-full text-left px-4 py-2 text-sm font-medium text-red-400 hover:text-red-300 hover:bg-red-500/10 transition-colors font-mono">TERMINATE</button>
                  {:else}
                    <div class="px-4 py-3 text-[10px] font-bold text-slate-500 uppercase tracking-tighter text-center italic">
                      View Only
                    </div>
                  {/if}
                </div>
              {/if}
            </div>
          </div>

          <h3 class="text-xl font-bold text-cyan-50 mb-2 truncate font-mono tracking-wide relative z-10">{project.name}</h3>
          <p class="text-slate-400/80 text-sm mb-6 flex-1 line-clamp-2 font-mono relative z-10">{project.description || "NO_DESCRIPTION_PROVIDED"}</p>

          <div class="pt-5 border-t border-slate-700/50 flex justify-between items-center relative z-10">
            <span class="text-[10px] font-bold bg-slate-900 border border-slate-700 group-hover:border-cyan-500/30 text-slate-400 group-hover:text-cyan-400 px-3 py-1.5 rounded-md tracking-wider font-mono transition-colors">
              {project.apis?.length || 0} APIs
            </span>
            <a href={`/dashboard/companies/${companyId}/project/${project.id}`}
              class="text-sm font-bold text-cyan-500 hover:text-cyan-300 flex items-center gap-1.5 font-mono tracking-wider transition-colors">
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
<Modal bind:open={showCreateModal} title="New Project" maxWidth="max-w-lg">
  <form onsubmit={(e) => { e.preventDefault(); handleCreate(); }} class="space-y-4">
    <div>
      <label for="p_name" class="block text-sm font-semibold text-cyan-50 mb-1">Project Name</label>
      <input id="p_name" type="text" bind:value={newProjectName} required class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm" placeholder="e.g. E-Commerce Core API" />
    </div>
    <div>
      <label for="p_desc" class="block text-sm font-semibold text-cyan-50 mb-1">Description</label>
      <textarea id="p_desc" rows="2" bind:value={newProjectDesc} class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm resize-none" placeholder="Brief description..."></textarea>
    </div>
    <div>
      <label for="p_cover" class="block text-sm font-semibold text-cyan-50 mb-1">Cover Image</label>
      <input id="p_cover" type="file" accept="image/*" onchange={(e) => (coverFile = (e.target as HTMLInputElement).files?.[0] || null)} class="w-full px-4 py-2 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50/70 text-xs" />
    </div>
    <div class="pt-2 flex gap-3">
      <button type="button" onclick={() => (showCreateModal = false)} class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-slate-400 bg-slate-800 hover:bg-slate-700 transition-colors text-sm">Cancel</button>
      <button type="submit" class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-white bg-cyan-600 hover:bg-cyan-700 transition-colors text-sm">Create Project</button>
    </div>
  </form>
</Modal>

<!-- Edit Project Modal -->
<Modal bind:open={showEditModal} title="Edit Project" maxWidth="max-w-lg">
  <form onsubmit={(e) => { e.preventDefault(); handleEdit(); }} class="space-y-4">
    <div>
      <label for="ep_name" class="block text-sm font-semibold text-cyan-50 mb-1">Project Name</label>
      <input id="ep_name" type="text" bind:value={editProjectName} required class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm" />
    </div>
    <div>
      <label for="ep_desc" class="block text-sm font-semibold text-cyan-50 mb-1">Description</label>
      <textarea id="ep_desc" rows="2" bind:value={editProjectDesc} class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm resize-none"></textarea>
    </div>
    <div>
      <label for="ep_cover" class="block text-sm font-semibold text-cyan-50 mb-1">Update Cover</label>
      <input id="ep_cover" type="file" accept="image/*" onchange={(e) => (editCoverFile = (e.target as HTMLInputElement).files?.[0] || null)} class="w-full px-4 py-2 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50/70 text-xs" />
    </div>
    <div class="space-y-1">
      <label for="ep_pos" class="block text-sm font-semibold text-cyan-50">Position Adjustment (Vertical: {editProjectCoverPos}%)</label>
      <input id="ep_pos" type="range" min="0" max="100" bind:value={editProjectCoverPos} class="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-cyan-500" />
    </div>
    <div class="pt-2 flex gap-3">
      <button type="button" onclick={() => (showEditModal = false)} class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-slate-400 bg-slate-800 hover:bg-slate-700 transition-colors text-sm">Cancel</button>
      <button type="submit" class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-white bg-cyan-600 hover:bg-cyan-700 transition-colors text-sm">Save Changes</button>
    </div>
  </form>
</Modal>

<!-- Delete Project Modal -->
<Modal bind:open={showDeleteModal} title="Delete Project">
  <div class="space-y-4">
    <div class="bg-amber-950/30 border border-amber-500/30 rounded-xl p-4">
      <p class="text-sm text-amber-300">Are you sure you want to delete <span class="font-bold text-white">{deletingProjectName}</span>? This removes all APIs and monitoring logs.</p>
    </div>
    <div class="flex gap-3">
      <button type="button" onclick={() => (showDeleteModal = false)} class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-slate-400 bg-slate-800 hover:bg-slate-700 transition-colors text-sm">Cancel</button>
      <button type="button" onclick={handleDelete} class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-white bg-red-600 hover:bg-red-700 transition-colors text-sm">Delete</button>
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
          {#if company && company.members}
            {#each company.members as cm}
              {#if !projectMembers.some((pm) => pm.user_id === cm.user_id)}
                <option value={cm.user_id}>{cm.user?.name || cm.user?.email || "Unknown"}</option>
              {/if}
            {/each}
          {/if}
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
                <p class="text-xs text-slate-400 truncate">{pm.user?.email || "No Email"}</p>
              </div>
            </div>

            <button
              onclick={() => removeProjectMember(pm.user_id)}
              class="flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-bold text-red-400 bg-red-400/5 border border-red-400/20 rounded-lg hover:bg-red-400/10 transition-all font-mono uppercase tracking-widest"
              title="Remove from project"
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
</style>
