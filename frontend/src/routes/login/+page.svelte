<script lang="ts">
  import { onMount } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import logoUrl from '../../image/SVG/Logo-T-monitor.svg';

  import { API_BASE_URL } from '$lib/config';

  let email = "";
  let password = "";
  let name = "";
  let phone = "";
  let department = "";
  let position = "";
  let error = "";
  let success = "";
  let isLoading = false;
  let isLogin = true;
  let rememberMe = false;
  let showDeptDropdown = false;

  const departments = ["Transform", "Timely", "Trailblazer"];

  function selectDepartment(dept: string) {
    department = dept;
    showDeptDropdown = false;
  }

  onMount(() => {
    // Check if already logged in
    const token = localStorage.getItem("monitor_token");
    if (token) {
      window.location.href = "/dashboard";
      return;
    }

    const savedEmail = localStorage.getItem("monitor_remembered_email");
    if (savedEmail) {
      email = savedEmail;
      rememberMe = true;
    }
  });

  // Password Validation function
  function validatePassword(pwd: string): string {
    if (isLogin) return ""; // Only validate structure heavily on Sign Up

    if (pwd.length < 8) return "Password must be at least 8 characters long.";
    if (!/[A-Z]/.test(pwd))
      return "Password must contain at least one uppercase letter.";
    if (!/[a-z]/.test(pwd))
      return "Password must contain at least one lowercase letter.";
    if (!/[0-9]/.test(pwd)) return "Password must contain at least one number.";
    if (!/[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(pwd))
      return "Password must contain at least one special character.";

    return "";
  }

  async function handleSubmit() {
    isLoading = true;
    error = "";
    success = "";

    const pwdError = validatePassword(password);
    if (pwdError) {
      error = pwdError;
      isLoading = false;
      return;
    }

    const endpoint = isLogin ? "/auth/login" : "/auth/register";

    if (!isLogin && !department) {
      error = "กรุณาเลือกแผนก (Department)";
      isLoading = false;
      return;
    }

    try {
      const payload = isLogin 
        ? { email, password } 
        : { email, password, name, phone, department, position };

      const res = await fetch(`${API_BASE_URL}/api/v1${endpoint}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      const data = await res.json();

      if (!res.ok) {
        if (data.error === "Waiting for admin approval") {
          error = "กำลังรอ Admin ระบบยืนยันการสมัคร";
        } else {
          error = data.error || "Authentication failed";
        }
      } else {
        if (isLogin) {
          localStorage.setItem("monitor_token", data.token);
          localStorage.setItem("monitor_user", JSON.stringify(data.user));
          
          if (rememberMe) {
            localStorage.setItem("monitor_remembered_email", email);
          } else {
            localStorage.removeItem("monitor_remembered_email");
          }

          // Redirect to dashboard
          window.location.href = "/dashboard";
        } else {
          success = "สมัครสมาชิกสำเร็จ! กรุณารอ Admin ระบบยืนยันการสมัคร";
          isLogin = true;
          password = "";
        }
      }
    } catch (err) {
      error = "Network error. Please try again.";
    } finally {
      isLoading = false;
    }
  }
</script>

<div 
  class="w-full max-w-md p-10 rounded-3xl bg-slate-800/60 backdrop-blur-xl shadow-[0_8px_30px_rgb(0,0,0,0.5)] border border-slate-700/50 relative z-10 transition-all duration-500 hover:shadow-[0_0px_40px_rgba(6,182,212,0.15)] hover:border-cyan-500/30"
  in:fly={{ y: 20, duration: 800, delay: 200 }}
>
  <div class="text-center mb-10">
    <!-- Custom TTT Logo -->
    <div class="flex justify-center mb-6">
      <div class="relative w-24 h-24 flex items-center justify-center transform hover:scale-105 transition-transform duration-500 group">
        <!-- Glow effect behind logo -->
        <div class="absolute inset-0 bg-blue-500 rounded-full mix-blend-screen filter blur-xl opacity-20 group-hover:opacity-40 transition-opacity duration-500"></div>
        
        <!-- T-Monitor User SVG Logo -->
        <img src={logoUrl} alt="T-Monitor Logo" class="w-20 h-20 object-contain drop-shadow-[0_0_15px_rgba(6,182,212,0.5)] z-10" />

        <!-- Status Pulse indicator on logo -->
        <div class="absolute right-0 bottom-0 w-4 h-4 bg-emerald-400 border-2 border-slate-800 rounded-full animate-pulse shadow-[0_0_15px_rgba(52,211,153,1)] z-20"></div>
      </div>
    </div>
    
    <h1 class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight mb-2 font-mono">
      T-Monitor
    </h1>
    <h2 class="text-lg font-medium text-slate-300 mb-1">
      {#key isLogin}
        <span in:fade={{duration: 300}}>
          {isLogin ? "System Access Protocol" : "Initialize Account"}
        </span>
      {/key}
    </h2>
    <p class="text-cyan-500/80 text-sm font-mono tracking-wide">API DIAGNOSTICS & RECOVERY UNIT</p>
  </div>

  {#if error}
    <div in:fly={{y: -10, duration: 300}} out:fade class="bg-red-50 text-red-600 text-sm p-4 rounded-xl border border-red-100 mb-6 flex items-start space-x-3 shadow-sm">
      <svg class="w-5 h-5 shrink-0 mt-0.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>
      <span class="font-medium">{error}</span>
    </div>
  {/if}

  {#if success}
    <div in:fly={{y: -10, duration: 300}} out:fade class="bg-green-50 text-green-700 text-sm p-4 rounded-xl border border-green-100 mb-6 flex items-start space-x-3 shadow-sm">
      <svg class="w-5 h-5 shrink-0 mt-0.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
      <span class="font-medium">{success}</span>
    </div>
  {/if}

  <form on:submit|preventDefault={handleSubmit} class="space-y-5">
    <div>
      <label for="email" class="block text-sm font-semibold text-slate-300 mb-2 font-mono">USER_ID</label>
      <div class="relative group">
        <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none transition-colors group-focus-within:text-cyan-400 text-slate-500">
          <svg class="h-5 w-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline></svg>
        </div>
        <input
          id="email"
          type="email"
          bind:value={email}
          required
          class="w-full pl-11 pr-4 py-3.5 rounded-xl border border-slate-700/80 bg-slate-900/50 hover:bg-slate-900/80 focus:bg-slate-900 focus:border-cyan-500 focus:ring-4 focus:ring-cyan-500/10 transition-all duration-300 text-sm outline-none text-cyan-50 font-medium placeholder:text-slate-600 placeholder:font-normal font-mono"
          placeholder="admin@monitor.com"
        />
      </div>
    </div>

    <div>
      <label for="password" class="block text-sm font-semibold text-slate-300 mb-2 font-mono">ACCESS_KEY</label>
      <div class="relative group">
        <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none transition-colors group-focus-within:text-cyan-400 text-slate-500">
          <svg class="h-5 w-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
        </div>
        <input
          id="password"
          type="password"
          bind:value={password}
          required
          minlength={isLogin ? null : 8}
          class="w-full pl-11 pr-4 py-3.5 rounded-xl border border-slate-700/80 bg-slate-900/50 hover:bg-slate-900/80 focus:bg-slate-900 focus:border-cyan-500 focus:ring-4 focus:ring-cyan-500/10 transition-all duration-300 text-sm outline-none text-cyan-50 font-medium placeholder:text-slate-600 placeholder:font-normal font-mono"
          placeholder="••••••••"
        />
      </div>
      {#if !isLogin}
        <p in:fly={{y: -10, duration: 200}} class="text-xs text-slate-500 mt-2.5 ml-1 leading-relaxed">
          Must be at least 8 chars with 1 uppercase, 1 lowercase, 1 number, and 1 special char.
        </p>
      {/if}
    </div>

    {#if !isLogin}
      <div in:fly={{y: 10, duration: 300, delay: 100}} class="grid grid-cols-1 md:grid-cols-2 gap-5">
        <div>
          <label for="name" class="block text-sm font-semibold text-slate-300 mb-2 font-mono">FULL_NAME</label>
          <input
            id="name"
            type="text"
            bind:value={name}
            required
            class="w-full px-4 py-3.5 rounded-xl border border-slate-700/80 bg-slate-900/50 hover:bg-slate-900/80 focus:bg-slate-900 focus:border-cyan-500 focus:ring-4 focus:ring-cyan-500/10 transition-all duration-300 text-sm outline-none text-cyan-50 font-medium placeholder:text-slate-600 placeholder:font-normal font-mono"
            placeholder="John Doe"
          />
        </div>
        <div>
          <label for="phone" class="block text-sm font-semibold text-slate-300 mb-2 font-mono">PHONE_NUMBER</label>
          <input
            id="phone"
            type="text"
            bind:value={phone}
            required
            class="w-full px-4 py-3.5 rounded-xl border border-slate-700/80 bg-slate-900/50 hover:bg-slate-900/80 focus:bg-slate-900 focus:border-cyan-500 focus:ring-4 focus:ring-cyan-500/10 transition-all duration-300 text-sm outline-none text-cyan-50 font-medium placeholder:text-slate-600 placeholder:font-normal font-mono"
            placeholder="+66 8X XXX XXXX"
          />
        </div>
      </div>

      <div in:fly={{y: 10, duration: 300, delay: 150}} class="grid grid-cols-1 md:grid-cols-2 gap-5">
        <div class="relative">
          <label for="department" class="block text-sm font-semibold text-slate-300 mb-2 font-mono">DEPARTMENT</label>
          <button
            type="button"
            on:click={() => (showDeptDropdown = !showDeptDropdown)}
            class="w-full px-4 py-3.5 border border-slate-700/80 bg-slate-900/50 hover:bg-slate-900/80 focus:border-cyan-500 focus:ring-4 focus:ring-cyan-500/10 rounded-xl text-left text-sm font-medium flex items-center justify-between transition-all focus:outline-none font-mono {department ? 'text-cyan-50' : 'text-slate-600'}"
          >
            <span>{department || "Select Unit"}</span>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-slate-500 shrink-0 transition-transform {showDeptDropdown ? 'rotate-180 text-cyan-400' : ''}">
              <polyline points="6 9 12 15 18 9" />
            </svg>
          </button>

          {#if showDeptDropdown}
            <div class="absolute z-20 top-full mt-2 w-full bg-slate-800 rounded-xl shadow-xl border border-slate-700 overflow-hidden animate-in slide-in-from-top-2 duration-150">
              {#each departments as dept}
                <button
                  type="button"
                  on:click={() => selectDepartment(dept)}
                  class="w-full text-left px-4 py-3 text-sm font-medium font-mono flex items-center gap-3 transition-colors {department === dept ? 'bg-slate-700/50 text-cyan-400' : 'text-slate-300 hover:bg-slate-700'}"
                >
                  {#if department === dept}
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                  {:else}
                    <span class="w-3.5 shrink-0"></span>
                  {/if}
                  {dept}
                </button>
              {/each}
            </div>
          {/if}
        </div>
        <div>
          <label for="position" class="block text-sm font-semibold text-slate-300 mb-2 font-mono">POSITION</label>
          <input
            id="position"
            type="text"
            bind:value={position}
            required
            class="w-full px-4 py-3.5 rounded-xl border border-slate-700/80 bg-slate-900/50 hover:bg-slate-900/80 focus:bg-slate-900 focus:border-cyan-500 focus:ring-4 focus:ring-cyan-500/10 transition-all duration-300 text-sm outline-none text-cyan-50 font-medium placeholder:text-slate-600 placeholder:font-normal font-mono"
            placeholder="System Engineer"
          />
        </div>
      </div>
    {/if}

    {#if isLogin}
      <div class="flex items-center justify-between mt-1 pt-1" in:fade={{duration: 200}}>
        <label class="flex items-center space-x-3 cursor-pointer group">
          <div class="relative flex items-center justify-center border-none">
            <input type="checkbox" bind:checked={rememberMe} class="peer sr-only" />
            <div class="w-5 h-5 border-2 border-slate-600 rounded peer-checked:bg-cyan-600 peer-checked:border-cyan-600 peer-focus:ring-4 peer-focus:ring-cyan-500/20 transition-all bg-slate-800 group-hover:border-cyan-500"></div>
            <svg class="w-3.5 h-3.5 text-slate-900 absolute inset-0 m-auto pointer-events-none opacity-0 peer-checked:opacity-100 transition-opacity duration-200" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
          </div>
          <span class="text-sm font-medium text-slate-400 group-hover:text-cyan-400 transition-colors font-mono">Store Credentials</span>
        </label>
      </div>
    {/if}

    <button
      type="submit"
      disabled={isLoading}
      class="w-full bg-slate-900 border border-cyan-500/50 text-cyan-400 hover:bg-cyan-950/50 hover:border-cyan-400 hover:text-cyan-300 hover:shadow-[0_0_20px_rgba(6,182,212,0.4)] font-bold py-3.5 rounded-xl transition-all transform hover:-translate-y-0.5 active:translate-y-0 disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none mt-6 overflow-hidden relative group font-mono tracking-widest"
    >
      <div class="absolute inset-0 w-full h-full bg-cyan-400/10 -translate-x-full group-hover:animate-[shimmer_1.5s_infinite] skew-x-12"></div>
      <div class="relative flex items-center justify-center">
        {#if isLoading}
          <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-cyan-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
          AUTHENTICATING...
        {:else}
          {#key isLogin}
            <span in:fly={{ y: -10, duration: 300 }}>
              {isLogin ? "INITIALIZE CONNECTION" : "REGISTER UNIT"}
            </span>
          {/key}
        {/if}
      </div>
    </button>
  </form>

  <div class="mt-8 pt-6 border-t border-slate-700/50 text-center">
    <button
      type="button"
      on:click={() => {
        isLogin = !isLogin;
        error = "";
        success = "";
      }}
      class="text-sm font-semibold text-slate-400 hover:text-cyan-400 transition-colors group flex items-center justify-center w-full font-mono"
    >
      {#key isLogin}
        <span in:fade={{duration: 200}} class="inline-flex items-center">
          {isLogin ? "No access key?" : "Already registered?"} 
          <span class="text-cyan-500 group-hover:underline ml-1.5 flex items-center">
            {isLogin ? "Request Access" : "Authenticate"}
            <svg class="w-4 h-4 ml-1 transform group-hover:translate-x-1 transition-transform" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"></line><polyline points="12 5 19 12 12 19"></polyline></svg>
          </span>
        </span>
      {/key}
    </button>
  </div>
</div>

<style>
  @keyframes shimmer {
    100% {
      transform: translateX(100%);
    }
  }
</style>
