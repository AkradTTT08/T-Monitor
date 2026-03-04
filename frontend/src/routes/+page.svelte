<script lang="ts">
  let email = "";
  let password = "";
  let error = "";
  let success = "";
  let isLoading = false;

  let isLogin = true;

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

    try {
      const res = await fetch(`http://localhost:5273/api/v1${endpoint}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
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

<div class="glass-panel w-full max-w-md p-8 rounded-2xl relative z-10 fade-in">
  <div class="text-center mb-8">
    <div
      class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-blue-100/50 mb-4 shadow-inner"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="w-8 h-8 text-blue-600"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        ><path d="M12 2v20" /><path
          d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"
        /></svg
      >
    </div>
    <h1
      class="text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-700 to-cyan-500"
    >
      {isLogin ? "Welcome Back" : "Create Account"}
    </h1>
    <p class="text-slate-500 text-sm mt-2">API HealthCheck Monitor</p>
  </div>

  {#if error}
    <div
      class="bg-red-50 text-red-600 text-sm p-3 rounded-lg border border-red-100 mb-4 animate-pulse"
    >
      {error}
    </div>
  {/if}

  {#if success}
    <div
      class="bg-green-50 text-green-700 text-sm p-3 rounded-lg border border-green-100 mb-4"
    >
      {success}
    </div>
  {/if}

  <form on:submit|preventDefault={handleSubmit} class="space-y-5">
    <div>
      <label for="email" class="block text-sm font-medium text-slate-700 mb-1"
        >Email Address</label
      >
      <input
        id="email"
        type="email"
        bind:value={email}
        required
        class="w-full px-4 py-3 rounded-xl border border-slate-200 bg-white/50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all text-sm"
        placeholder="you@company.com"
      />
    </div>

    <div>
      <label
        for="password"
        class="block text-sm font-medium text-slate-700 mb-1">Password</label
      >
      <input
        id="password"
        type="password"
        bind:value={password}
        required
        minlength={isLogin ? null : 8}
        class="w-full px-4 py-3 rounded-xl border border-slate-200 bg-white/50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/50 transition-all text-sm"
        placeholder="••••••••"
      />
      {#if !isLogin}
        <p class="text-xs text-slate-400 mt-2 ml-1">
          Must be at least 8 chars with 1 uppercase, 1 lowercase, 1 number, and
          1 special char.
        </p>
      {/if}
    </div>

    <button
      type="submit"
      disabled={isLoading}
      class="w-full bg-gradient-to-r from-blue-600 to-cyan-500 text-white font-medium py-3 rounded-xl hover:shadow-lg hover:shadow-blue-500/30 transition-all transform hover:-translate-y-0.5 disabled:opacity-50 disabled:cursor-not-allowed"
    >
      {#if isLoading}
        <svg
          class="animate-spin -ml-1 mr-2 h-4 w-4 text-white inline-block"
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
        Processing...
      {:else}
        {isLogin ? "Sign In" : "Sign Up"}
      {/if}
    </button>
  </form>

  <div class="mt-6 text-center text-sm text-slate-500">
    <button
      type="button"
      on:click={() => {
        isLogin = !isLogin;
        error = "";
        success = "";
      }}
      class="text-blue-600 hover:text-blue-800 font-medium hover:underline transition-colors"
    >
      {isLogin
        ? "Don't have an account? Sign up"
        : "Already have an account? Sign in"}
    </button>
  </div>
</div>
