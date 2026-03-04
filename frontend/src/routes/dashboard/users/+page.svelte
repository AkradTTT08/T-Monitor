<script lang="ts">
  import { onMount } from "svelte";

  let users: any[] = [];
  let isLoading = true;
  let currentUser: any = null;
  let errorMsg = "";
  let successMsg = "";

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
    errorMsg = "";

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch("http://localhost:5273/api/v1/users", {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        users = await res.json();
      } else {
        errorMsg = "Failed to load users. Ensure you have admin permissions.";
      }
    } catch (err) {
      errorMsg = "Network error fetching users.";
    } finally {
      isLoading = false;
    }
  }

  async function toggleRole(userId: number, currentRole: string) {
    const newRole = currentRole === "admin" ? "user" : "admin";

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `http://localhost:5273/api/v1/users/${userId}/role`,
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
        `http://localhost:5273/api/v1/users/${userId}/approve`,
        {
          method: "PUT",
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        successMsg = "User approved successfully.";
        setTimeout(() => (successMsg = ""), 3000);
        await fetchUsers();
      } else {
        errorMsg = "Failed to approve user.";
        setTimeout(() => (errorMsg = ""), 3000);
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function disapproveUser(userId: number) {
    if (!confirm("Are you sure you want to disapprove and remove this user?"))
      return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `http://localhost:5273/api/v1/users/${userId}/disapprove`,
        {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        successMsg = "User disapproved and removed.";
        setTimeout(() => (successMsg = ""), 3000);
        await fetchUsers();
      } else {
        errorMsg = "Failed to disapprove user.";
        setTimeout(() => (errorMsg = ""), 3000);
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function resetPassword(userId: number) {
    if (
      !confirm(
        "Are you sure you want to reset this user password to the default?",
      )
    )
      return;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `http://localhost:5273/api/v1/users/${userId}/reset-password`,
        {
          method: "PUT",
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        successMsg = "Password reset to default (T@monitor123) successfully.";
        setTimeout(() => (successMsg = ""), 5000);
      } else {
        errorMsg = "Failed to reset password.";
        setTimeout(() => (errorMsg = ""), 3000);
      }
    } catch (err) {
      console.error(err);
    }
  }
</script>

<div class="fade-in max-w-7xl mx-auto w-full overflow-hidden">
  <div class="mb-8">
    <h1 class="text-2xl md:text-3xl font-bold text-slate-900 tracking-tight">
      Manage Users
    </h1>
    <p class="text-sm md:text-base text-slate-500 mt-2 text-wrap">
      Administrate platform access and assignment of administrative privileges.
    </p>
  </div>

  {#if errorMsg}
    <div
      class="bg-red-50 text-red-600 p-4 rounded-xl border border-red-100 mb-6 max-w-full overflow-hidden text-ellipsis"
    >
      {errorMsg}
    </div>
  {/if}

  {#if successMsg}
    <div
      class="bg-green-50 text-green-700 p-4 rounded-xl border border-green-100 mb-6"
    >
      {successMsg}
    </div>
  {/if}

  <div
    class="bg-white border border-slate-200 rounded-2xl shadow-sm overflow-x-auto w-full"
  >
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
    {:else}
      <table class="w-full text-left border-collapse min-w-[600px]">
        <thead>
          <tr
            class="bg-slate-50 border-b border-slate-200 text-sm font-semibold text-slate-600"
          >
            <th class="p-4 pl-6">Identifier</th>
            <th class="p-4">Email Address</th>
            <th class="p-4">Global Role</th>
            <th class="p-4">Status</th>
            <th class="p-4">Joined Date</th>
            <th class="p-4 pr-6 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          {#each users as u}
            <tr class="hover:bg-slate-50/50 transition-colors group">
              <td class="p-4 pl-6">
                <div class="flex items-center gap-3">
                  <div
                    class="w-8 h-8 rounded-full bg-blue-100 flex shrink-0 items-center justify-center text-blue-700 font-bold uppercase text-xs"
                  >
                    {u.email.charAt(0)}
                  </div>
                  <span
                    class="text-slate-500 font-mono text-sm max-w-[80px] md:max-w-[none] truncate"
                    >#{u.id}</span
                  >
                </div>
              </td>
              <td
                class="p-4 font-medium text-slate-800 max-w-[120px] md:max-w-xs truncate"
                title={u.email}>{u.email}</td
              >
              <td class="p-4">
                <span
                  class="px-2.5 py-1 rounded text-xs font-bold uppercase tracking-wider whitespace-nowrap
                  {u.role === 'admin'
                    ? 'bg-purple-100 text-purple-700'
                    : 'bg-slate-100 text-slate-600'}"
                >
                  {u.role}
                </span>
              </td>
              <td class="p-4">
                {#if u.is_approved}
                  <span
                    class="px-2 py-1 bg-green-100 text-green-700 text-xs font-bold uppercase rounded flex items-center gap-1 w-max"
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
                    Active
                  </span>
                {:else}
                  <span
                    class="px-2 py-1 bg-amber-100 text-amber-700 text-xs font-bold uppercase rounded flex items-center gap-1 w-max animate-pulse"
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
                    Pending
                  </span>
                {/if}
              </td>
              <td class="p-4 text-slate-500 text-sm whitespace-nowrap">
                {new Date(u.created_at).toLocaleDateString()}
              </td>
              <td class="p-4 pr-6">
                <div
                  class="flex items-center justify-end gap-3 text-sm font-medium"
                >
                  {#if !u.is_approved && (!currentUser || currentUser.id !== u.id)}
                    <button
                      on:click={() => approveUser(u.id)}
                      class="text-green-600 hover:text-green-800 transition-colors bg-green-50 hover:bg-green-100 px-3 py-1.5 rounded-lg border border-green-200 shadow-sm text-xs flex items-center gap-1"
                      title="Approve this user"
                    >
                      ยืนยัน (Approve)
                    </button>
                    <button
                      on:click={() => disapproveUser(u.id)}
                      class="text-red-500 hover:text-red-700 transition-colors bg-red-50 hover:bg-red-100 px-3 py-1.5 rounded-lg border border-red-200 shadow-sm text-xs flex items-center gap-1"
                      title="Disapprove (Reject) this user"
                    >
                      ไม่อนุมัติ (Reject)
                    </button>
                  {/if}

                  <button
                    on:click={() => resetPassword(u.id)}
                    class="text-slate-500 hover:text-slate-700 transition-colors flex items-center gap-1"
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
                    <span class="hidden md:inline">Reset</span>
                  </button>

                  <button
                    on:click={() => toggleRole(u.id, u.role)}
                    disabled={currentUser && currentUser.id === u.id}
                    class="transition-colors whitespace-nowrap ml-2
                      {currentUser && currentUser.id === u.id
                      ? 'text-slate-300 cursor-not-allowed'
                      : u.role === 'admin'
                        ? 'text-slate-500 hover:text-slate-700'
                        : 'text-blue-600 hover:text-blue-800'}"
                  >
                    {u.role === "admin"
                      ? "ปรับเป็น User (Demote)"
                      : "ตั้งเป็น Admin (Promote)"}
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
