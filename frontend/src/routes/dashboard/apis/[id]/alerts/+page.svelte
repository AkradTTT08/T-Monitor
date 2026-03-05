<script lang="ts">
  import { page } from "$app/stores";
  import { onMount } from "svelte";

  let apiId = $page.params.id;

  let isSaving = false;
  let saveSuccess = false;

  let config = {
    api_id: parseInt(apiId || "0"),
    enable_telegram: false,
    telegram_chat_id: "",
    enable_line: false,
    line_user_id: "",
    enable_email: false,
    email_address: "",
    enable_ticketing: false,
  };

  onMount(async () => {
    await fetchApiAndConfig();
  });

  async function fetchApiAndConfig() {
    try {
      const token = localStorage.getItem("monitor_token");
      // Fetch the API to get its current notification config
      const res = await fetch(`http://localhost:5273/api/v1/apis?project_id=`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        const apis = await res.json();
        const api = apis.find((a: any) => a.id.toString() === apiId);
        if (api && api.notification_config) {
          config = {
            ...config,
            enable_telegram: api.notification_config.enable_telegram,
            telegram_chat_id: api.notification_config.telegram_chat_id,
            enable_line: api.notification_config.enable_line,
            line_user_id: api.notification_config.line_user_id,
            enable_email: api.notification_config.enable_email,
            email_address: api.notification_config.email_address,
            enable_ticketing: api.notification_config.enable_ticketing,
          };
        }
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function handleSave() {
    isSaving = true;
    saveSuccess = false;

    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`http://localhost:5273/api/v1/notifications`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(config),
      });

      if (res.ok) {
        saveSuccess = true;
        setTimeout(() => (saveSuccess = false), 3000);
      }
    } catch (err) {
      console.error(err);
    } finally {
      isSaving = false;
    }
  }
</script>

<div class="fade-in max-w-4xl mx-auto">
  <div class="flex items-center gap-4 mb-8">
    <a
      href="/dashboard/apis"
      class="p-2 bg-slate-950/80 border border-slate-700/80 rounded-xl text-cyan-500/50 hover:text-cyan-400 hover:border-cyan-500/50 hover:shadow-[0_0_15px_rgba(6,182,212,0.3)] transition-all"
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
        ><line x1="19" y1="12" x2="5" y2="12"></line><polyline
          points="12 19 5 12 12 5"
        ></polyline></svg
      >
    </a>
    <div>
      <h1
        class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase"
      >
        NOTIFICATION_CHANNELS
      </h1>
      <p class="text-cyan-500/80 mt-2 font-mono text-sm tracking-wide">
        CONFIGURE WHERE ALERTS SHOULD BE EXPLICITLY ROUTED WHEN THIS API FAILS.
      </p>
    </div>
  </div>

  <div
    class="bg-slate-900/60 backdrop-blur-xl border border-slate-700/50 rounded-3xl shadow-[0_8px_30px_rgb(0,0,0,0.5)] overflow-hidden"
  >
    <div class="p-6 md:p-8">
      <form on:submit|preventDefault={handleSave} class="space-y-8">
        <!-- Telegram -->
        <div
          class="rounded-xl border p-6 relative overflow-hidden transition-all {config.enable_telegram
            ? 'bg-sky-950/30 border-sky-500/50 shadow-[0_0_15px_rgba(14,165,233,0.15)]'
            : 'bg-slate-900/50 border-slate-700 hover:border-slate-500'}"
        >
          <div
            class="absolute right-0 top-0 w-32 h-32 bg-sky-400 opacity-5 rounded-bl-[100px] pointer-events-none"
          ></div>

          <div class="flex items-start justify-between gap-4">
            <div
              class="flex items-center gap-4 text-cyan-50 font-mono tracking-widest text-lg uppercase font-bold mb-4"
            >
              <div
                class="w-10 h-10 rounded-full bg-slate-950/50 flex items-center justify-center text-sky-400 border border-sky-500/30 {config.enable_telegram
                  ? 'shadow-[0_0_15px_rgba(56,189,248,0.5)]'
                  : ''}"
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
                  ><line x1="22" y1="2" x2="11" y2="13"></line><polygon
                    points="22 2 15 22 11 13 2 9 22 2"
                  ></polygon></svg
                >
              </div>
              TELEGRAM
            </div>

            <label class="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                bind:checked={config.enable_telegram}
                class="sr-only peer"
              />
              <div
                class="w-11 h-6 bg-slate-800 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-sky-500/50 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-slate-300 after:border-slate-500 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-sky-500 peer-checked:shadow-[0_0_10px_rgba(14,165,233,0.5)]"
              ></div>
            </label>
          </div>

          <div
            class="mt-4 transition-all"
            style="display: {config.enable_telegram ? 'block' : 'none'}"
          >
            <label
              for="tg_chat"
              class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
              >CHAT_ID</label
            >
            <input
              id="tg_chat"
              type="text"
              bind:value={config.telegram_chat_id}
              class="w-full px-4 py-3 bg-slate-950/50 rounded-xl border border-slate-700 text-sky-100 placeholder:text-slate-600 focus:border-sky-500/50 focus:ring-1 focus:ring-sky-500/50 block outline-none transition-all font-mono text-sm tracking-wide"
              placeholder="E.G. -10012345678"
            />
            <p
              class="text-[10px] text-slate-500 mt-2 font-mono tracking-widest uppercase font-bold"
            >
              THE CHAT ID OR GROUP ID WHERE N8N WILL DISPATCH THE TELEGRAM
              FALLBACK MESSAGE.
            </p>
          </div>
        </div>

        <!-- LINE -->
        <div
          class="rounded-xl border p-6 relative overflow-hidden transition-all {config.enable_line
            ? 'bg-emerald-950/30 border-emerald-500/50 shadow-[0_0_15px_rgba(16,185,129,0.15)]'
            : 'bg-slate-900/50 border-slate-700 hover:border-slate-500'}"
        >
          <div
            class="absolute right-0 top-0 w-32 h-32 bg-emerald-500 opacity-5 rounded-bl-[100px] pointer-events-none"
          ></div>

          <div class="flex items-start justify-between gap-4">
            <div
              class="flex items-center gap-4 text-cyan-50 font-mono tracking-widest text-lg uppercase font-bold mb-4"
            >
              <div
                class="w-10 h-10 rounded-full bg-slate-950/50 flex items-center justify-center text-emerald-400 border border-emerald-500/30 {config.enable_line
                  ? 'shadow-[0_0_15px_rgba(52,211,153,0.5)]'
                  : ''}"
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
                  ><path
                    d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"
                  ></path></svg
                >
              </div>
              LINE_NOTIFY
            </div>

            <label class="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                bind:checked={config.enable_line}
                class="sr-only peer"
              />
              <div
                class="w-11 h-6 bg-slate-800 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-emerald-500/50 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-slate-300 after:border-slate-500 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-500 peer-checked:shadow-[0_0_10px_rgba(16,185,129,0.5)]"
              ></div>
            </label>
          </div>

          <div
            class="mt-4 transition-all"
            style="display: {config.enable_line ? 'block' : 'none'}"
          >
            <label
              for="line_id"
              class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
              >LINE_USER/GROUP_ID</label
            >
            <input
              id="line_id"
              type="text"
              bind:value={config.line_user_id}
              class="w-full px-4 py-3 bg-slate-950/50 rounded-xl border border-slate-700 text-emerald-100 placeholder:text-slate-600 focus:border-emerald-500/50 focus:ring-1 focus:ring-emerald-500/50 block outline-none transition-all font-mono text-sm tracking-wide"
              placeholder="E.G. U1ABCDEFGHIJKLMN..."
            />
          </div>
        </div>

        <!-- System Ticketing -->
        <div
          class="rounded-xl border p-6 relative overflow-hidden transition-all {config.enable_ticketing
            ? 'bg-purple-950/30 border-purple-500/50 shadow-[0_0_15px_rgba(168,85,247,0.15)]'
            : 'bg-slate-900/50 border-slate-700 hover:border-slate-500'}"
        >
          <div
            class="absolute right-0 top-0 w-32 h-32 bg-purple-500 opacity-5 rounded-bl-[100px] pointer-events-none"
          ></div>

          <div class="flex items-start justify-between gap-4">
            <div
              class="flex items-center gap-4 text-cyan-50 font-mono tracking-widest text-lg uppercase font-bold mb-1"
            >
              <div
                class="w-10 h-10 rounded-full bg-slate-950/50 flex items-center justify-center text-purple-400 border border-purple-500/30 {config.enable_ticketing
                  ? 'shadow-[0_0_15px_rgba(192,132,252,0.5)]'
                  : ''}"
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
                  ><rect x="3" y="11" width="18" height="11" rx="2" ry="2"
                  ></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg
                >
              </div>
              EXTERNAL_TICKETING_SYSTEM
            </div>

            <label
              class="relative inline-flex items-center cursor-pointer mt-2"
            >
              <input
                type="checkbox"
                bind:checked={config.enable_ticketing}
                class="sr-only peer"
              />
              <div
                class="w-11 h-6 bg-slate-800 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-purple-500/50 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-slate-300 after:border-slate-500 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-purple-500 peer-checked:shadow-[0_0_10px_rgba(168,85,247,0.5)]"
              ></div>
            </label>
          </div>
          <p
            class="text-[10px] text-slate-400 ml-[56px] -mt-1 font-mono tracking-widest uppercase font-bold"
          >
            AUTOMATICALLY OPEN INCIDENTS IN JIRA, ZENDESK, OR SERVICENOW VIA
            N8N.
          </p>
        </div>

        <!-- Submission -->
        <div
          class="pt-4 border-t border-slate-700/50 flex items-center justify-between"
        >
          {#if saveSuccess}
            <span
              class="text-emerald-400 bg-emerald-950/50 px-3 py-1.5 rounded-lg border border-emerald-500/30 shadow-[0_0_10px_rgba(52,211,153,0.2)] font-mono text-[10px] uppercase font-bold tracking-widest flex items-center gap-2 fade-in"
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
                ><polyline points="20 6 9 17 4 12"></polyline></svg
              >
              SAVED_SUCCESSFULLY!
            </span>
          {:else}
            <div></div>
            <!-- Spacer -->
          {/if}

          <button
            type="submit"
            disabled={isSaving}
            class="px-8 py-3 font-mono text-xs font-bold tracking-widest uppercase rounded-xl transition-all flex items-center gap-2 bg-amber-950/80 text-amber-400 border border-amber-500/30 hover:bg-amber-900/80 hover:border-amber-400/50 hover:shadow-[0_0_15px_rgba(245,158,11,0.3)] disabled:opacity-50 min-w-[150px] justify-center"
          >
            {#if isSaving}
              <svg
                class="animate-spin h-5 w-5 text-amber-500"
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
              UPDATING...
            {:else}
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
                ><path
                  d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"
                ></path><polyline points="17 21 17 13 7 13 7 21"
                ></polyline><polyline points="7 3 7 8 15 8"></polyline></svg
              >
              SAVE_CONFIGURATION
            {/if}
          </button>
        </div>
      </form>
    </div>
  </div>
</div>
