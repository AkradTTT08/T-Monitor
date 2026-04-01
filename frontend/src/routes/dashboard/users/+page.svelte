<script lang="ts">
  import { onMount } from "svelte";
  import Modal from "$lib/components/Modal.svelte";
  import { API_BASE_URL } from "$lib/config";
  import { systemAlert, systemToast } from "$lib/swal-design";

  let users: any[] = [];
  let isLoading = true;
  let currentUser: any = null;

  // Detail Modal State
  let showDetailModal = false;
  let selectedUser: any = null;

  onMount(async () => {
    const userData = localStorage.getItem("monitor_user");
    if (userData) {
      currentUser = JSON.parse(userData);
      if (currentUser.role !== "admin") {
        window.location.href = "/dashboard";
        return;
      }
    }

    await fetchUsers();
  });

  async function fetchUsers() {
    isLoading = true;

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`${API_BASE_URL}/api/v1/users`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        users = await res.json();
      } else {
        systemAlert.fire({ icon: "error", title: "Load Failed", text: "Failed to load users. Ensure you have admin permissions." });
      }
    } catch (err) {
      systemAlert.fire({ icon: "error", title: "Network Error", text: "Network error fetching users." });
    } finally {
      isLoading = false;
    }
  }

  async function toggleRole(userId: number, currentRole: string) {
    const newRole = currentRole === "admin" ? "user" : "admin";

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/users/${userId}/role`,
        {
          method: "PUT",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ role: newRole }),
        },
      );

      if (res.ok) {
        await fetchUsers();
      }
    } catch (err) {
      console.error("Failed to update role", err);
    }
  }

  async function approveUser(userId: number) {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/users/${userId}/approve`,
        {
          method: "PUT",
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        systemToast.fire({ icon: "success", title: "User approved successfully." });
        await fetchUsers();
      } else {
        systemAlert.fire({ icon: "error", title: "Approval Failed", text: "Failed to approve user." });
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function disapproveUser(userId: number) {
    const result = await systemAlert.fire({
      title: "Are you sure?",
      text: "You want to disapprove and remove this user?",
      icon: "warning",
      showCancelButton: true,
      confirmButtonText: "YES, REMOVE",
    });

    if (!result.isConfirmed) return;

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/users/${userId}/disapprove`,
        {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        systemToast.fire({ icon: "success", title: "User removed." });
        await fetchUsers();
      } else {
        systemAlert.fire({ icon: "error", title: "Failed", text: "Failed to disapprove user." });
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function resetPassword(userId: number) {
    const result = await systemAlert.fire({
      title: "Reset Password?",
      text: "Reset this user's password to the default (T@monitor123)?",
      icon: "question",
      showCancelButton: true,
      confirmButtonText: "YES, RESET",
    });

    if (!result.isConfirmed) return;

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/users/${userId}/reset-password`,
        {
          method: "PUT",
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        systemAlert.fire({
          icon: "success",
          title: "Password Reset",
          text: "Reset to default: T@monitor123",
          timer: 10000,
        });
      } else {
        systemAlert.fire({ icon: "error", title: "Failed", text: "Failed to reset password." });
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function toggleBlock(userId: number) {
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `${API_BASE_URL}/api/v1/users/${userId}/block`,
        {
          method: "PUT",
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        systemToast.fire({ icon: "success", title: "Block status updated." });
        await fetchUsers();
      } else {
        const errorData = await res.json();
        systemAlert.fire({ icon: "error", title: "Action Failed", text: errorData.error || "Failed to update block status." });
      }
    } catch (err) {
      console.error(err);
    }
  }

  function openDetailModal(user: any) {
    selectedUser = user;
    showDetailModal = true;
  }
</script>

<div class="fade-in max-w-7xl mx-auto w-full overflow-hidden">
  <div class="mb-8">
    <h1
      class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase"
    >
      MANAGE_USERS
    </h1>
    <p class="text-cyan-500/80 mt-2 font-mono text-sm tracking-wide text-wrap">
      ADMINISTRATE PLATFORM ACCESS AND ASSIGNMENT OF ADMINISTRATIVE PRIVILEGES.
    </p>
  </div>


  <div
    class="bg-slate-900/60 backdrop-blur-md rounded-3xl border border-slate-700/50 shadow-[0_8px_30px_rgb(0,0,0,0.5)] overflow-x-auto w-full relative z-0"
  >
    {#if isLoading}
      <div class="flex justify-center p-12">
        <svg
          class="animate-spin h-8 w-8 text-cyan-500"
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
    {:else}
      <table class="w-full text-left border-collapse min-w-[600px]">
        <thead>
          <tr
            class="bg-slate-950/80 border-b border-slate-700/50 text-[10px] font-bold text-slate-400 uppercase tracking-widest font-mono"
          >
            <th class="p-4 pl-6">Identifier</th>
            <th class="p-4">Email_Address</th>
            <th class="p-4">Global_Role</th>
            <th class="p-4">Status</th>
            <th class="p-4">Joined_Date</th>
            <th class="p-4 pr-6 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/50 text-sm">
          {#each users as u}
            <tr
              class="hover:bg-slate-800/50 transition-colors group cursor-pointer active:bg-slate-800"
              onclick={() => openDetailModal(u)}
            >
              <td class="p-4 pl-6">
                <div class="flex items-center gap-3">
                  <div
                    class="w-8 h-8 rounded-full bg-cyan-950/50 flex shrink-0 items-center justify-center text-cyan-400 font-bold uppercase text-xs border border-cyan-500/30 shadow-[0_0_10px_rgba(6,182,212,0.2)]"
                  >
                    {u.email.charAt(0)}
                  </div>
                  <span
                    class="text-slate-400 font-mono text-xs font-bold tracking-widest max-w-[80px] md:max-w-[none] truncate"
                    >#{u.id}</span
                  >
                </div>
              </td>
              <td
                class="p-4 font-bold text-cyan-50 font-mono tracking-wide text-xs max-w-[120px] md:max-w-xs truncate"
                title={u.email}>{u.email}</td
              >
              <td class="p-4">
                <span
                  class="px-2 py-0.5 rounded border text-[10px] font-bold uppercase tracking-widest whitespace-nowrap
                  {u.role === 'admin'
                    ? 'bg-fuchsia-950/50 text-fuchsia-400 border-fuchsia-500/30 shadow-[0_0_8px_rgba(217,70,239,0.3)]'
                    : 'bg-slate-800 text-slate-400 border-slate-700'}"
                >
                  {u.role}
                </span>
              </td>
              <td class="p-4">
                {#if u.is_approved}
                  <span
                    class="px-2 py-0.5 bg-emerald-950/50 text-emerald-400 border border-emerald-500/30 shadow-[0_0_8px_rgba(52,211,153,0.3)] text-[10px] tracking-widest font-mono font-bold uppercase rounded flex items-center gap-1 w-max"
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
                      ><polyline points="20 6 9 17 4 12"></polyline></svg
                    >
                    ACTIVE
                  </span>
                {:else if u.is_blocked}
                  <span
                    class="px-2 py-0.5 bg-red-950/50 text-red-400 border border-red-500/30 shadow-[0_0_8px_rgba(239,68,68,0.3)] text-[10px] tracking-widest font-mono font-bold uppercase rounded flex items-center gap-1 w-max"
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
                      ><circle cx="12" cy="12" r="10"></circle><line
                        x1="4.93"
                        y1="4.93"
                        x2="19.07"
                        y2="19.07"
                      ></line></svg
                    >
                    BLOCKED
                  </span>
                {:else}
                  <span
                    class="px-2 py-0.5 bg-amber-950/50 text-amber-400 border border-amber-500/30 shadow-[0_0_8px_rgba(245,158,11,0.3)] text-[10px] tracking-widest font-mono font-bold uppercase rounded flex items-center gap-1 w-max animate-pulse"
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
                      ><circle cx="12" cy="12" r="10"></circle><polyline
                        points="12 6 12 12 16 14"
                       ></polyline></svg
                    >
                    PENDING
                  </span>
                {/if}
              </td>
              <td
                class="p-4 text-slate-400 text-[10px] font-mono font-bold tracking-widest uppercase whitespace-nowrap"
              >
                {new Date(u.created_at).toLocaleDateString()}
              </td>
              <td class="p-4 pr-6">
                <div
                  class="flex items-center justify-end gap-3 text-sm font-medium"
                >
                  {#if !u.is_approved && (!currentUser || currentUser.id !== u.id)}
                    <button
                    onclick={(e) => { e.stopPropagation(); approveUser(u.id); }}
                      class="text-emerald-400 hover:text-emerald-300 transition-colors bg-emerald-950/30 hover:bg-emerald-900/50 px-3 py-1.5 rounded-lg border border-emerald-500/30 hover:border-emerald-400/50 shadow-[0_0_10px_rgba(52,211,153,0.1)] hover:shadow-[0_0_15px_rgba(52,211,153,0.3)] text-[10px] font-bold font-mono tracking-widest uppercase flex items-center gap-1"
                      title="Approve this user"
                    >
                      APPROVE
                    </button>
                    <button
                    onclick={(e) => { e.stopPropagation(); disapproveUser(u.id); }}
                      class="text-red-400 hover:text-red-300 transition-colors bg-red-950/30 hover:bg-red-900/50 px-3 py-1.5 rounded-lg border border-red-500/30 hover:border-red-400/50 shadow-[0_0_10px_rgba(239,68,68,0.1)] hover:shadow-[0_0_15px_rgba(239,68,68,0.3)] text-[10px] font-bold font-mono tracking-widest uppercase flex items-center gap-1"
                      title="Disapprove (Reject) this user"
                    >
                      REJECT
                    </button>
                  {/if}

                  <button
                    onclick={(e) => { e.stopPropagation(); resetPassword(u.id); }}
                    class="text-slate-500 hover:text-cyan-400 transition-colors flex items-center gap-1 font-mono tracking-widest uppercase text-[10px] font-bold disabled:opacity-50 disabled:hover:text-slate-500"
                    title="Reset Password"
                    disabled={currentUser && currentUser.id === u.id}
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
                      class="mr-0.5"
                      ><path
                        d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"
                      ></path></svg
                    >
                    <span class="hidden md:inline">RESET</span>
                  </button>

                  <button
                    onclick={(e) => { e.stopPropagation(); toggleRole(u.id, u.role); }}
                    disabled={currentUser && currentUser.id === u.id}
                    class="transition-colors whitespace-nowrap ml-2 font-mono tracking-widest uppercase text-[10px] font-bold px-3 py-1.5 rounded-lg border
                      {currentUser && currentUser.id === u.id
                      ? 'text-slate-600 border-slate-700 bg-slate-800 cursor-not-allowed'
                      : u.role === 'admin'
                        ? 'text-slate-400 hover:text-slate-300 border-slate-700 hover:border-slate-500 bg-slate-800 hover:bg-slate-700'
                        : 'text-cyan-400 hover:text-cyan-300 border-cyan-500/30 hover:border-cyan-400/50 bg-cyan-950/30 hover:bg-cyan-900/50 shadow-[0_0_10px_rgba(6,182,212,0.1)] hover:shadow-[0_0_15px_rgba(6,182,212,0.3)]'}"
                  >
                    {u.role === "admin" ? "DEMOTE TO USER" : "PROMOTE TO ADMIN"}
                  </button>

                  <button
                    onclick={(e) => { e.stopPropagation(); toggleBlock(u.id); }}
                    disabled={currentUser &&
                      (currentUser.id === u.id || u.role === "admin")}
                    class="transition-all whitespace-nowrap ml-2 font-mono tracking-widest uppercase text-[10px] font-bold px-3 py-1.5 rounded-lg border flex items-center gap-2
                      {currentUser &&
                    (currentUser.id === u.id || u.role === 'admin')
                      ? 'text-slate-600 border-slate-700 bg-slate-800 cursor-not-allowed'
                      : u.is_blocked
                        ? 'text-emerald-400 border-emerald-500/30 bg-emerald-950/20 hover:bg-emerald-900/40 hover:border-emerald-400/50 shadow-[0_0_10px_rgba(52,211,153,0.1)]'
                        : 'text-red-400 border-red-500/30 bg-red-950/20 hover:bg-red-900/40 hover:border-red-400/50 shadow-[0_0_10px_rgba(239,68,68,0.1)]'}"
                    title={u.is_blocked ? "Unblock user" : "Block user"}
                  >
                    {#if u.is_blocked}
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
                        ><path
                          d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"
                        /></svg
                      >
                      UNBLOCK
                    {:else}
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
                        ><path
                          d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"
                        /><line x1="12" y1="9" x2="12" y2="13" /><line
                          x1="12"
                          y1="17"
                          x2="12.01"
                          y2="17"
                        /></svg
                      >
                      BLOCK
                    {/if}
                  </button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>

<!-- User Detail Modal -->
<Modal bind:open={showDetailModal} title="User Detail" maxWidth="max-w-lg">
  {#if selectedUser}
    <div class="space-y-6">
      <!-- Profile Header -->
      <div
        class="flex flex-col items-center gap-4 py-4 border-b border-slate-700/50"
      >
        <div
          class="w-24 h-24 rounded-full bg-slate-900 border-2 border-cyan-500/30 flex items-center justify-center text-3xl font-bold text-cyan-400 uppercase shadow-[0_0_20px_rgba(6,182,212,0.2)] overflow-hidden"
        >
          {#if selectedUser.profile_image_url}
            <img
              src={selectedUser.profile_image_url}
              alt={selectedUser.name}
              class="w-full h-full object-cover"
            />
          {:else}
            {selectedUser.name?.charAt(0) || selectedUser.email.charAt(0)}
          {/if}
        </div>
        <div class="text-center">
          <h2 class="text-xl font-bold text-cyan-50 font-mono tracking-wide">
            {selectedUser.name || selectedUser.email || "UNNAMED_USER"}
          </h2>
          <p class="text-slate-400 text-sm font-mono">{selectedUser.email}</p>
        </div>
      </div>

      <!-- Info Grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div class="space-y-4">
          <div>
            <p
              class="text-[10px] font-bold text-slate-500 uppercase tracking-widest font-mono mb-1"
            >
              DEPARTMENT
            </p>
            <p class="text-sm text-cyan-50 font-mono">
              {selectedUser.department || "N/A"}
            </p>
          </div>
          <div>
            <p
              class="text-[10px] font-bold text-slate-500 uppercase tracking-widest font-mono mb-1"
            >
              POSITION
            </p>
            <p class="text-sm text-cyan-50 font-mono">
              {selectedUser.position || "N/A"}
            </p>
          </div>
          <div>
            <p
              class="text-[10px] font-bold text-slate-500 uppercase tracking-widest font-mono mb-1"
            >
              PHONE
            </p>
            <p class="text-sm text-cyan-50 font-mono">
              {selectedUser.phone || "N/A"}
            </p>
          </div>
        </div>

        <div class="space-y-4">
          <div>
            <p
              class="text-[10px] font-bold text-slate-500 uppercase tracking-widest font-mono mb-1"
            >
              ROLE
            </p>
            <span
              class="px-2 py-0.5 rounded border text-[10px] font-bold uppercase tracking-widest
              {selectedUser.role === 'admin'
                ? 'bg-fuchsia-950/50 text-fuchsia-400 border-fuchsia-500/30 shadow-[0_0_8px_rgba(217,70,239,0.3)]'
                : 'bg-slate-800 text-slate-400 border-slate-700'}"
            >
              {selectedUser.role}
            </span>
          </div>
          <div>
            <p
              class="text-[10px] font-bold text-slate-500 uppercase tracking-widest font-mono mb-1"
            >
              STATUS
            </p>
            {#if selectedUser.is_approved}
              <span class="text-emerald-400 text-xs font-bold font-mono"
                >ACTIVE</span
              >
            {:else if selectedUser.is_blocked}
              <span class="text-red-400 text-xs font-bold font-mono"
                >BLOCKED</span
              >
            {:else}
              <span class="text-amber-400 text-xs font-bold font-mono"
                >PENDING</span
              >
            {/if}
          </div>
          <div>
            <p
              class="text-[10px] font-bold text-slate-500 uppercase tracking-widest font-mono mb-1"
            >
              JOINED DATE
            </p>
            <p class="text-sm text-cyan-50 font-mono">
              {new Date(selectedUser.created_at).toLocaleString()}
            </p>
          </div>
        </div>
      </div>

      <div class="pt-6 border-t border-slate-700/50 flex justify-end">
        <button
          onclick={() => (showDetailModal = false)}
          class="px-6 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl font-bold font-mono transition-colors border border-slate-700 hover:border-slate-500"
        >
          CLOSE_PROFILE
        </button>
      </div>
    </div>
  {/if}
</Modal>
