<script lang="ts">
    import { onMount } from "svelte";
    import Swal from "sweetalert2";

    let user = {
        email: "",
        name: "",
        department: "",
        position: "",
        phone: "",
        profile_image_url: "",
    };

    let passwords = {
        current: "",
        new: "",
        confirm: "",
    };
    let isProfileLoading = false;
    let isPasswordLoading = false;
    let initialLoading = true;
    let fileInput: HTMLInputElement;
    let showDeptDropdown = false;

    const departments = ["Transform", "Timely", "Trailblazer"];

    function selectDepartment(dept: string) {
        user.department = dept;
        showDeptDropdown = false;
    }

    $: passwordRules = [
        { label: "At least 8 characters", valid: passwords.new.length >= 8 },
        {
            label: "One uppercase letter (A-Z)",
            valid: /[A-Z]/.test(passwords.new),
        },
        {
            label: "One lowercase letter (a-z)",
            valid: /[a-z]/.test(passwords.new),
        },
        { label: "One number (0-9)", valid: /[0-9]/.test(passwords.new) },
        {
            label: "One special character (!@#…)",
            valid: /[!@#$%^&*()_+\-=\[\]{};':"|,.<>\/?]/.test(passwords.new),
        },
    ];

    $: passwordAllValid =
        passwords.new.length > 0 && passwordRules.every((r) => r.valid);

    function handleImageUpload(e: Event) {
        const target = e.target as HTMLInputElement;
        if (target.files && target.files.length > 0) {
            const file = target.files[0];

            if (file.size > 2 * 1024 * 1024) {
                Swal.fire("Error", "Image size exceeds 2MB limit", "error");
                return;
            }

            const validTypes = ["image/jpeg", "image/png", "image/jpg"];
            if (!validTypes.includes(file.type)) {
                Swal.fire(
                    "Error",
                    "Only .jpg, .jpeg, and .png are allowed",
                    "error",
                );
                return;
            }

            const reader = new FileReader();
            reader.onload = (e) => {
                if (e.target?.result) {
                    user.profile_image_url = e.target.result.toString();
                    // Also update localStorage and notify the sidebar immediately
                    const storedUser = JSON.parse(
                        localStorage.getItem("monitor_user") || "{}",
                    );
                    storedUser.profile_image_url = user.profile_image_url;
                    localStorage.setItem(
                        "monitor_user",
                        JSON.stringify(storedUser),
                    );
                    window.dispatchEvent(
                        new CustomEvent("user-updated", { detail: storedUser }),
                    );
                }
            };
            reader.readAsDataURL(file);
        }
    }

    onMount(async () => {
        await fetchProfile();
    });

    async function fetchProfile() {
        initialLoading = true;
        try {
            const token = localStorage.getItem("monitor_token");
            const res = await fetch("http://localhost:5273/api/v1/profile", {
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            });

            if (res.ok) {
                user = await res.json();
            } else {
                console.error("Failed to load profile");
            }
        } catch (err) {
            console.error(err);
        } finally {
            initialLoading = false;
        }
    }

    async function handleUpdateProfile() {
        isProfileLoading = true;
        try {
            const token = localStorage.getItem("monitor_token");
            const res = await fetch("http://localhost:5273/api/v1/profile", {
                method: "PUT",
                headers: {
                    Authorization: `Bearer ${token}`,
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    name: user.name,
                    department: user.department,
                    position: user.position,
                    phone: user.phone,
                    profile_image_url: user.profile_image_url,
                }),
            });

            if (res.ok) {
                const updatedUser = await res.json();
                // Update local storage user data for the layout
                const storedUser = JSON.parse(
                    localStorage.getItem("monitor_user") || "{}",
                );
                storedUser.name = updatedUser.name;
                storedUser.profile_image_url = updatedUser.profile_image_url;
                localStorage.setItem(
                    "monitor_user",
                    JSON.stringify(storedUser),
                );
                window.dispatchEvent(
                    new CustomEvent("user-updated", { detail: storedUser }),
                );

                Swal.fire({
                    icon: "success",
                    title: "Profile Updated",
                    text: "Your profile information has been saved.",
                    timer: 2000,
                    showConfirmButton: false,
                });
            } else {
                const err = await res.json();
                Swal.fire(
                    "Error",
                    err.error || "Failed to update profile",
                    "error",
                );
            }
        } catch (err) {
            console.error(err);
            Swal.fire("Error", "A network error occurred.", "error");
        } finally {
            isProfileLoading = false;
        }
    }

    async function handleUpdatePassword() {
        if (passwords.new !== passwords.confirm) {
            Swal.fire("Error", "New passwords do not match.", "error");
            return;
        }

        const pwdErrors = passwordRules
            .filter((r) => !r.valid)
            .map((r) => r.label);
        if (pwdErrors.length > 0) {
            Swal.fire(
                "Password too weak",
                "Password must contain: " + pwdErrors.join(", "),
                "error",
            );
            return;
        }

        isPasswordLoading = true;
        try {
            const token = localStorage.getItem("monitor_token");
            const res = await fetch(
                "http://localhost:5273/api/v1/profile/password",
                {
                    method: "PUT",
                    headers: {
                        Authorization: `Bearer ${token}`,
                        "Content-Type": "application/json",
                    },
                    body: JSON.stringify({
                        current_password: passwords.current,
                        new_password: passwords.new,
                    }),
                },
            );

            if (res.ok) {
                Swal.fire({
                    icon: "success",
                    title: "Password Updated",
                    text: "Your password has been changed successfully.",
                    timer: 2000,
                    showConfirmButton: false,
                });
                passwords = { current: "", new: "", confirm: "" };
            } else {
                const err = await res.json();
                Swal.fire(
                    "Error",
                    err.error || "Failed to update password",
                    "error",
                );
            }
        } catch (err) {
            console.error(err);
            Swal.fire("Error", "A network error occurred.", "error");
        } finally {
            isPasswordLoading = false;
        }
    }
</script>

<div
    class="max-w-4xl mx-auto space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500"
>
    <div class="mb-8">
        <h1 class="text-3xl font-extrabold text-slate-900 tracking-tight">
            Profile Settings
        </h1>
        <p class="text-slate-500 mt-2">
            Manage your account details and password.
        </p>
    </div>

    {#if initialLoading}
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
        <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
            <!-- Profile Information Section -->
            <div class="md:col-span-1">
                <h3 class="text-lg font-bold text-slate-900 mb-2">
                    Personal Information
                </h3>
                <p class="text-sm text-slate-500">
                    Update your basic profile information and email address.
                </p>

                <div class="mt-6 flex flex-col items-center">
                    <input
                        type="file"
                        accept=".jpg,.jpeg,.png"
                        class="hidden"
                        bind:this={fileInput}
                        on:change={handleImageUpload}
                    />
                    <div
                        class="relative group cursor-pointer"
                        on:click={() => fileInput.click()}
                        on:keydown={(e) =>
                            e.key === "Enter" && fileInput.click()}
                        role="button"
                        tabindex="0"
                    >
                        <div
                            class="w-32 h-32 rounded-full bg-slate-900 flex items-center justify-center text-white text-3xl font-bold uppercase overflow-hidden shadow-xl border-4 border-white"
                        >
                            {#if user.profile_image_url}
                                <img
                                    src={user.profile_image_url}
                                    alt="Profile"
                                    class="w-full h-full object-cover"
                                />
                            {:else}
                                {user.email?.charAt(0) || "U"}
                            {/if}
                        </div>
                        <div
                            class="absolute inset-0 bg-black/50 rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                        >
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                width="24"
                                height="24"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="white"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                ><path
                                    d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"
                                /><circle cx="12" cy="13" r="4" /></svg
                            >
                        </div>
                    </div>
                    <span
                        class="text-xs text-slate-500 mt-3 font-medium text-center leading-relaxed"
                        >Max size 2MB <br /> Allowed: .jpg, .jpeg, .png</span
                    >
                </div>
            </div>

            <div
                class="md:col-span-2 bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden"
            >
                <form
                    on:submit|preventDefault={handleUpdateProfile}
                    class="p-6 md:p-8 space-y-6"
                >
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div class="space-y-2">
                            <label
                                for="name"
                                class="block text-sm font-semibold text-slate-700"
                                >Full Name</label
                            >
                            <input
                                id="name"
                                type="text"
                                bind:value={user.name}
                                placeholder="John Doe"
                                class="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-medium"
                            />
                        </div>
                        <div class="space-y-2">
                            <label
                                for="phone"
                                class="block text-sm font-semibold text-slate-700"
                                >Phone Number</label
                            >
                            <input
                                id="phone"
                                type="text"
                                bind:value={user.phone}
                                placeholder="+1 (555) 000-0000"
                                class="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-medium"
                            />
                        </div>
                    </div>

                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 pt-2">
                        <div class="space-y-2">
                            <label
                                for="department"
                                class="block text-sm font-semibold text-slate-700"
                                >Department</label
                            >
                            <!-- Custom Dropdown -->
                            <div class="relative">
                                <button
                                    type="button"
                                    on:click={() =>
                                        (showDeptDropdown = !showDeptDropdown)}
                                    class="w-full px-4 py-3 bg-slate-50 border {showDeptDropdown
                                        ? 'border-blue-500 ring-2 ring-blue-500/20'
                                        : 'border-slate-200'} rounded-xl text-left text-slate-900 font-medium flex items-center justify-between transition-all focus:outline-none"
                                >
                                    <span
                                        class={user.department
                                            ? ""
                                            : "text-slate-400"}
                                    >
                                        {user.department || "Select Department"}
                                    </span>
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
                                        class="text-slate-400 shrink-0 transition-transform {showDeptDropdown
                                            ? 'rotate-180'
                                            : ''}"
                                    >
                                        <polyline points="6 9 12 15 18 9" />
                                    </svg>
                                </button>

                                {#if showDeptDropdown}
                                    <div
                                        class="absolute z-20 top-full mt-1 w-full bg-white rounded-xl shadow-xl border border-slate-200 overflow-hidden animate-in slide-in-from-top-2 duration-150"
                                    >
                                        {#each departments as dept}
                                            <button
                                                type="button"
                                                on:click={() =>
                                                    selectDepartment(dept)}
                                                class="w-full text-left px-4 py-3 text-sm font-medium flex items-center gap-3 transition-colors {user.department ===
                                                dept
                                                    ? 'bg-blue-50 text-blue-700'
                                                    : 'text-slate-700 hover:bg-slate-50'}"
                                            >
                                                {#if user.department === dept}
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
                                                        ><polyline
                                                            points="20 6 9 17 4 12"
                                                        /></svg
                                                    >
                                                {:else}
                                                    <span class="w-3.5 shrink-0"
                                                    ></span>
                                                {/if}
                                                {dept}
                                            </button>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                        </div>
                        <div class="space-y-2">
                            <label
                                for="position"
                                class="block text-sm font-semibold text-slate-700"
                                >Position</label
                            >
                            <input
                                id="position"
                                type="text"
                                bind:value={user.position}
                                placeholder="Senior Developer"
                                class="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-medium"
                            />
                        </div>
                    </div>

                    <div class="space-y-2">
                        <label
                            for="email"
                            class="block text-sm font-semibold text-slate-700"
                            >Email Address</label
                        >
                        <input
                            id="email"
                            type="email"
                            value={user.email}
                            disabled
                            class="w-full px-4 py-3 bg-slate-100 border border-slate-200 rounded-xl text-slate-500 cursor-not-allowed font-medium"
                        />
                        <p class="text-xs text-slate-500 mt-1">
                            Email is managed by the system administrator.
                        </p>
                    </div>

                    <div class="pt-4 flex justify-end">
                        <button
                            type="submit"
                            disabled={isProfileLoading}
                            class="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 active:bg-blue-800 text-white font-semibold rounded-xl shadow-sm hover:shadow transition-all disabled:opacity-70 flex items-center gap-2"
                        >
                            {#if isProfileLoading}
                                <svg
                                    class="animate-spin h-5 w-5 text-white"
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
                                Save Changes
                            {/if}
                        </button>
                    </div>
                </form>
            </div>

            <!-- Password Section -->
            <div
                class="md:col-span-1 mt-8 md:mt-0 pt-8 border-t border-slate-200 md:border-0 md:pt-0"
            >
                <h3 class="text-lg font-bold text-slate-900 mb-2">Security</h3>
                <p class="text-sm text-slate-500">
                    Ensure your account is using a long, random password to stay
                    secure.
                </p>
            </div>

            <div
                class="md:col-span-2 bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden"
            >
                <form
                    on:submit|preventDefault={handleUpdatePassword}
                    class="p-6 md:p-8 space-y-6"
                >
                    <div class="space-y-2">
                        <label
                            for="current_password"
                            class="block text-sm font-semibold text-slate-700"
                            >Current Password</label
                        >
                        <input
                            id="current_password"
                            type="password"
                            bind:value={passwords.current}
                            required
                            class="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-900/20 focus:border-slate-900 transition-all font-medium"
                        />
                    </div>

                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 pt-2">
                        <div class="space-y-2">
                            <label
                                for="new_password"
                                class="block text-sm font-semibold text-slate-700"
                                >New Password</label
                            >
                            <input
                                id="new_password"
                                type="password"
                                bind:value={passwords.new}
                                required
                                class="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-900/20 focus:border-slate-900 transition-all font-medium"
                            />
                            {#if passwords.new.length > 0}
                                <ul class="mt-2 space-y-1">
                                    {#each passwordRules as rule}
                                        <li
                                            class="flex items-center gap-1.5 text-xs font-medium {rule.valid
                                                ? 'text-emerald-600'
                                                : 'text-slate-400'}"
                                        >
                                            {#if rule.valid}
                                                <svg
                                                    xmlns="http://www.w3.org/2000/svg"
                                                    class="w-3.5 h-3.5 shrink-0"
                                                    viewBox="0 0 24 24"
                                                    fill="none"
                                                    stroke="currentColor"
                                                    stroke-width="2.5"
                                                    stroke-linecap="round"
                                                    stroke-linejoin="round"
                                                    ><polyline
                                                        points="20 6 9 17 4 12"
                                                    /></svg
                                                >
                                            {:else}
                                                <span
                                                    class="w-3.5 h-3.5 shrink-0 flex items-center justify-center"
                                                    >•</span
                                                >
                                            {/if}
                                            {rule.label}
                                        </li>
                                    {/each}
                                </ul>
                            {/if}
                        </div>
                        <div class="space-y-2">
                            <label
                                for="confirm_password"
                                class="block text-sm font-semibold text-slate-700"
                                >Confirm Password</label
                            >
                            <input
                                id="confirm_password"
                                type="password"
                                bind:value={passwords.confirm}
                                required
                                class="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl text-slate-900 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-900/20 focus:border-slate-900 transition-all font-medium"
                            />
                        </div>
                    </div>

                    <div class="pt-4 flex justify-end">
                        <button
                            type="submit"
                            disabled={isPasswordLoading ||
                                !passwords.current ||
                                !passwords.new ||
                                !passwords.confirm}
                            class="px-6 py-2.5 bg-slate-900 hover:bg-black active:bg-slate-800 text-white font-semibold rounded-xl shadow-sm hover:shadow transition-all disabled:opacity-70 flex items-center gap-2"
                        >
                            {#if isPasswordLoading}
                                <svg
                                    class="animate-spin h-5 w-5 text-white"
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
                                Updating...
                            {:else}
                                Update Password
                            {/if}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    {/if}
</div>
