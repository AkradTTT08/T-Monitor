<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';

  let projectId = $page.params.id;
  let isLoadingConfig = false;
  let isSavingConfig = false;
  let saveSuccess = false;
  
  let notifConfig = {
    enable_telegram: false,
    telegram_chat_id: '',
    enable_line: false,
    line_user_id: '',
    enable_email: false,
    email_address: '',
    enable_ticketing: false
  };

  onMount(() => {
    fetchNotificationSettings();
  });

  async function fetchNotificationSettings() {
    isLoadingConfig = true;
    try {
      const token = localStorage.getItem('monitor_token');
      const res = await fetch(`http://localhost:5273/api/v1/projects/${projectId}/notifications`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });
      if (res.ok) {
        const data = await res.json();
        if (data.config) {
          notifConfig = {
            enable_telegram: data.config.enable_telegram || false,
            telegram_chat_id: data.config.telegram_chat_id || '',
            enable_line: data.config.enable_line || false,
            line_user_id: data.config.line_user_id || '',
            enable_email: data.config.enable_email || false,
            email_address: data.config.email_address || '',
            enable_ticketing: data.config.enable_ticketing || false
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
      const token = localStorage.getItem('monitor_token');
      const res = await fetch(`http://localhost:5273/api/v1/notifications`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          project_id: parseInt(projectId),
          ...notifConfig
        })
      });
      
      if (res.ok) {
        saveSuccess = true;
        setTimeout(() => saveSuccess = false, 3000);
      }
    } catch (err) {
      console.error(err);
    } finally {
      isSavingConfig = false;
    }
  }
</script>

<div class="h-full flex flex-col bg-white overflow-hidden p-8 max-w-4xl mx-auto w-full">
  <div class="mb-8">
    <div class="flex items-center gap-3 mb-2">
      <a href={`/dashboard/projects/${projectId}`} class="text-slate-400 hover:text-slate-600 transition-colors">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline></svg>
      </a>
      <h1 class="text-2xl font-black text-slate-800 tracking-tight flex items-center gap-3">
        Notification Channels 🔔
      </h1>
    </div>
    <p class="text-slate-500 font-medium ml-8">Manage where alerts are sent when endpoints in this project fail.</p>
  </div>

  {#if isLoadingConfig}
    <div class="flex justify-center p-24">
      <svg class="animate-spin h-8 w-8 text-amber-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
    </div>
  {:else}
    <div class="flex-1 overflow-y-auto space-y-6 pr-2 mb-20 scrollbar-hide">
      
      <div class="bg-amber-50 text-amber-800 p-4 rounded-xl border border-amber-100 flex items-start gap-3">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="shrink-0 mt-0.5"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
        <p class="text-sm">These settings apply to <strong>ALL APIs</strong> in this project. If any managed endpoint fails its health check, alerts will be broadcasted to the enabled channels below via your n8n workflows.</p>
      </div>

      <!-- Telegram Config -->
      <div class="border border-slate-200 rounded-2xl p-5 hover:border-slate-300 transition-all {notifConfig.enable_telegram ? 'bg-sky-50/20' : 'bg-white'}">
        <label class="flex items-center gap-3 cursor-pointer mb-4">
          <input type="checkbox" bind:checked={notifConfig.enable_telegram} class="w-5 h-5 text-sky-500 rounded border-slate-300 focus:ring-sky-500 transition-all">
          <div class="flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" class="text-sky-500" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg>
            <span class="font-bold text-slate-800 text-lg">Telegram</span>
          </div>
        </label>
        
        {#if notifConfig.enable_telegram}
          <div class="pl-8 fade-in space-y-3">
             <div class="bg-white p-4 rounded-xl border border-slate-100 placeholder-slate-200 shadow-sm">
               <label class="block text-xs font-bold text-slate-600 mb-2 uppercase tracking-wide">Telegram Chat ID</label>
               <input type="text" bind:value={notifConfig.telegram_chat_id} placeholder="e.g. -10012345678 or @YourChannelName" class="w-full bg-slate-50 border border-slate-200 text-slate-900 rounded-lg focus:ring-sky-500 focus:border-sky-500 block p-3 outline-none transition-all font-mono text-sm">
               <p class="text-[11px] text-slate-400 mt-2 font-medium">Add the bot to your channel and grab the ID from Telegram Web or use your @channelname.</p>
             </div>
          </div>
        {/if}
      </div>
      
      <!-- LINE Config -->
      <div class="border border-slate-200 rounded-2xl p-5 hover:border-slate-300 transition-all {notifConfig.enable_line ? 'bg-green-50/20' : 'bg-white'}">
        <label class="flex items-center gap-3 cursor-pointer mb-4">
          <input type="checkbox" bind:checked={notifConfig.enable_line} class="w-5 h-5 text-green-500 rounded border-slate-300 focus:ring-green-500 transition-all">
          <div class="flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" class="text-green-500" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"></path></svg>
            <span class="font-bold text-slate-800 text-lg">LINE Notify</span>
          </div>
        </label>
        
        {#if notifConfig.enable_line}
          <div class="pl-8 fade-in space-y-3">
             <div class="bg-white p-4 rounded-xl border border-slate-100 shadow-sm">
               <label class="block text-xs font-bold text-slate-600 mb-2 uppercase tracking-wide">User / Group ID</label>
               <input type="text" bind:value={notifConfig.line_user_id} placeholder="e.g. U1234567890abcdef" class="w-full bg-slate-50 border border-slate-200 text-slate-900 rounded-lg focus:ring-green-500 focus:border-green-500 block p-3 outline-none transition-all font-mono text-sm">
               <p class="text-[11px] text-slate-400 mt-2 font-medium">To alert a group, use the generated Group ID from your LINE Developer console.</p>
             </div>
          </div>
        {/if}
      </div>

      <!-- General Operations Config -->
      <div class="border border-slate-200 rounded-2xl p-5 hover:border-slate-300 transition-all bg-white cursor-pointer" on:click={() => notifConfig.enable_ticketing = !notifConfig.enable_ticketing}>
        <div class="flex items-center gap-3">
          <input type="checkbox" bind:checked={notifConfig.enable_ticketing} class="w-5 h-5 text-purple-600 rounded border-slate-300 focus:ring-purple-600 transition-all">
          <div class="flex flex-col">
            <span class="font-bold text-slate-800 text-lg">Enable Incident Ticketing</span>
            <span class="text-sm text-slate-500 mt-1 font-medium">Auto-create tickets in Jira/Trello via n8n integration when critical errors occur.</span>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Fixed Bottom Action Bar -->
    <div class="absolute bottom-0 left-0 right-0 p-6 bg-white/80 backdrop-blur-md border-t border-slate-200 flex justify-between items-center px-8">
      <div>
        {#if saveSuccess}
          <div class="flex items-center gap-2 text-emerald-600 bg-emerald-50 px-3 py-1.5 rounded-lg border border-emerald-200 fade-in text-sm font-bold">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
            Settings saved successfully
          </div>
        {/if}
      </div>
      <div class="flex gap-4">
        <a href={`/dashboard/projects/${projectId}`} class="px-6 py-3 text-slate-600 hover:text-slate-900 bg-slate-100 hover:bg-slate-200 rounded-xl font-bold transition-colors shadow-sm text-sm border border-slate-200">
          Cancel & Return
        </a>
        <button on:click={saveNotificationSettings} disabled={isSavingConfig} class="px-8 py-3 bg-amber-500 text-white rounded-xl hover:bg-amber-600 font-bold transition-all shadow-md hover:shadow-lg text-sm flex items-center gap-2 disabled:opacity-70 disabled:shadow-none min-w-[150px] justify-center">
          {#if isSavingConfig}
            <svg class="animate-spin h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
            Saving...
          {:else}
            Save Configuration
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
    from { opacity: 0; transform: translateY(-5px); }
    to { opacity: 1; transform: translateY(0); }
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
