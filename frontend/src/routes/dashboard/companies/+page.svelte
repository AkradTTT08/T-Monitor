<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import Modal from "$lib/components/Modal.svelte";
  import { API_BASE_URL } from "$lib/config";
  import Swal from "sweetalert2";
  import { systemAlert, systemToast } from "$lib/swal-design";

  let companies = $state<any[]>([]);
  let isLoading = $state(true);

  // Create modal
  let showCreateModal = $state(false);
  let newCompanyName = $state("");
  let newCompanyDesc = $state("");
  let newLogoFile = $state<File | null>(null);

  // Edit modal
  let showEditModal = $state(false);
  let editingId = $state(0);
  let editName = $state("");
  let editDesc = $state("");
  let editLogoFile = $state<File | null>(null);

  // Delete modal
  let showDeleteModal = $state(false);
  let deletingId = $state("");
  let deletingName = $state("");

  let activeDropdownId = $state<string | null>(null);
  let showInviteModal = $state(false);
  let invitingCompanyId = $state("");
  let invitingCompanyName = $state("");
  let inviteEmail = $state("");
  let isInviting = $state(false);
  let userSuggestions = $state<any[]>([]);
  let showSuggestions = $state(false);
  let searchTimeout: any;

  // Member list modal
  let showMembersModal = $state(false);
  let viewingMembersCompany = $state<any>(null);

  async function handleSearchUsers(e: Event) {
    const q = (e.target as HTMLInputElement).value;
    inviteEmail = q;

    if (searchTimeout) clearTimeout(searchTimeout);

    if (q.length < 2) {
      userSuggestions = [];
      showSuggestions = false;
      return;
    }

    searchTimeout = setTimeout(async () => {
      try {
        const token = localStorage.getItem("monitor_token");
        const res = await fetch(
          `${API_BASE_URL}/api/v1/users/search?q=${encodeURIComponent(q)}`,
          {
            headers: { Authorization: `Bearer ${token}` },
          },
        );
        if (res.ok) {
          userSuggestions = await res.json();
          showSuggestions = userSuggestions.length > 0;
        }
      } catch (err) {
        console.error(err);
      }
    }, 300);
  }

  function selectUser(user: any) {
    inviteEmail = user.email;
    showSuggestions = false;
    userSuggestions = [];
  }

  function toggleDropdown(id: string, e: Event) {
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
      if (res.ok) {
        companies = await res.json();
      }
    } catch (err) {
      console.error(err);
    } finally {
      isLoading = false;
    }
  }

  async function uploadLogo(companyId: string, file: File) {
    const formData = new FormData();
    formData.append("logo", file);
    const token = localStorage.getItem("monitor_token");
    const res = await fetch(
      `${API_BASE_URL}/api/v1/companies/${companyId}/logo`,
      {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: formData,
      },
    );
    return res.ok;
  }

  async function handleCreate() {
    if (!newCompanyName.trim()) return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/companies`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          name: newCompanyName,
          description: newCompanyDesc,
        }),
      });
      if (res.ok) {
        const company = await res.json();
        if (newLogoFile) await uploadLogo(company.id, newLogoFile);
        showCreateModal = false;
        newCompanyName = "";
        newCompanyDesc = "";
        newLogoFile = null;
        goto(`/dashboard/companies/${company.id}`);
      }
    } catch (err) {
      console.error(err);
    }
  }

  function openInvite(company: any) {
    invitingCompanyId = company.id;
    invitingCompanyName = company.name;
    inviteEmail = "";
    showInviteModal = true;
    activeDropdownId = null;
  }

  async function handleInvite() {
    if (!inviteEmail.trim()) return;
    isInviting = true;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/companies/${invitingCompanyId}/invite`,
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ email: inviteEmail }),
        },
      );
      const data = await res.json();
      if (res.ok) {
        showInviteModal = false;
        systemToast.fire({
          icon: "success",
          title: "Invitation Sent",
          text: `Invitation sent to ${inviteEmail}`,
          timer: 3000,
        });
      } else {
        systemAlert.fire({
          icon: "error",
          title: "Invite Failed",
          text: data.error || "Failed to send invitation",
        });
      }
    } catch (err) {
      console.error(err);
      systemAlert.fire({
        icon: "error",
        title: "Error",
        text: "An error occurred while sending the invitation.",
      });
    } finally {
      isInviting = false;
    }
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
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ name: editName, description: editDesc }),
      });
      if (res.ok) {
        let uploadSuccess = true;
        if (editLogoFile) {
          uploadSuccess = await uploadLogo(editingId, editLogoFile);
        }

        if (uploadSuccess) {
          showEditModal = false;
          systemToast.fire({ icon: "success", title: "Updated", timer: 2000 });
          await fetchCompanies();
        } else {
          systemAlert.fire({
            icon: "warning",
            title: "Partial Update",
            text: "Company details updated, but logo upload failed.",
            timer: 4000,
          });
        }
      } else {
        const data = await res.json();
        systemAlert.fire({
          icon: "error",
          title: "Update Failed",
          text: data.error || "Failed to update company",
        });
      }
    } catch (err) {
      console.error(err);
      systemAlert.fire({
        icon: "error",
        title: "Error",
        text: "An error occurred while updating the company.",
      });
    }
  }

  let user = $state<any>(null);

  function openDelete(company: any) {
    deletingId = company.id;
    deletingName = company.name;
    showDeleteModal = true;
    activeDropdownId = null;
  }

  async function handleDelete() {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/companies/${deletingId}`,
        {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        showDeleteModal = false;
        systemToast.fire({
          icon: "success",
          title: "Deleted",
          text: `${deletingName} removed.`,
          timer: 2500,
        });
        await fetchCompanies();
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function openMembersModal(company: any) {
    viewingMembersCompany = company;
    showMembersModal = true;

    // Attempt re-fetch to ensure relations (Owner, Members.User) are loaded
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/companies/${company.id}`,
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        const data = await res.json();
        // If we found the company in the array, update it too for consistency
        const idx = companies.findIndex((c) => c.id === company.id);
        if (idx !== -1) {
          companies[idx] = data;
        }
        viewingMembersCompany = data;
      }
    } catch (err) {
      console.error("Failed to re-fetch company members:", err);
    }
  }

  async function removeCompanyMember(memberId: string) {
    const confirm = await Swal.fire({
      title: "Are you sure?",
      text: "This user will lose access to the company and all its projects.",
      icon: "warning",
      showCancelButton: true,
      confirmButtonText: "Yes, remove",
      cancelButtonText: "Cancel",
      background: "#0f172a",
      color: "#f1f5f9",
      customClass: {
        confirmButton:
          "bg-red-600 hover:bg-red-700 text-white font-bold py-2 px-6 rounded-xl ml-3",
        cancelButton:
          "bg-slate-700 hover:bg-slate-600 text-white font-bold py-2 px-6 rounded-xl",
      },
      buttonsStyling: false,
    });

    if (!confirm.isConfirmed) return;

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/companies/${viewingMembersCompany.id}/members/${memberId}`,
        {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
        },
      );

      if (res.ok) {
        systemToast.fire({
          icon: "success",
          title: "Member removed",
          timer: 2000,
        });
        // Refresh the viewing company data
        await openMembersModal(viewingMembersCompany);
      } else {
        const data = await res.json();
        systemAlert.fire({
          icon: "error",
          title: "Failed",
          text: data.error || "Could not remove member",
        });
      }
    } catch (err) {
      console.error(err);
    }
  }

  onMount(() => {
    const userData = localStorage.getItem("monitor_user");
    if (userData) user = JSON.parse(userData);
  });
</script>

<svelte:window onclick={() => (activeDropdownId = null)} />

<div class="fade-in">
  <!-- Header -->
  <div
    class="flex flex-col sm:flex-row justify-between items-start sm:items-end mb-10 gap-4 relative z-10"
  >
    <div>
      <h1
        class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase"
      >
        Company Monitor
      </h1>
      <p class="text-cyan-500/80 mt-2 font-mono text-sm tracking-wide">
        SELECT A COMPANY TO VIEW ITS PROJECT WORKSPACES.
      </p>
    </div>
    <button
      onclick={() => (showCreateModal = true)}
      class="bg-slate-900 border border-cyan-500/50 text-cyan-400 hover:bg-cyan-950/50 hover:border-cyan-400 hover:text-cyan-300 font-bold py-2.5 px-6 rounded-xl shadow-[0_0_15px_rgba(6,182,212,0.3)] hover:shadow-[0_0_25px_rgba(6,182,212,0.5)] transition-all flex items-center gap-2 group transform hover:-translate-y-0.5 font-mono tracking-wider overflow-hidden relative"
    >
      <div
        class="absolute inset-0 w-full h-full bg-cyan-400/10 -translate-x-full group-hover:animate-[shimmer_1.5s_infinite] skew-x-12"
      ></div>
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="18"
        height="18"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.5"
        stroke-linecap="round"
        stroke-linejoin="round"
        class="relative z-10 group-hover:rotate-90 transition-transform duration-300"
      >
        <line x1="12" y1="5" x2="12" y2="19"></line><line
          x1="5"
          y1="12"
          x2="19"
          y2="12"
        ></line>
      </svg>
      <span class="relative z-10">+ NEW COMPANY</span>
    </button>
  </div>

  <!-- Loading -->
  {#if isLoading}
    <div class="flex justify-center p-12">
      <svg
        class="animate-spin h-8 w-8 text-cyan-500"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle
          class="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          stroke-width="4"
        ></circle>
        <path
          class="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
        ></path>
      </svg>
    </div>

    <!-- Empty state -->
  {:else if companies.length === 0}
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
        >
          <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
          <polyline points="9 22 9 12 15 12 15 22"></polyline>
        </svg>
      </div>
      <h3
        class="text-2xl font-bold text-cyan-50 mb-3 font-mono tracking-wide relative z-10"
      >
        NO_COMPANIES_FOUND
      </h3>
      <p
        class="text-slate-400/80 max-w-md mx-auto mb-10 font-mono text-sm relative z-10"
      >
        CREATE A COMPANY TO START ORGANIZING YOUR PROJECT WORKSPACES.
      </p>
      <button
        onclick={() => (showCreateModal = true)}
        class="bg-cyan-950/40 border border-cyan-500/50 text-cyan-400 font-bold py-3 px-8 rounded-xl hover:bg-cyan-900/60 hover:text-cyan-300 transition-colors font-mono tracking-widest relative z-10 hover:shadow-[0_0_15px_rgba(6,182,212,0.3)]"
      >
        CREATE_FIRST_COMPANY
      </button>
    </div>

    <!-- Company grid -->
  {:else}
    <div
      class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 relative z-10"
    >
      {#each companies as company}
        <!-- Card: no overflow-hidden on wrapper so dropdown can escape the card boundary -->
        <div
          class="bg-slate-800/40 backdrop-blur-xl rounded-3xl border border-slate-700/50 shadow-[0_8px_30px_rgb(0,0,0,0.5)] transition-all duration-500 hover:shadow-[0_0px_30px_rgba(6,182,212,0.15)] hover:border-cyan-500/40 hover:-translate-y-1 flex flex-col group relative"
        >
          <div
            class="absolute inset-0 rounded-3xl bg-gradient-to-br from-cyan-900/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"
          ></div>

          <!-- Logo / Banner — overflow-hidden scoped to this div only -->
          <div
            class="h-28 rounded-t-3xl overflow-hidden relative bg-gradient-to-br from-slate-900 to-slate-800 shrink-0"
          >
            {#if company.logo_url}
              <img
                src={company.logo_url.startsWith("http")
                  ? company.logo_url
                  : `${API_BASE_URL}${company.logo_url}`}
                alt={company.name}
                class="w-full h-full object-cover opacity-70 group-hover:opacity-90 transition-opacity"
              />
            {:else}
              <div class="w-full h-full flex items-center justify-center">
                <div
                  class="w-16 h-16 rounded-2xl bg-slate-800 border border-slate-700 flex items-center justify-center text-2xl font-bold text-cyan-400 font-mono"
                >
                  {company.name.charAt(0).toUpperCase()}
                </div>
              </div>
            {/if}
            <div
              class="absolute inset-0 bg-gradient-to-t from-slate-800/90 to-transparent"
            ></div>
          </div>

          <!-- Card body -->
          <div class="p-5 flex flex-col flex-1">
            <div class="flex justify-between items-start mb-3 relative z-20">
              <div class="flex-1 min-w-0 pr-2">
                <h3
                  class="text-lg font-bold text-cyan-50 font-mono tracking-wide truncate"
                >
                  {company.name}
                </h3>
                <p class="text-slate-400 text-xs mt-0.5 line-clamp-1 font-mono">
                  {company.description || "NO DESCRIPTION"}
                </p>
              </div>

              <!-- Dropdown — z-50 ensures it floats above everything -->
              <div class="relative shrink-0">
                <button
                  onclick={(e) => toggleDropdown(company.id, e)}
                  class="text-slate-500 hover:text-cyan-400 p-2 rounded-lg hover:bg-slate-800/80 transition-colors"
                  title="Options"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="18"
                    height="18"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <circle cx="12" cy="12" r="1.5"></circle><circle
                      cx="19"
                      cy="12"
                      r="1.5"
                    ></circle><circle cx="5" cy="12" r="1.5"></circle>
                  </svg>
                </button>
                {#if activeDropdownId === company.id}
                  <div
                    class="absolute right-0 top-full mt-1 w-40 bg-slate-800 rounded-xl shadow-2xl border border-slate-600 py-1 z-50"
                    onclick={(e) => e.stopPropagation()}
                    onkeydown={(e) => e.stopPropagation()}
                    role="menu"
                    tabindex="-1"
                  >
                    {#if user && (user.id === company.user_id || user.role === "admin")}
                      <button
                        onclick={() => openEdit(company)}
                        class="w-full text-left px-4 py-2.5 text-sm font-semibold text-slate-300 hover:text-cyan-400 hover:bg-slate-700/60 transition-colors font-mono flex items-center gap-2.5"
                      >
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          width="13"
                          height="13"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          stroke-width="2"
                        >
                          <path
                            d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"
                          />
                          <path
                            d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"
                          />
                        </svg>
                        EDIT
                      </button>
                      <div class="h-px w-full bg-slate-700/60"></div>
                      <button
                        onclick={() => openInvite(company)}
                        class="w-full text-left px-4 py-2.5 text-sm font-semibold text-slate-300 hover:text-cyan-400 hover:bg-slate-700/60 transition-colors font-mono flex items-center gap-2.5"
                      >
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          width="13"
                          height="13"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          stroke-width="2"
                        >
                          <path
                            d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"
                          /><circle cx="8.5" cy="7" r="4" /><line
                            x1="20"
                            y1="8"
                            x2="20"
                            y2="14"
                          /><line x1="17" y1="11" x2="23" y2="11" />
                        </svg>
                        ADD MEMBER
                      </button>
                      <div class="h-px w-full bg-slate-700/60"></div>
                      <button
                        onclick={() => openDelete(company)}
                        class="w-full text-left px-4 py-2.5 text-sm font-semibold text-red-400 hover:text-red-300 hover:bg-red-500/10 transition-colors font-mono flex items-center gap-2.5"
                      >
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          width="13"
                          height="13"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          stroke-width="2"
                        >
                          <polyline points="3 6 5 6 21 6" /><path
                            d="M19 6l-1 14H6L5 6"
                          />
                          <path d="M10 11v6M14 11v6" /><path d="M9 6V4h6v2" />
                        </svg>
                        DELETE
                      </button>
                    {:else}
                      <div
                        class="px-4 py-3 text-[10px] font-bold text-slate-500 uppercase tracking-tighter text-center italic"
                      >
                        View Only
                      </div>
                    {/if}
                  </div>
                {/if}
              </div>
            </div>

            <!-- Members + Stats -->
            <div
              class="mt-auto pt-4 border-t border-slate-700/50 flex justify-between items-center relative z-10"
            >
              <button
                class="flex items-center -space-x-1.5 overflow-hidden hover:opacity-80 transition-opacity p-1 -m-1 rounded-lg"
                onclick={() => openMembersModal(company)}
                title="View all members"
              >
                <!-- Owner first -->
                {#if company.owner && company.owner.id}
                  <div
                    class="w-7 h-7 rounded-full border-2 border-slate-800 bg-slate-900 flex items-center justify-center overflow-hidden shrink-0 ring-2 ring-cyan-500/20"
                    title={`Owner: ${company.owner.name || company.owner.email}`}
                  >
                    {#if company.owner.profile_image_url}
                      <img
                        src={company.owner.profile_image_url.startsWith(
                          "http",
                        ) || company.owner.profile_image_url.startsWith("data:")
                          ? company.owner.profile_image_url
                          : `${API_BASE_URL}${company.owner.profile_image_url}`}
                        alt=""
                        class="w-full h-full object-cover"
                      />
                    {:else}
                      <span class="text-[10px] font-bold text-cyan-400"
                        >{(company.owner.name || company.owner.email || "O")
                          .charAt(0)
                          .toUpperCase()}</span
                      >
                    {/if}
                  </div>
                {:else if company.user_id}
                  <!-- Fallback if owner record not preloaded but we have user_id -->
                  <div
                    class="w-7 h-7 rounded-full border-2 border-slate-800 bg-slate-900 flex items-center justify-center shrink-0"
                  >
                    <span class="text-[10px] font-bold text-slate-500">?</span>
                  </div>
                {/if}

                <!-- Members -->
                {#each (company.members || []).slice(0, 4) as member}
                  {#if member.user}
                    <div
                      class="w-7 h-7 rounded-full border-2 border-slate-800 bg-slate-900 flex items-center justify-center overflow-hidden shrink-0"
                      title={member.user.name || member.user.email}
                    >
                      {#if member.user.profile_image_url}
                        <img
                          src={member.user.profile_image_url.startsWith(
                            "http",
                          ) || member.user.profile_image_url.startsWith("data:")
                            ? member.user.profile_image_url
                            : `${API_BASE_URL}${member.user.profile_image_url}`}
                          alt=""
                          class="w-full h-full object-cover"
                        />
                      {:else}
                        <span class="text-[10px] font-bold text-slate-400"
                          >{(member.user.name || member.user.email || "U")
                            .charAt(0)
                            .toUpperCase()}</span
                        >
                      {/if}
                    </div>
                  {/if}
                {/each}

                {#if (company.members || []).length > 4}
                  <div
                    class="w-7 h-7 rounded-full border-2 border-slate-800 bg-slate-800 flex items-center justify-center shrink-0 text-[8px] font-bold text-slate-500"
                  >
                    +{(company.members || []).length - 4}
                  </div>
                {/if}

                {#if !company.owner && (!company.members || company.members.length === 0)}
                  <span class="text-[10px] font-mono text-slate-600 pl-2"
                    >NO MEMBERS</span
                  >
                {/if}
              </button>

              <div class="flex items-center gap-3">
                <span
                  class="text-[10px] font-bold bg-slate-900 border border-slate-700 group-hover:border-cyan-500/30 text-slate-400 group-hover:text-cyan-400 px-3 py-1.5 rounded-md tracking-wider font-mono transition-colors"
                >
                  {company.projects?.length || 0} Projects
                </span>
                <a
                  href={`/dashboard/companies/${company.id}`}
                  class="text-sm font-bold text-cyan-500 hover:text-cyan-300 flex items-center gap-1.5 font-mono tracking-wider transition-colors"
                >
                  ENTER
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    class="group-hover:translate-x-1 transition-transform"
                  >
                    <line x1="5" y1="12" x2="19" y2="12"></line><polyline
                      points="12 5 19 12 12 19"
                    ></polyline>
                  </svg>
                </a>
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Member List Modal -->
</div>
{#if showMembersModal && viewingMembersCompany}
  <Modal
    bind:open={showMembersModal}
    title="COMPANY_MEMBERS"
    maxWidth="max-w-xl"
  >
    <div class="space-y-6">
      <!-- Owner Section -->
      <div>
        <h4
          class="text-[10px] font-bold text-cyan-500 uppercase tracking-[0.2em] mb-4 flex items-center gap-2"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
            ><path
              d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"
            /></svg
          >
          COMPANY_OWNER
        </h4>
        {#if viewingMembersCompany.owner}
          <div
            class="flex items-center gap-4 bg-slate-900/50 border border-cyan-500/20 p-4 rounded-2xl"
          >
            <div
              class="w-12 h-12 rounded-full border-2 border-cyan-500/30 bg-slate-800 flex items-center justify-center overflow-hidden shrink-0"
            >
              {#if viewingMembersCompany.owner.profile_image_url}
                <img
                  src={viewingMembersCompany.owner.profile_image_url.startsWith(
                    "http",
                  ) ||
                  viewingMembersCompany.owner.profile_image_url.startsWith(
                    "data:",
                  )
                    ? viewingMembersCompany.owner.profile_image_url
                    : `${API_BASE_URL}${viewingMembersCompany.owner.profile_image_url}`}
                  alt=""
                  class="w-full h-full object-cover"
                />
              {:else}
                <span class="text-lg font-bold text-cyan-400"
                  >{(
                    viewingMembersCompany.owner.name ||
                    viewingMembersCompany.owner.email ||
                    "O"
                  )
                    .charAt(0)
                    .toUpperCase()}</span
                >
              {/if}
            </div>
            <div class="min-w-0">
              <p class="text-sm font-bold text-slate-100 font-mono truncate">
                {viewingMembersCompany.owner.name || viewingMembersCompany.owner.email || "UNNAMED_OWNER"}
              </p>
              <p class="text-xs text-slate-400 font-mono truncate">
                {viewingMembersCompany.owner.email}
              </p>
            </div>
          </div>
        {:else}
          <p class="text-xs text-slate-500 italic font-mono p-4">
            OWNER_DATA_NOT_FOUND
          </p>
        {/if}
      </div>

      <!-- Members Section -->
      <div>
        <h4
          class="text-[10px] font-bold text-slate-400 uppercase tracking-[0.2em] mb-4 flex items-center gap-2"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
            ><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" /><circle
              cx="9"
              cy="7"
              r="4"
            /><path d="M23 21v-2a4 4 0 0 0-3-3.87" /><path
              d="M16 3.13a4 4 0 0 1 0 7.75"
            /></svg
          >
          TEAM_MEMBERS ({viewingMembersCompany.members?.length || 0})
        </h4>
        <div class="space-y-3 flex-1 pr-2">
          {#each viewingMembersCompany.members || [] as member}
            <div
              class="flex items-center gap-4 bg-slate-800/30 border border-slate-700/50 p-3 rounded-xl hover:border-slate-600 transition-colors"
            >
              <div
                class="w-10 h-10 rounded-full border border-slate-700 bg-slate-900 flex items-center justify-center overflow-hidden shrink-0"
              >
                {#if member.user?.profile_image_url}
                  <img
                    src={member.user.profile_image_url.startsWith("http") ||
                    member.user.profile_image_url.startsWith("data:")
                      ? member.user.profile_image_url
                      : `${API_BASE_URL}${member.user.profile_image_url}`}
                    alt=""
                    class="w-full h-full object-cover"
                  />
                {:else}
                  <span class="text-sm font-bold text-slate-400"
                    >{(member.user?.name || member.user?.email || "U")
                      .charAt(0)
                      .toUpperCase()}</span
                  >
                {/if}
              </div>
              <div class="min-w-0">
                <p class="text-sm font-bold text-slate-200 font-mono truncate">
                  {member.user?.name || member.user?.email || "UNNAMED_MEMBER"}
                </p>
                <p class="text-xs text-slate-500 font-mono truncate">
                  {member.user?.email}
                </p>
              </div>
              <div class="ml-auto flex items-center gap-3">
                <span
                  class="text-[9px] font-bold px-2 py-0.5 rounded bg-slate-800 text-slate-500 border border-slate-700 uppercase tracking-tighter"
                >
                  {member.role || "MEMBER"}
                </span>

                {#if user && (user.id === viewingMembersCompany.user_id || user.role === "admin")}
                  <button
                    onclick={() => removeCompanyMember(member.id)}
                    class="p-1.5 text-slate-500 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-all"
                    title="Remove Member"
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
                    >
                      <path d="M3 6h18" />
                      <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
                      <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
                    </svg>
                  </button>
                {/if}
              </div>
            </div>
          {:else}
            <div
              class="text-center py-10 bg-slate-900/20 rounded-2xl border border-dashed border-slate-700"
            >
              <p
                class="text-xs text-slate-500 font-mono uppercase tracking-widest"
              >
                NO_OTHER_MEMBERS_YET
              </p>
            </div>
          {/each}
        </div>
      </div>

      <div
        class="mt-auto pt-6 border-t border-slate-800/50 flex justify-between items-center bg-slate-900/50 -mx-6 -mb-6 px-6 py-4"
      >
        <details class="group flex-1">
          <summary
            class="text-[9px] text-slate-600 cursor-pointer hover:text-slate-400 font-mono flex items-center gap-2 uppercase tracking-tighter"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="10"
              height="10"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="group-open:rotate-90 transition-transform"
              ><polyline points="9 18 15 12 9 6"></polyline></svg
            >
            RAW_JSON_INSPEC (ID:{viewingMembersCompany.id})
          </summary>
          <div class="mt-2 p-3 bg-black/60 rounded-xl border border-slate-800">
            <pre
              class="text-[9px] text-cyan-400/60 custom-scrollbar font-mono leading-tight whitespace-pre-wrap break-all max-h-24 overflow-y-auto">
                {JSON.stringify(viewingMembersCompany, null, 2)}
              </pre>
          </div>
        </details>

        <button
          onclick={() => (showMembersModal = false)}
          class="ml-4 px-8 py-3 bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-bold rounded-2xl shadow-lg shadow-cyan-900/20 transition-all active:scale-95 uppercase tracking-widest font-mono"
        >
          CLOSE_MODAL
        </button>
      </div>
    </div>
  </Modal>
{/if}
<!-- Create Company Modal -->
<Modal bind:open={showCreateModal} title="Create Company" maxWidth="max-w-lg">
  <form
    onsubmit={(e) => {
      e.preventDefault();
      handleCreate();
    }}
    class="space-y-4"
  >
    <div>
      <label for="co_name" class="block text-sm font-semibold text-cyan-50 mb-1"
        >Company Name</label
      >
      <input
        id="co_name"
        type="text"
        bind:value={newCompanyName}
        required
        class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm"
        placeholder="e.g. TTT Brother Co., Ltd."
      />
    </div>
    <div>
      <label for="co_desc" class="block text-sm font-semibold text-cyan-50 mb-1"
        >Description</label
      >
      <textarea
        id="co_desc"
        rows="2"
        bind:value={newCompanyDesc}
        class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm resize-none"
        placeholder="Brief description..."
      ></textarea>
    </div>
    <div>
      <label for="co_logo" class="block text-sm font-semibold text-cyan-50 mb-1"
        >Company Logo (optional)</label
      >
      <input
        id="co_logo"
        type="file"
        accept="image/*"
        onchange={(e) =>
          (newLogoFile = (e.target as HTMLInputElement).files?.[0] || null)}
        class="w-full px-4 py-2 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50/70 text-xs"
      />
    </div>
    <div class="pt-2 flex gap-3">
      <button
        type="button"
        onclick={() => (showCreateModal = false)}
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-slate-400 bg-slate-800 hover:bg-slate-700 transition-colors text-sm"
        >Cancel</button
      >
      <button
        type="submit"
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-white bg-cyan-600 hover:bg-cyan-700 transition-colors shadow-sm text-sm"
        >Create Company</button
      >
    </div>
  </form>
</Modal>

<!-- Edit Company Modal -->
<Modal bind:open={showEditModal} title="Edit Company" maxWidth="max-w-lg">
  <form
    onsubmit={(e) => {
      e.preventDefault();
      handleEdit();
    }}
    class="space-y-4"
  >
    <div>
      <label for="ed_name" class="block text-sm font-semibold text-cyan-50 mb-1"
        >Company Name</label
      >
      <input
        id="ed_name"
        type="text"
        bind:value={editName}
        required
        class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm"
      />
    </div>
    <div>
      <label for="ed_desc" class="block text-sm font-semibold text-cyan-50 mb-1"
        >Description</label
      >
      <textarea
        id="ed_desc"
        rows="2"
        bind:value={editDesc}
        class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm resize-none"
      ></textarea>
    </div>
    <div>
      <label for="ed_logo" class="block text-sm font-semibold text-cyan-50 mb-1"
        >Update Logo (optional)</label
      >
      <input
        id="ed_logo"
        type="file"
        accept="image/*"
        onchange={(e) =>
          (editLogoFile = (e.target as HTMLInputElement).files?.[0] || null)}
        class="w-full px-4 py-2 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50/70 text-xs"
      />
    </div>
    <div class="pt-2 flex gap-3">
      <button
        type="button"
        onclick={() => (showEditModal = false)}
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-slate-400 bg-slate-800 hover:bg-slate-700 transition-colors text-sm"
        >Cancel</button
      >
      <button
        type="submit"
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-white bg-cyan-600 hover:bg-cyan-700 transition-colors shadow-sm text-sm"
        >Save Changes</button
      >
    </div>
  </form>
</Modal>

<!-- Delete Company Modal -->
<Modal bind:open={showDeleteModal} title="Delete Company">
  <div class="space-y-4">
    <div class="bg-amber-950/30 border border-amber-500/30 rounded-xl p-4">
      <p class="text-sm text-amber-300">
        Are you sure you want to delete <span class="font-bold text-white"
          >{deletingName}</span
        >? All projects under this company will be kept but unlinked.
      </p>
    </div>
    <div class="flex gap-3 pt-1">
      <button
        type="button"
        onclick={() => (showDeleteModal = false)}
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-slate-400 bg-slate-800 hover:bg-slate-700 transition-colors text-sm"
        >Cancel</button
      >
      <button
        type="button"
        onclick={handleDelete}
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-white bg-red-600 hover:bg-red-700 transition-colors text-sm"
        >Delete Company</button
      >
    </div>
  </div>
</Modal>

<!-- Invite Member Modal -->
<Modal
  bind:open={showInviteModal}
  title="Invite Member"
  maxWidth="max-w-2xl"
  overflowVisible={true}
>
  <form
    onsubmit={(e) => {
      e.preventDefault();
      handleInvite();
    }}
    class="space-y-4"
  >
    <div class="bg-cyan-950/20 border border-cyan-500/20 rounded-xl p-4 mb-2">
      <p class="text-sm text-cyan-400 font-mono">
        Inviting to join <span class="font-bold text-white uppercase"
          >{invitingCompanyName}</span
        >
      </p>
    </div>
    <div class="relative">
      <label
        for="inv_email"
        class="block text-sm font-semibold text-cyan-50 mb-1"
        >User Email Address</label
      >
      <input
        id="inv_email"
        type="email"
        bind:value={inviteEmail}
        oninput={handleSearchUsers}
        onfocus={() => {
          if (userSuggestions.length > 0) showSuggestions = true;
        }}
        required
        class="w-full px-4 py-2.5 rounded-xl border border-slate-700/50 bg-slate-900/60 text-cyan-50 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 text-sm"
        placeholder="Type name or email..."
        autocomplete="off"
      />

      {#if showSuggestions}
        <div
          class="absolute z-[60] w-full mt-1 bg-slate-900 border border-slate-700 rounded-xl shadow-2xl max-h-48 overflow-y-auto overflow-x-hidden custom-scrollbar"
        >
          {#each userSuggestions as user}
            <button
              type="button"
              onclick={() => selectUser(user)}
              class="w-full text-left px-4 py-2.5 hover:bg-cyan-900/40 transition-colors border-b border-slate-800/50 last:border-0 group"
            >
              <div class="flex flex-col">
                <span
                  class="text-sm font-bold text-cyan-400 group-hover:text-cyan-300"
                  >{user.name || user.email || "No Name"}</span
                >
                <span class="text-xs text-slate-400 font-mono"
                  >{user.email}</span
                >
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>
    <div class="pt-2 flex gap-3">
      <button
        type="button"
        onclick={() => (showInviteModal = false)}
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-slate-400 bg-slate-800 hover:bg-slate-700 transition-colors text-sm"
        >Cancel</button
      >
      <button
        type="submit"
        disabled={isInviting}
        class="flex-1 px-4 py-2.5 rounded-xl font-semibold text-white bg-cyan-600 hover:bg-cyan-700 transition-colors shadow-sm text-sm disabled:opacity-50"
      >
        {isInviting ? "Sending..." : "Send Invitation"}
      </button>
    </div>
  </form>
</Modal>

<style>
  .custom-scrollbar::-webkit-scrollbar {
    width: 4px;
  }
  .custom-scrollbar::-webkit-scrollbar-track {
    background: transparent;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb {
    background: #334155;
    border-radius: 10px;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb:hover {
    background: #475569;
  }
</style>
