<script lang="ts">
    import { onMount } from "svelte";
    import Swal from "sweetalert2";
    import { systemAlert, systemToast } from "$lib/swal-design";
    import { API_BASE_URL } from "$lib/config";

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
                systemAlert.fire("Error", "Image size exceeds 2MB limit", "error");
                return;
            }

            const validTypes = ["image/jpeg", "image/png", "image/jpg"];
            if (!validTypes.includes(file.type)) {
                systemAlert.fire(
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
            const res = await fetch(`${API_BASE_URL}/api/v1/profile`, {
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
            const res = await fetch(`${API_BASE_URL}/api/v1/profile`, {
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

                systemToast.fire({
                    icon: "success",
                    title: "Profile Updated",
                    text: "Your profile information has been saved.",
                });
            } else {
                const err = await res.json();
                systemAlert.fire(
                    "Error",
                    err.error || "Failed to update profile",
                    "error",
                );
            }
        } catch (err) {
            console.error(err);
            systemAlert.fire("Error", "A network error occurred.", "error");
        } finally {
            isProfileLoading = false;
        }
    }

    async function handleUpdatePassword() {
        if (passwords.new !== passwords.confirm) {
            systemAlert.fire("Error", "New passwords do not match.", "error");
            return;
        }

        const pwdErrors = passwordRules
            .filter((r) => !r.valid)
            .map((r) => r.label);
        if (pwdErrors.length > 0) {
            systemAlert.fire(
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
                `${API_BASE_URL}/api/v1/profile/password`,
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
                systemToast.fire({
                    icon: "success",
                    title: "Password Updated",
                    text: "Your password has been changed successfully.",
                });
                passwords = { current: "", new: "", confirm: "" };
            } else {
                const err = await res.json();
                systemAlert.fire(
                    "Error",
                    err.error || "Failed to update password",
                    "error",
                );
            }
        } catch (err) {
            console.error(err);
            systemAlert.fire("Error", "A network error occurred.", "error");
        } finally {
            isPasswordLoading = false;
        }
    }
</script>

<div
    class="max-w-4xl mx-auto space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500"
>
    <div class="mb-8">
        <h1
            class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase"
        >
            PROFILE_SETTINGS
        </h1>
        <p class="text-cyan-500/80 mt-2 font-mono text-sm tracking-wide">
            MANAGE YOUR ACCOUNT DETAILS AND PASSWORD.
        </p>
    </div>

    {#if initialLoading}
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
        <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
            <!-- Profile Information Section -->
            <div class="md:col-span-1">
                <h3
                    class="text-lg font-bold text-cyan-50 font-mono tracking-widest uppercase mb-2"
                >
                    PERSONAL_INFO
                </h3>
                <p
                    class="text-xs font-mono tracking-wider text-slate-400 uppercase"
                >
                    UPDATE YOUR BASIC PROFILE INFORMATION AND EMAIL ADDRESS.
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
                            class="w-32 h-32 rounded-full bg-slate-900 flex items-center justify-center text-cyan-400 text-3xl font-bold uppercase overflow-hidden shadow-[0_0_20px_rgba(6,182,212,0.3)] border-2 border-cyan-500/50"
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
                        class="text-[10px] text-slate-500 mt-4 font-bold font-mono tracking-widest uppercase text-center leading-relaxed"
                        >MAX_SIZE 2MB <br /> ALLOWED: .JPG, .JPEG, .PNG</span
                    >
                </div>
            </div>

            <div
                class="md:col-span-2 bg-slate-900/60 backdrop-blur-md rounded-3xl border border-slate-700/50 shadow-[0_8px_30px_rgb(0,0,0,0.5)] overflow-hidden"
            >
                <form
                    on:submit|preventDefault={handleUpdateProfile}
                    class="p-6 md:p-8 space-y-6"
                >
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div class="space-y-2">
                            <label
                                for="name"
                                class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                                >FULL_NAME</label
                            >
                            <input
                                id="name"
                                type="text"
                                bind:value={user.name}
                                placeholder="John Doe"
                                class="w-full px-4 py-3 bg-slate-950/50 border border-slate-700 rounded-xl text-cyan-50 font-mono text-sm tracking-wide placeholder:text-slate-600 focus:outline-none focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/50 transition-all"
                            />
                        </div>
                        <div class="space-y-2">
                            <label
                                for="phone"
                                class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                                >PHONE_NUMBER</label
                            >
                            <input
                                id="phone"
                                type="text"
                                bind:value={user.phone}
                                placeholder="+1 (555) 000-0000"
                                class="w-full px-4 py-3 bg-slate-950/50 border border-slate-700 rounded-xl text-cyan-50 font-mono text-sm tracking-wide placeholder:text-slate-600 focus:outline-none focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/50 transition-all"
                            />
                        </div>
                    </div>

                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 pt-2">
                        <div class="space-y-2">
                            <label
                                for="department"
                                class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                                >DEPARTMENT</label
                            >
                            <!-- Custom Dropdown -->
                            <div class="relative">
                                <button
                                    type="button"
                                    on:click={() =>
                                        (showDeptDropdown = !showDeptDropdown)}
                                    class="w-full px-4 py-3 bg-slate-950/50 border {showDeptDropdown
                                        ? 'border-cyan-500/50 ring-1 ring-cyan-500/50'
                                        : 'border-slate-700'} rounded-xl text-left text-cyan-50 font-mono text-sm tracking-wide flex items-center justify-between transition-all focus:outline-none"
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
                                        class="absolute z-20 top-full mt-2 w-full bg-slate-900 border border-slate-700/80 rounded-xl shadow-[0_8px_30px_rgb(0,0,0,0.8)] overflow-hidden animate-in slide-in-from-top-2 duration-150"
                                    >
                                        {#each departments as dept}
                                            <button
                                                type="button"
                                                on:click={() =>
                                                    selectDepartment(dept)}
                                                class="w-full text-left px-4 py-3 text-xs font-mono font-bold tracking-widest uppercase flex items-center gap-3 transition-colors {user.department ===
                                                dept
                                                    ? 'bg-cyan-950/50 text-cyan-400'
                                                    : 'text-slate-400 hover:bg-slate-800 hover:text-cyan-50'}"
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
                                class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                                >POSITION</label
                            >
                            <input
                                id="position"
                                type="text"
                                bind:value={user.position}
                                placeholder="Senior Developer"
                                class="w-full px-4 py-3 bg-slate-950/50 border border-slate-700 rounded-xl text-cyan-50 font-mono text-sm tracking-wide placeholder:text-slate-600 focus:outline-none focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/50 transition-all"
                            />
                        </div>
                    </div>

                    <div class="space-y-2">
                        <label
                            for="email"
                            class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                            >EMAIL_ADDRESS</label
                        >
                        <input
                            id="email"
                            type="email"
                            value={user.email}
                            disabled
                            class="w-full px-4 py-3 bg-slate-800/80 border border-slate-700 rounded-xl text-slate-500 cursor-not-allowed font-mono text-sm tracking-wide"
                        />
                        <p
                            class="text-[10px] text-slate-500 mt-2 font-mono uppercase tracking-widest font-bold"
                        >
                            EMAIL IS MANAGED BY THE SYSTEM ADMINISTRATOR.
                        </p>
                    </div>

                    <div class="pt-4 flex justify-end">
                        <button
                            type="submit"
                            disabled={isProfileLoading}
                            class="px-6 py-3 font-mono text-xs font-bold tracking-widest uppercase rounded-xl transition-all flex items-center gap-2 bg-cyan-950/80 text-cyan-400 border border-cyan-500/30 hover:bg-cyan-900/80 hover:border-cyan-400/50 hover:shadow-[0_0_15px_rgba(6,182,212,0.3)] disabled:opacity-50"
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
                                SAVE_CHANGES
                            {/if}
                        </button>
                    </div>
                </form>
            </div>

            <!-- Password Section -->
            <div
                class="md:col-span-1 mt-8 md:mt-0 pt-8 border-t border-slate-800 md:border-0 md:pt-0"
            >
                <h3
                    class="text-lg font-bold text-cyan-50 font-mono tracking-widest uppercase mb-2"
                >
                    SECURITY
                </h3>
                <p
                    class="text-xs font-mono tracking-wider text-slate-400 uppercase"
                >
                    ENSURE YOUR ACCOUNT IS USING A LONG, RANDOM PASSWORD TO STAY
                    SECURE.
                </p>
            </div>

            <div
                class="md:col-span-2 bg-slate-900/60 backdrop-blur-md rounded-3xl border border-slate-700/50 shadow-[0_8px_30px_rgb(0,0,0,0.5)] overflow-hidden"
            >
                <form
                    on:submit|preventDefault={handleUpdatePassword}
                    class="p-6 md:p-8 space-y-6"
                >
                    <div class="space-y-2">
                        <label
                            for="current_password"
                            class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                            >CURRENT_PASSWORD</label
                        >
                        <input
                            id="current_password"
                            type="password"
                            bind:value={passwords.current}
                            required
                            class="w-full px-4 py-3 bg-slate-950/50 border border-slate-700 rounded-xl text-cyan-50 font-mono text-sm tracking-wide placeholder:text-slate-600 focus:outline-none focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/50 transition-all"
                        />
                    </div>

                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 pt-2">
                        <div class="space-y-2">
                            <label
                                for="new_password"
                                class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                                >NEW_PASSWORD</label
                            >
                            <input
                                id="new_password"
                                type="password"
                                bind:value={passwords.new}
                                required
                                class="w-full px-4 py-3 bg-slate-950/50 border border-slate-700 rounded-xl text-cyan-50 font-mono text-sm tracking-wide placeholder:text-slate-600 focus:outline-none focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/50 transition-all"
                            />
                            {#if passwords.new.length > 0}
                                <ul class="mt-2 space-y-1">
                                    {#each passwordRules as rule}
                                        <li
                                            class="flex items-center gap-2 text-[10px] font-bold font-mono tracking-widest uppercase mt-1 mb-1 {rule.valid
                                                ? 'text-emerald-400 drop-shadow-[0_0_5px_rgba(52,211,153,0.5)]'
                                                : 'text-slate-500'}"
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
                                class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                                >CONFIRM_PASSWORD</label
                            >
                            <input
                                id="confirm_password"
                                type="password"
                                bind:value={passwords.confirm}
                                required
                                class="w-full px-4 py-3 bg-slate-950/50 border border-slate-700 rounded-xl text-cyan-50 font-mono text-sm tracking-wide placeholder:text-slate-600 focus:outline-none focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/50 transition-all"
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
                            class="px-6 py-3 font-mono text-xs font-bold tracking-widest uppercase rounded-xl transition-all flex items-center gap-2 bg-purple-950/50 text-purple-400 border border-purple-500/30 hover:bg-purple-900/60 hover:border-purple-400/50 hover:shadow-[0_0_15px_rgba(168,85,247,0.3)] disabled:opacity-50"
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
                                UPDATE_PASSWORD
                            {/if}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    {/if}
</div>
