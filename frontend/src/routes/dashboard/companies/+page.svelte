<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import Modal from "$lib/components/Modal.svelte";
  import { API_BASE_URL } from "$lib/config";
  import Swal from "sweetalert2";

  let companies: any[] = [];
  let isLoading = true;

  // Create modal
  let showCreateModal = false;
  let newCompanyName = "";
  let newCompanyDesc = "";
  let newLogoFile: File | null = null;

  // Edit modal
  let showEditModal = false;
  let editingId = 0;
  let editName = "";
  let editDesc = "";
  let editLogoFile: File | null = null;

  // Delete modal
  let showDeleteModal = false;
  let deletingId = 0;
  let deletingName = "";

  let activeDropdownId: number | null = null;

  function toggleDropdown(id: number, e: Event) {
    e.stopPropagation();
    activeDropdownId = activeDropdownId === id ? null : id;
  }

  onMount(async () => {
    await fetchCompanies();
  });

  async function fetchCompanies() {
    isLoading = true;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/companies`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) companies = await res.json();
    } catch (err) {
      console.error(err);
    } finally {
      isLoading = false;
    }
  }

  async function uploadLogo(companyId: number, file: File) {
    const formData = new FormData();
    formData.append("logo", file);
    const token = localStorage.getItem("monitor_token");
    const res = await fetch(`${API_BASE_URL}/api/v1/companies/${companyId}/logo`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: formData,
    });
    return res.ok;
  }

  async function handleCreate() {
    if (!newCompanyName.trim()) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/companies`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ name: newCompanyName, description: newCompanyDesc }),
      });
      if (res.ok) {
        const company = await res.json();
        if (newLogoFile) await uploadLogo(company.id, newLogoFile);
        showCreateModal = false;
        newCompanyName = ""; newCompanyDesc = ""; newLogoFile = null;
        goto(`/dashboard/companies/${company.id}`);
      }
    } catch (err) { console.error(err); }
  }

  function openEdit(company: any) {
    editingId = company.id;
    editName = company.name;
    editDesc = company.description || "";
    editLogoFile = null;
    showEditModal = true;
    activeDropdownId = null;
  }

  async function handleEdit() {
    if (!editName.trim()) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/companies/${editingId}`, {
        method: "PUT",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ name: editName, description: editDesc }),
      });
      if (res.ok) {
        let uploadSuccess = true;
        if (editLogoFile) {
          uploadSuccess = await uploadLogo(editingId, editLogoFile);
        }

        if (uploadSuccess) {
          showEditModal = false;
          Swal.fire({ icon: "success", title: "Updated", toast: true, position: "top-end", showConfirmButton: false, timer: 2000 });
          await fetchCompanies();
        } else {
          Swal.fire({ icon: "warning", title: "Partial Update", text: "Company details updated, but logo upload failed.", toast: true, position: "top-end", showConfirmButton: false, timer: 4000 });
        }
      } else {
        const data = await res.json();
        Swal.fire({ icon: "error", title: "Update Failed", text: data.error || "Failed to update company" });
      }
    } catch (err) { 
      console.error(err); 
      Swal.fire({ icon: "error", title: "Error", text: "An error occurred while updating the company." });
    }
  }

  function openDelete(company: any) {
    deletingId = company.id;
    deletingName = company.name;
    showDeleteModal = true;
    activeDropdownId = null;
  }

  async function handleDelete() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/companies/${deletingId}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        showDeleteModal = false;
        Swal.fire({ icon: "success", title: "Deleted", text: `${deletingName} removed.`, toast: true, position: "top-end", showConfirmButton: false, timer: 2500 });
        await fetchCompanies();
      }
    } catch (err) { console.error(err); }
  }
</script>

<svelte:window on:click={() => (activeDropdownId = null)} />

<div class="fade-in">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row justify-between items-start sm:items-end mb-10 gap-4 relative z-10">
    <div>
      <h1 class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase">
        Company Monitor
      </h1>
      <p class="text-cyan-500/80 mt-2 font-mono text-sm tracking-wide">
        SELECT A COMPANY TO VIEW ITS PROJECT WORKSPACES.
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
      <span class="relative z-10">+ NEW COMPANY</span>
    </button>
  </div>

  <!-- Loading -->
  {#if isLoading}
    <div class="flex justify-center p-12">
      <svg class="animate-spin h-8 w-8 text-cyan-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
    </div>

  <!-- Empty state -->
  {:else if companies.length === 0}
    <div class="bg-slate-800/40 backdrop-blur-xl border border-slate-700/50 text-center rounded-3xl p-16 shadow-[0_8px_30px_rgb(0,0,0,0.5)] relative overflow-hidden group/empty">
      <div class="absolute inset-0 bg-cyan-900/5 opacity-0 group-hover/empty:opacity-100 transition-opacity duration-500"></div>
      <div class="inline-flex items-center justify-center w-24 h-24 rounded-full bg-slate-900 border border-cyan-500/30 mb-6 shadow-[0_0_15px_rgba(6,182,212,0.2)] relative z-10">
        <svg xmlns="http://www.w3.org/2000/svg" class="text-cyan-400 h-10 w-10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
          <polyline points="9 22 9 12 15 12 15 22"></polyline>
        </svg>
      </div>
      <h3 class="text-2xl font-bold text-cyan-50 mb-3 font-mono tracking-wide relative z-10">NO_COMPANIES_FOUND</h3>
      <p class="text-slate-400/80 max-w-md mx-auto mb-10 font-mono text-sm relative z-10">
        CREATE A COMPANY TO START ORGANIZING YOUR PROJECT WORKSPACES.
      </p>
      <button
        on:click={() => (showCreateModal = true)}
        class="bg-cyan-950/40 border border-cyan-500/50 text-cyan-400 font-bold py-3 px-8 rounded-xl hover:bg-cyan-900/60 hover:text-cyan-300 transition-colors font-mono tracking-widest relative z-10 hover:shadow-[0_0_15px_rgba(6,182,212,0.3)]"
      >
        CREATE_FIRST_COMPANY
      </button>
    </div>

  <!-- Company grid -->
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 relative z-10">
      {#each companies as company}
        <!-- Card: no overflow-hidden on wrapper so dropdown can escape the card boundary -->
        <div class="bg-slate-800/40 backdrop-blur-xl rounded-3xl border border-slate-700/50 shadow-[0_8px_30px_rgb(0,0,0,0.5)] transition-all duration-500 hover:shadow-[0_0px_30px_rgba(6,182,212,0.15)] hover:border-cyan-500/40 hover:-translate-y-1 flex flex-col group relative">
          <div class="absolute inset-0 rounded-3xl bg-gradient-to-br from-cyan-900/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"></div>

          <!-- Logo / Banner — overflow-hidden scoped to this div only -->
          <div class="h-28 rounded-t-3xl overflow-hidden relative bg-gradient-to-br from-slate-900 to-slate-800 shrink-0">
            {#if company.logo_url}
              <img
                src={company.logo_url.startsWith("http") ? company.logo_url : `${API_BASE_URL}${company.logo_url}`}
                alt={company.name}
                class="w-full h-full object-cover opacity-70 group-hover:opacity-90 transition-opacity"
              />
            {:else}
              <div class="w-full h-full flex items-center justify-center">
                <div class="w-16 h-16 rounded-2xl bg-slate-800 border border-slate-700 flex items-center justify-center text-2xl font-bold text-cyan-400 font-mono">
                  {company.name.charAt(0).toUpperCase()}
                </div>
              </div>
            {/if}
            <div class="absolute inset-0 bg-gradient-to-t from-slate-800/90 to-transparent"></div>
          </div>

          <!-- Card body -->
          <div class="p-5 flex flex-col flex-1">
            <div class="flex justify-between items-start mb-3 relative z-20">
              <div class="flex-1 min-w-0 pr-2">
                <h3 class="text-lg font-bold text-cyan-50 font-mono tracking-wide truncate">{company.name}</h3>
                <p class="text-slate-400 text-xs mt-0.5 line-clamp-1 font-mono">{company.description || "NO DESCRIPTION"}</p>
              </div>

              <!-- Dropdown — z-50 ensures it floats above everything -->
              <div class="relative shrink-0">
                <button
                  on:click={(e) => toggleDropdown(company.id, e)}
                  class="text-slate-500 hover:text-cyan-400 p-2 rounded-lg hover:bg-slate-800/80 transition-colors"
                  title="Options"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="1.5"></circle><circle cx="19" cy="12" r="1.5"></circle><circle cx="5" cy="12" r="1.5"></circle>
                  </svg>
                </button>
                {#if activeDropdownId === company.id}
                  <div
                    class="absolute right-0 top-full mt-1 w-40 bg-slate-800 rounded-xl shadow-2xl border border-slate-600 py-1 z-50"
                    on:click|stopPropagation
                    role="menu"
                    tabindex="-1"
                    on:keydown
                  >
                    <button
                      on:click={() => openEdit(company)}
                      class="w-full text-left px-4 py-2.5 text-sm font-semibold text-slate-300 hover:text-cyan-400 hover:bg-slate-700/60 transition-colors font-mono flex items-center gap-2.5"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                      </svg>
                      EDIT
                    </button>
                    <div class="h-px w-full bg-slate-700/60"></div>
                    <button
                      on:click={() => openDelete(company)}
                      class="w-full text-left px-4 py-2.5 text-sm font-semibold text-red-400 hover:text-red-300 hover:bg-red-500/10 transition-colors font-mono flex items-center gap-2.5"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14H6L5 6"/>
                        <path d="M10 11v6M14 11v6"/><path d="M9 6V4h6v2"/>
                      </svg>
                      DELETE
                    </button>
                  </div>
                {/if}
              </div>
            </div>

            <!-- Stats + Access button -->
            <div class="mt-auto pt-4 border-t border-slate-700/50 flex justify-between items-center relative z-10">
              <span class="text-[10px] font-bold bg-slate-900 border border-slate-700 group-hover:border-cyan-500/30 text-slate-400 group-hover:text-cyan-400 px-3 py-1.5 rounded-md tracking-wider font-mono transition-colors">
                {company.projects?.length || 0} Projects
              </span>
              <a
                href={`/dashboard/companies/${company.id}`}
                class="text-sm font-bold text-cyan-500 hover:text-cyan-300 flex items-center gap-1.5 font-mono tracking-wider transition-colors"
              >
                ENTER
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="group-hover:translate-x-1 transition-transform">
                  <line x1="5" y1="12" x2="19" y2="12"></line><polyline points="12 5 19 12 12 19"></polyline>
                </svg>
              </a>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Create Company Modal -->
<Modal bind:open={showCreateModal} title="Create Company">
  <form on:submit|preventDefault={handleCreate} class="space-y-4">
    <div>
      <label for="co_name" class="block text-sm font-semibold text-cyan-50 mb-1">Company Name</label>
      <input id="co_name" type="text" bind:value={newCompanyName} required
        class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm"
        placeholder="e.g. TTT Brother Co., Ltd." />
    </div>
    <div>
      <label for="co_desc" class="block text-sm font-semibold text-cyan-50 mb-1">Description</label>
      <textarea id="co_desc" rows="2" bind:value={newCompanyDesc}
        class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm resize-none"
        placeholder="Brief description..."></textarea>
    </div>
    <div>
      <label for="co_logo" class="block text-sm font-semibold text-cyan-50 mb-1">Company Logo (optional)</label>
      <input id="co_logo" type="file" accept="image/*"
        on:change={(e) => (newLogoFile = (e.target as HTMLInputElement).files?.[0] || null)}
        class="w-full px-4 py-2 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50/70 text-xs" />
    </div>
    <div class="pt-2 flex gap-3">
      <button type="button" on:click={() => (showCreateModal = false)}
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-slate-400 bg-slate-800 hover:bg-slate-700 transition-colors text-sm">Cancel</button>
      <button type="submit"
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-white bg-cyan-600 hover:bg-cyan-700 transition-colors shadow-sm text-sm">Create Company</button>
    </div>
  </form>
</Modal>

<!-- Edit Company Modal -->
<Modal bind:open={showEditModal} title="Edit Company">
  <form on:submit|preventDefault={handleEdit} class="space-y-4">
    <div>
      <label for="ed_name" class="block text-sm font-semibold text-cyan-50 mb-1">Company Name</label>
      <input id="ed_name" type="text" bind:value={editName} required
        class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm" />
    </div>
    <div>
      <label for="ed_desc" class="block text-sm font-semibold text-cyan-50 mb-1">Description</label>
      <textarea id="ed_desc" rows="2" bind:value={editDesc}
        class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm resize-none"></textarea>
    </div>
    <div>
      <label for="ed_logo" class="block text-sm font-semibold text-cyan-50 mb-1">Update Logo (optional)</label>
      <input id="ed_logo" type="file" accept="image/*"
        on:change={(e) => (editLogoFile = (e.target as HTMLInputElement).files?.[0] || null)}
        class="w-full px-4 py-2 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50/70 text-xs" />
    </div>
    <div class="pt-2 flex gap-3">
      <button type="button" on:click={() => (showEditModal = false)}
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-slate-400 bg-slate-800 hover:bg-slate-700 transition-colors text-sm">Cancel</button>
      <button type="submit"
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-white bg-cyan-600 hover:bg-cyan-700 transition-colors shadow-sm text-sm">Save Changes</button>
    </div>
  </form>
</Modal>

<!-- Delete Company Modal -->
<Modal bind:open={showDeleteModal} title="Delete Company">
  <div class="space-y-4">
    <div class="bg-amber-950/30 border border-amber-500/30 rounded-xl p-4">
      <p class="text-sm text-amber-300">
        Are you sure you want to delete <span class="font-bold text-white">{deletingName}</span>?
        All projects under this company will be kept but unlinked.
      </p>
    </div>
    <div class="flex gap-3 pt-1">
      <button type="button" on:click={() => (showDeleteModal = false)}
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-slate-400 bg-slate-800 hover:bg-slate-700 transition-colors text-sm">Cancel</button>
      <button type="button" on:click={handleDelete}
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-white bg-red-600 hover:bg-red-700 transition-colors text-sm">Delete Company</button>
    </div>
  </div>
</Modal>
