<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";

  let projectId = $page.params.id;
  let isLoadingConfig = false;
  let isSavingConfig = false;
  let saveSuccess = false;

  let notifConfig = {
    enable_telegram: false,
    telegram_bot_token: "",
    telegram_chat_id: "",
    enable_line: false,
    line_user_id: "",
    enable_email: false,
    email_address: "",
    smtp_host: "",
    smtp_port: 587,
    smtp_user: "",
    smtp_pass: "",
    enable_ticketing: false,
  };

  onMount(() => {
    fetchNotificationSettings();
  });

  async function fetchNotificationSettings() {
    isLoadingConfig = true;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(
        `http://localhost:5273/api/v1/projects/${projectId}/notifications`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      );
      if (res.ok) {
        const data = await res.json();
        if (data.config) {
          notifConfig = {
            enable_telegram: data.config.enable_telegram || false,
            telegram_bot_token: data.config.telegram_bot_token || "",
            telegram_chat_id: data.config.telegram_chat_id || "",
            enable_line: data.config.enable_line || false,
            line_user_id: data.config.line_user_id || "",
            enable_email: data.config.enable_email || false,
            email_address: data.config.email_address || "",
            smtp_host: data.config.smtp_host || "",
            smtp_port: data.config.smtp_port || 587,
            smtp_user: data.config.smtp_user || "",
            smtp_pass: data.config.smtp_pass || "",
            enable_ticketing: data.config.enable_ticketing || false,
          };
        }
      }
    } catch (err) {
      console.error(err);
    } finally {
      isLoadingConfig = false;
    }
  }

  async function saveNotificationSettings() {
    isSavingConfig = true;
    saveSuccess = false;
    try {
      const token = localStorage.getItem("monitor_token");
      const res = await fetch(`http://localhost:5273/api/v1/notifications`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          project_id: parseInt(projectId || "0"),
          ...notifConfig,
        }),
      });

      if (res.ok) {
        saveSuccess = true;
        setTimeout(() => (saveSuccess = false), 3000);
      }
    } catch (err) {
      console.error(err);
    } finally {
      isSavingConfig = false;
    }
  }
</script>

<div
  class="h-full flex flex-col bg-slate-900/60 backdrop-blur-xl border border-slate-700/50 rounded-3xl overflow-hidden p-8 max-w-4xl mx-auto w-full shadow-[0_8px_30px_rgb(0,0,0,0.5)] relative"
>
  <div class="mb-8">
    <div class="flex items-center gap-3 mb-2">
      <a
        href={`/dashboard/projects/${projectId}`}
        class="text-cyan-500/50 hover:text-cyan-400 transition-colors"
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
      <h1
        class="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-blue-400 tracking-tight font-mono uppercase flex items-center gap-3"
      >
        NOTIFICATION_CHANNELS 🔔
      </h1>
    </div>
    <p class="text-cyan-500/80 font-mono tracking-wide text-sm ml-10">
      MANAGE WHERE ALERTS ARE SENT WHEN ENDPOINTS IN THIS PROJECT FAIL.
    </p>
  </div>

  {#if isLoadingConfig}
    <div class="flex justify-center p-24">
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
    <div class="flex-1 overflow-y-auto space-y-6 pr-2 mb-20 scrollbar-hide">
      <div
        class="bg-amber-950/50 text-amber-400 p-4 rounded-xl border border-amber-500/30 flex items-start gap-3 shadow-[0_0_15px_rgba(245,158,11,0.15)] font-mono text-[10px] uppercase font-bold tracking-widest leading-loose"
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
          class="shrink-0 mt-0.5"
          ><circle cx="12" cy="12" r="10"></circle><line
            x1="12"
            y1="16"
            x2="12"
            y2="12"
          ></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg
        >
        <p class="text-xs">
          THESE SETTINGS APPLY TO <strong>ALL APIS</strong> IN THIS PROJECT. IF ANY
          MANAGED ENDPOINT FAILS ITS HEALTH CHECK, ALERTS WILL BE BROADCASTED TO
          THE ENABLED CHANNELS BELOW VIA YOUR N8N WORKFLOWS.
        </p>
      </div>

      <!-- Telegram Config -->
      <div
        class="border rounded-2xl p-5 transition-all {notifConfig.enable_telegram
          ? 'bg-sky-950/30 border-sky-500/50 shadow-[0_0_15px_rgba(14,165,233,0.15)]'
          : 'bg-slate-900/50 border-slate-700 hover:border-slate-500'}"
      >
        <label class="flex items-center gap-3 cursor-pointer mb-4">
          <input
            type="checkbox"
            bind:checked={notifConfig.enable_telegram}
            class="w-5 h-5 text-sky-500 bg-slate-900 rounded border-slate-600 focus:ring-sky-500/50 focus:ring-offset-slate-900 transition-all cursor-pointer appearance-none checked:bg-sky-500 checked:border-sky-500 relative before:content-[''] checked:before:absolute checked:before:inset-0 checked:before:flex checked:before:items-center checked:before:justify-center"
          />
          <div class="flex items-center gap-2">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="22"
              height="22"
              viewBox="0 0 24 24"
              fill="none"
              class="text-sky-400 {notifConfig.enable_telegram
                ? 'drop-shadow-[0_0_8px_rgba(56,189,248,0.8)]'
                : ''}"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><line x1="22" y1="2" x2="11" y2="13"></line><polygon
                points="22 2 15 22 11 13 2 9 22 2"
              ></polygon></svg
            >
            <span
              class="font-bold text-cyan-50 font-mono tracking-widest text-lg uppercase"
              >TELEGRAM</span
            >
          </div>
        </label>

        {#if notifConfig.enable_telegram}
          <div class="pl-8 fade-in space-y-3">
            <!-- Bot Token -->
            <div
              class="bg-slate-900/80 p-4 rounded-xl border border-slate-700/50 shadow-inner"
            >
              <label
                class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                >TELEGRAM_BOT_TOKEN</label
              >
              <input
                type="text"
                bind:value={notifConfig.telegram_bot_token}
                placeholder="E.G. 123456789:ABCDEF..."
                class="w-full bg-slate-950/50 border border-slate-700 text-sky-100 rounded-lg focus:border-sky-500/50 focus:ring-1 focus:ring-sky-500/50 block p-3 outline-none transition-all font-mono text-sm tracking-wide placeholder:text-slate-600"
              />
              <p
                class="text-[10px] text-slate-500 mt-2 font-mono tracking-widest uppercase font-bold"
              >
                GET YOUR BOT TOKEN FROM @BOTFATHER ON TELEGRAM.
              </p>
            </div>
            <!-- Chat ID -->
            <div
              class="bg-slate-900/80 p-4 rounded-xl border border-slate-700/50 shadow-inner"
            >
              <label
                class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                >TELEGRAM_CHAT_ID</label
              >
              <input
                type="text"
                bind:value={notifConfig.telegram_chat_id}
                placeholder="E.G. -10012345678 OR @YOURCHANNELNAME"
                class="w-full bg-slate-950/50 border border-slate-700 text-sky-100 rounded-lg focus:border-sky-500/50 focus:ring-1 focus:ring-sky-500/50 block p-3 outline-none transition-all font-mono text-sm tracking-wide placeholder:text-slate-600"
              />
              <p
                class="text-[10px] text-slate-500 mt-2 font-mono tracking-widest uppercase font-bold"
              >
                ADD THE BOT TO YOUR CHANNEL AND GRAB THE ID FROM TELEGRAM WEB OR
                USE YOUR @CHANNELNAME.
              </p>
            </div>
          </div>
        {/if}
      </div>

      <!-- LINE Config -->
      <div
        class="border rounded-2xl p-5 transition-all {notifConfig.enable_line
          ? 'bg-emerald-950/30 border-emerald-500/50 shadow-[0_0_15px_rgba(16,185,129,0.15)]'
          : 'bg-slate-900/50 border-slate-700 hover:border-slate-500'}"
      >
        <label class="flex items-center gap-3 cursor-pointer mb-4">
          <input
            type="checkbox"
            bind:checked={notifConfig.enable_line}
            class="w-5 h-5 text-emerald-500 bg-slate-900 rounded border-slate-600 focus:ring-emerald-500/50 focus:ring-offset-slate-900 transition-all cursor-pointer appearance-none checked:bg-emerald-500 checked:border-emerald-500 relative before:content-[''] checked:before:absolute checked:before:inset-0 checked:before:flex checked:before:items-center checked:before:justify-center"
          />
          <div class="flex items-center gap-2">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="22"
              height="22"
              viewBox="0 0 24 24"
              fill="none"
              class="text-emerald-400 {notifConfig.enable_line
                ? 'drop-shadow-[0_0_8px_rgba(52,211,153,0.8)]'
                : ''}"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><path
                d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"
              ></path></svg
            >
            <span
              class="font-bold text-cyan-50 font-mono tracking-widest text-lg uppercase"
              >LINE_NOTIFY</span
            >
          </div>
        </label>

        {#if notifConfig.enable_line}
          <div class="pl-8 fade-in space-y-3">
            <div
              class="bg-slate-900/80 p-4 rounded-xl border border-slate-700/50 shadow-inner"
            >
              <label
                class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                >USER_/_GROUP_ID</label
              >
              <input
                type="text"
                bind:value={notifConfig.line_user_id}
                placeholder="E.G. U1234567890ABCDEF"
                class="w-full bg-slate-950/50 border border-slate-700 text-emerald-100 rounded-lg focus:border-emerald-500/50 focus:ring-1 focus:ring-emerald-500/50 block p-3 outline-none transition-all font-mono text-sm tracking-wide placeholder:text-slate-600"
              />
              <p
                class="text-[10px] text-slate-500 mt-2 font-mono tracking-widest uppercase font-bold"
              >
                TO ALERT A GROUP, USE THE GENERATED GROUP ID FROM YOUR LINE
                DEVELOPER CONSOLE.
              </p>
            </div>
          </div>
        {/if}
      </div>

      <!-- Gmail Config -->
      <div
        class="border rounded-2xl p-5 transition-all {notifConfig.enable_email
          ? 'bg-red-950/30 border-red-500/50 shadow-[0_0_15px_rgba(239,68,68,0.15)]'
          : 'bg-slate-900/50 border-slate-700 hover:border-slate-500'}"
      >
        <label class="flex items-center gap-3 cursor-pointer mb-4">
          <input
            type="checkbox"
            bind:checked={notifConfig.enable_email}
            class="w-5 h-5 text-red-500 bg-slate-900 rounded border-slate-600 focus:ring-red-500/50 focus:ring-offset-slate-900 transition-all cursor-pointer appearance-none checked:bg-red-500 checked:border-red-500 relative before:content-[''] checked:before:absolute checked:before:inset-0 checked:before:flex checked:before:items-center checked:before:justify-center"
          />
          <div class="flex items-center gap-2">
            <!-- Gmail M-shaped icon -->
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="22"
              height="22"
              viewBox="0 0 24 24"
              fill="none"
              class="text-red-400 {notifConfig.enable_email
                ? 'drop-shadow-[0_0_8px_rgba(239,68,68,0.8)]'
                : ''}"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              ><rect x="2" y="4" width="20" height="16" rx="2"></rect><path
                d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"
              ></path></svg
            >
            <span
              class="font-bold text-cyan-50 font-mono tracking-widest text-lg uppercase"
              >GMAIL</span
            >
          </div>
        </label>

        {#if notifConfig.enable_email}
          <div class="pl-8 fade-in space-y-4">
            <!-- Multi-Email Recipients -->
            <div
              class="bg-slate-900/80 p-4 rounded-xl border border-slate-700/50 shadow-inner"
            >
              <label
                class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                >RECIPIENT_EMAIL_ADDRESSES (COMMA SEPARATED)</label
              >
              <textarea
                bind:value={notifConfig.email_address}
                placeholder="E.G. user1@gmail.com, user2@gmail.com"
                rows="2"
                class="w-full bg-slate-950/50 border border-slate-700 text-red-100 rounded-lg focus:border-red-500/50 focus:ring-1 focus:ring-red-500/50 block p-3 outline-none transition-all font-mono text-sm tracking-wide placeholder:text-slate-600 resize-none"
              ></textarea>
              <p
                class="text-[10px] text-slate-500 mt-2 font-mono tracking-widest uppercase font-bold"
              >
                SEPARATE MULTIPLE EMAILS WITH COMMAS.
              </p>
            </div>

            <!-- SMTP Settings -->
            <div class="grid grid-cols-2 gap-4">
              <div
                class="bg-slate-900/80 p-4 rounded-xl border border-slate-700/50 shadow-inner"
              >
                <label
                  class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                  >SMTP_HOST</label
                >
                <input
                  type="text"
                  bind:value={notifConfig.smtp_host}
                  placeholder="smtp.gmail.com"
                  class="w-full bg-slate-950/50 border border-slate-700 text-red-100 rounded-lg focus:border-red-500/50 focus:ring-1 focus:ring-red-500/50 block p-3 outline-none transition-all font-mono text-sm tracking-wide placeholder:text-slate-600"
                />
              </div>
              <div
                class="bg-slate-900/80 p-4 rounded-xl border border-slate-700/50 shadow-inner"
              >
                <label
                  class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                  >SMTP_PORT</label
                >
                <input
                  type="number"
                  bind:value={notifConfig.smtp_port}
                  placeholder="587"
                  class="w-full bg-slate-950/50 border border-slate-700 text-red-100 rounded-lg focus:border-red-500/50 focus:ring-1 focus:ring-red-500/50 block p-3 outline-none transition-all font-mono text-sm tracking-wide placeholder:text-slate-600"
                />
              </div>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div
                class="bg-slate-900/80 p-4 rounded-xl border border-slate-700/50 shadow-inner"
              >
                <label
                  class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                  >SMTP_USER</label
                >
                <input
                  type="text"
                  bind:value={notifConfig.smtp_user}
                  placeholder="your-email@gmail.com"
                  class="w-full bg-slate-950/50 border border-slate-700 text-red-100 rounded-lg focus:border-red-500/50 focus:ring-1 focus:ring-red-500/50 block p-3 outline-none transition-all font-mono text-sm tracking-wide placeholder:text-slate-600"
                />
              </div>
              <div
                class="bg-slate-900/80 p-4 rounded-xl border border-slate-700/50 shadow-inner"
              >
                <label
                  class="block text-[10px] font-bold text-slate-400 font-mono tracking-widest uppercase mb-2"
                  >SMTP_PASSWORD</label
                >
                <input
                  type="password"
                  bind:value={notifConfig.smtp_pass}
                  placeholder="App Password"
                  class="w-full bg-slate-950/50 border border-slate-700 text-red-100 rounded-lg focus:border-red-500/50 focus:ring-1 focus:ring-red-500/50 block p-3 outline-none transition-all font-mono text-sm tracking-wide placeholder:text-slate-600"
                />
              </div>
            </div>
          </div>
        {/if}
      </div>

      <!-- General Operations Config -->
      <div
        class="border rounded-2xl p-5 transition-all cursor-pointer {notifConfig.enable_ticketing
          ? 'bg-purple-950/30 border-purple-500/50 shadow-[0_0_15px_rgba(168,85,247,0.15)]'
          : 'bg-slate-900/50 border-slate-700 hover:border-slate-500'}"
        on:click={() =>
          (notifConfig.enable_ticketing = !notifConfig.enable_ticketing)}
      >
        <div class="flex items-center gap-3">
          <input
            type="checkbox"
            bind:checked={notifConfig.enable_ticketing}
            class="w-5 h-5 text-purple-500 bg-slate-900 rounded border-slate-600 focus:ring-purple-500/50 focus:ring-offset-slate-900 transition-all cursor-pointer appearance-none checked:bg-purple-500 checked:border-purple-500 relative before:content-[''] checked:before:absolute checked:before:inset-0 checked:before:flex checked:before:items-center checked:before:justify-center"
          />
          <div class="flex flex-col">
            <span
              class="font-bold text-cyan-50 font-mono tracking-widest text-lg uppercase {notifConfig.enable_ticketing
                ? 'drop-shadow-[0_0_8px_rgba(192,132,252,0.5)]'
                : ''}">ENABLE_INCIDENT_TICKETING</span
            >
            <span
              class="text-[10px] text-slate-400 mt-1 font-mono tracking-widest uppercase font-bold"
              >AUTO-CREATE TICKETS IN JIRA/TRELLO VIA N8N INTEGRATION WHEN
              CRITICAL ERRORS OCCUR.</span
            >
          </div>
        </div>
      </div>
    </div>

    <!-- Fixed Bottom Action Bar -->
    <div
      class="absolute bottom-0 left-0 right-0 p-6 bg-slate-950/80 backdrop-blur-md border-t border-slate-800 flex justify-between items-center px-8 rounded-b-3xl"
    >
      <div>
        {#if saveSuccess}
          <div
            class="flex items-center gap-2 text-emerald-400 bg-emerald-950/50 px-3 py-1.5 rounded-lg border border-emerald-500/30 shadow-[0_0_10px_rgba(52,211,153,0.2)] fade-in font-mono text-[10px] uppercase font-bold tracking-widest"
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
              ><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline
                points="22 4 12 14.01 9 11.01"
              ></polyline></svg
            >
            SETTINGS_SAVED_SUCCESSFULLY
          </div>
        {/if}
      </div>
      <div class="flex gap-4">
        <a
          href={`/dashboard/projects/${projectId}`}
          class="px-6 py-3 font-mono text-xs font-bold tracking-widest uppercase rounded-xl transition-all flex items-center gap-2 bg-slate-800 text-slate-400 border border-slate-700 hover:bg-slate-700 hover:text-cyan-50"
        >
          CANCEL_&_RETURN
        </a>
        <button
          on:click={saveNotificationSettings}
          disabled={isSavingConfig}
          class="px-8 py-3 font-mono text-xs font-bold tracking-widest uppercase rounded-xl transition-all flex items-center gap-2 bg-amber-950/80 text-amber-400 border border-amber-500/30 hover:bg-amber-900/80 hover:border-amber-400/50 hover:shadow-[0_0_15px_rgba(245,158,11,0.3)] disabled:opacity-50 min-w-[150px] justify-center"
        >
          {#if isSavingConfig}
            <svg
              class="animate-spin h-4 w-4 text-amber-500"
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
            SAVING...
          {:else}
            SAVE_CONFIGURATION
          {/if}
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .fade-in {
    animation: fadeIn 0.3s ease-out;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(-5px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  /* Hide scrollbar for Chrome, Safari and Opera */
  .scrollbar-hide::-webkit-scrollbar {
    display: none;
  }

  /* Hide scrollbar for IE, Edge and Firefox */
  .scrollbar-hide {
    -ms-overflow-style: none; /* IE and Edge */
    scrollbar-width: none; /* Firefox */
  }
</style>
