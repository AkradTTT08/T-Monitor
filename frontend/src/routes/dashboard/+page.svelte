<script lang="ts">
  import { onMount } from "svelte";
  import Modal from "$lib/components/Modal.svelte";

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
      const res = await fetch("http://localhost:5273/api/v1/projects", {
        headers: {
          Authorization: `Bearer ${token}`,
        },
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

  async function handleCreateProject() {
    if (!newProjectName) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch("http://localhost:5273/api/v1/projects", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          name: newProjectName,
          description: newProjectDesc,
          environment_variables: "{}",
        }),
      });
      if (res.ok) {
        showCreateModal = false;
        newProjectName = "";
        newProjectDesc = "";
        window.location.reload();
      }
    } catch (err) {
      console.error(err);
    }
  }

  function openEditModal(project: any) {
    editingProjectId = project.id;
    editProjectName = project.name;
    editProjectDesc = project.description || "";
    editProjectEnvVars = project.environment_variables || "{}";
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
      const res = await fetch(
        `http://localhost:5273/api/v1/projects/${editingProjectId}`,
        {
          method: "PUT",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            name: editProjectName,
            description: editProjectDesc,
            environment_variables: editProjectEnvVars,
          }),
        },
      );
      if (res.ok) {
        showEditModal = false;
        window.location.reload();
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function handleDeleteProject() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `http://localhost:5273/api/v1/projects/${deletingProjectId}`,
        {
          method: "DELETE",
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      );
      if (res.ok) {
        showDeleteModal = false;
        window.location.reload();
      }
    } catch (err) {
      console.error(err);
    }
  }
</script>

<svelte:window on:click={handleWindowClick} />

<div class="fade-in">
  <div class="flex justify-between items-end mb-8">
    <div>
      <h1 class="text-3xl font-bold text-slate-900 tracking-tight">
        Project API
      </h1>
      <p class="text-slate-500 mt-2">
        Manage your API workspaces and configurations.
      </p>
    </div>
    <button
      on:click={() => (showCreateModal = true)}
      class="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2.5 px-5 rounded-xl shadow-md shadow-blue-500/20 transition-all hover:-translate-y-0.5 flex items-center gap-2"
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
        ><line x1="12" y1="5" x2="12" y2="19"></line><line
          x1="5"
          y1="12"
          x2="19"
          y2="12"
        ></line></svg
      >
      New Project
    </button>
  </div>

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
  {:else if projects.length === 0}
    <div
      class="bg-white border text-center border-slate-200 rounded-2xl p-12 shadow-sm"
    >
      <div
        class="inline-flex items-center justify-center w-20 h-20 rounded-full bg-blue-50 mb-4"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="text-blue-500 h-10 w-10"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          ><path
            d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
          ></path></svg
        >
      </div>
      <h3 class="text-xl font-bold text-slate-800 mb-2">No Projects Found</h3>
      <p class="text-slate-500 max-w-md mx-auto mb-6">
        Create a project workspace to start grouping your APIs for health
        monitoring.
      </p>
      <button
        on:click={() => (showCreateModal = true)}
        class="bg-blue-50 text-blue-700 font-medium py-2.5 px-6 rounded-xl hover:bg-blue-100 transition-colors"
      >
        Create First Project
      </button>
    </div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {#each projects as project}
        <div
          class="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm hover-lift flex flex-col group"
        >
          <div class="flex justify-between items-start mb-4 relative">
            <div
              class="p-3 bg-blue-50 text-blue-600 rounded-xl group-hover:bg-blue-600 group-hover:text-white transition-colors"
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
                ><path
                  d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                ></path></svg
              >
            </div>
            <button
              on:click={(e) => toggleDropdown(project.id, e)}
              class="text-slate-400 hover:text-blue-600 p-2 rounded-lg hover:bg-slate-50 transition-colors"
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
                ><circle cx="12" cy="12" r="1"></circle><circle
                  cx="19"
                  cy="12"
                  r="1"
                ></circle><circle cx="5" cy="12" r="1"></circle></svg
              >
            </button>
            {#if activeDropdownId === project.id}
              <div
                class="absolute right-0 top-10 mt-1 w-36 bg-white rounded-xl shadow-lg border border-slate-100 py-1 z-20"
                on:click|stopPropagation
              >
                <button
                  on:click={() => openEditModal(project)}
                  class="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                  >Edit</button
                >
                <div class="h-px w-full bg-slate-100 my-1"></div>
                <button
                  on:click={() => openDeleteModal(project)}
                  class="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 transition-colors"
                  >Delete</button
                >
              </div>
            {/if}
          </div>
          <h3 class="text-lg font-bold text-slate-800 mb-2 truncate">
            {project.name}
          </h3>
          <p class="text-slate-500 text-sm mb-6 flex-1 line-clamp-2">
            {project.description || "No description provided."}
          </p>

          <div
            class="pt-4 border-t border-slate-100 flex justify-between items-center"
          >
            <span
              class="text-xs font-medium bg-slate-100 text-slate-600 px-2.5 py-1 rounded-full"
            >
              {project.apis?.length || 0} APIs
            </span>
            <a
              href={`/dashboard/projects/${project.id}`}
              class="text-sm font-medium text-blue-600 hover:text-blue-800 flex items-center gap-1 group-hover:underline"
            >
              View Space
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
                ><polyline points="9 18 15 12 9 6"></polyline></svg
              >
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
      <label for="name" class="block text-sm font-medium text-slate-700 mb-1"
        >Project Name</label
      >
      <input
        id="name"
        type="text"
        bind:value={newProjectName}
        required
        class="w-full px-4 py-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all text-sm"
        placeholder="e.g. E-Commerce Core API"
      />
    </div>
    <div>
      <label for="desc" class="block text-sm font-medium text-slate-700 mb-1"
        >Description</label
      >
      <textarea
        id="desc"
        rows="3"
        bind:value={newProjectDesc}
        class="w-full px-4 py-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all text-sm resize-none"
        placeholder="Brief description of this workspace..."
      ></textarea>
    </div>
    <div class="pt-2 flex gap-3">
      <button
        type="button"
        on:click={() => (showCreateModal = false)}
        class="flex-1 px-4 py-3 rounded-xl font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 transition-colors"
        >Cancel</button
      >
      <button
        type="submit"
        class="flex-1 px-4 py-3 rounded-xl font-medium text-white bg-blue-600 hover:bg-blue-700 transition-colors"
        >Create</button
      >
    </div>
  </form>
</Modal>

<!-- Edit Project Modal -->
<Modal bind:open={showEditModal} title="Edit Project">
  <form on:submit|preventDefault={handleEditProject} class="space-y-5">
    <div>
      <label
        for="edit_name"
        class="block text-sm font-medium text-slate-700 mb-1"
        >Project Name</label
      >
      <input
        id="edit_name"
        type="text"
        bind:value={editProjectName}
        required
        class="w-full px-4 py-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all text-sm"
      />
    </div>
    <div>
      <label
        for="edit_desc"
        class="block text-sm font-medium text-slate-700 mb-1">Description</label
      >
      <textarea
        id="edit_desc"
        rows="3"
        bind:value={editProjectDesc}
        class="w-full px-4 py-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all text-sm resize-none"
      ></textarea>
    </div>
    <div class="pt-2 flex gap-3">
      <button
        type="button"
        on:click={() => (showEditModal = false)}
        class="flex-1 px-4 py-3 rounded-xl font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 transition-colors"
        >Cancel</button
      >
      <button
        type="submit"
        class="flex-1 px-4 py-3 rounded-xl font-medium text-white bg-blue-600 hover:bg-blue-700 transition-colors"
        >Save Changes</button
      >
    </div>
  </form>
</Modal>

<!-- Delete Project Modal -->
<Modal bind:open={showDeleteModal} title="Delete Project">
  <div class="space-y-5">
    <p class="text-slate-600 text-sm">
      Are you sure you want to delete <span class="font-bold text-slate-900"
        >{deletingProjectName}</span
      >? This action cannot be undone and will remove all associated APIs and
      monitoring logs.
    </p>
    <div class="pt-2 flex gap-3">
      <button
        type="button"
        on:click={() => (showDeleteModal = false)}
        class="flex-1 px-4 py-3 rounded-xl font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 transition-colors"
        >Cancel</button
      >
      <button
        type="button"
        on:click={handleDeleteProject}
        class="flex-1 px-4 py-3 rounded-xl font-medium text-white bg-red-600 hover:bg-red-700 transition-colors"
        >Delete Workspace</button
      >
    </div>
  </div>
</Modal>
