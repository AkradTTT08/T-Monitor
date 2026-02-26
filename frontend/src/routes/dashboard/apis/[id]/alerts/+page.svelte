<script lang="ts">
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  
  let apiId = $page.params.id;
  
  let isSaving = false;
  let saveSuccess = false;
  
  let config = {
    api_id: parseInt(apiId),
    enable_telegram: false,
    telegram_chat_id: '',
    enable_line: false,
    line_user_id: '',
    enable_email: false,
    email_address: '',
    enable_ticketing: false
  };

  onMount(async () => {
    await fetchApiAndConfig();
  });

  async function fetchApiAndConfig() {
    try {
      const token = localStorage.getItem('monitor_token');
      // Fetch the API to get its current notification config
      const res = await fetch(`http://localhost:5273/api/v1/apis?project_id=`, {
        headers: { 'Authorization': `Bearer ${token}` }
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
      const token = localStorage.getItem('monitor_token');
      const res = await fetch(`http://localhost:5273/api/v1/notifications`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(config)
      });
      
      if (res.ok) {
        saveSuccess = true;
        setTimeout(() => saveSuccess = false, 3000);
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
    <a href="/dashboard/apis" class="p-2 bg-white border border-slate-200 rounded-xl text-slate-500 hover:text-blue-600 hover:border-blue-200 transition-colors shadow-sm">
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline></svg>
    </a>
    <div>
      <h1 class="text-3xl font-bold text-slate-900 tracking-tight">Notification Channels</h1>
      <p class="text-slate-500 mt-1">Configure where alerts should be explicitly routed when this API fails.</p>
    </div>
  </div>

  <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
    
    <div class="p-6 md:p-8">
      <form on:submit|preventDefault={handleSave} class="space-y-8">
        
        <!-- Telegram -->
        <div class="bg-slate-50 rounded-xl border border-slate-200 p-6 relative overflow-hidden transition-all hover:shadow-md">
          <div class="absolute right-0 top-0 w-32 h-32 bg-blue-400 opacity-5 rounded-bl-[100px] pointer-events-none"></div>
          
          <div class="flex items-start justify-between gap-4">
            <div class="flex items-center gap-4 text-slate-800 font-bold text-lg mb-4">
               <div class="w-10 h-10 rounded-full bg-[#0088cc] flex items-center justify-center text-white shadow">
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg>
               </div>
               Telegram
            </div>
            
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" bind:checked={config.enable_telegram} class="sr-only peer">
              <div class="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[#0088cc]"></div>
            </label>
          </div>
          
          <div class="mt-4 transition-all" style="display: {config.enable_telegram ? 'block' : 'none'}">
            <label for="tg_chat" class="block text-sm font-medium text-slate-700 mb-1.5">Chat ID</label>
            <input 
              id="tg_chat" 
              type="text" 
              bind:value={config.telegram_chat_id}
              class="w-full px-4 py-3 bg-white rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-[#0088cc]/50 transition-all text-sm font-mono"
              placeholder="e.g. -10012345678"
            />
            <p class="text-xs text-slate-500 mt-2">The Chat ID or Group ID where n8n will dispatch the Telegram fallback message.</p>
          </div>
        </div>

        <!-- LINE -->
        <div class="bg-slate-50 rounded-xl border border-slate-200 p-6 relative overflow-hidden transition-all hover:shadow-md">
           <div class="absolute right-0 top-0 w-32 h-32 bg-green-500 opacity-5 rounded-bl-[100px] pointer-events-none"></div>

          <div class="flex items-start justify-between gap-4">
            <div class="flex items-center gap-4 text-slate-800 font-bold text-lg mb-4">
               <div class="w-10 h-10 rounded-full bg-[#00B900] flex items-center justify-center text-white shadow">
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"></path></svg>
               </div>
               LINE Notify
            </div>
            
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" bind:checked={config.enable_line} class="sr-only peer">
              <div class="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-green-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[#00B900]"></div>
            </label>
          </div>
          
          <div class="mt-4 transition-all" style="display: {config.enable_line ? 'block' : 'none'}">
            <label for="line_id" class="block text-sm font-medium text-slate-700 mb-1.5">LINE User/Group ID</label>
            <input 
              id="line_id" 
              type="text" 
              bind:value={config.line_user_id}
              class="w-full px-4 py-3 bg-white rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-[#00B900]/50 transition-all text-sm font-mono"
              placeholder="e.g. U1abcdefghijklmn..."
            />
          </div>
        </div>

        <!-- System Ticketing -->
        <div class="bg-slate-50 rounded-xl border border-slate-200 p-6 relative overflow-hidden transition-all hover:shadow-md">
           <div class="absolute right-0 top-0 w-32 h-32 bg-slate-600 opacity-5 rounded-bl-[100px] pointer-events-none"></div>

          <div class="flex items-start justify-between gap-4">
            <div class="flex items-center gap-4 text-slate-800 font-bold text-lg mb-1">
               <div class="w-10 h-10 rounded-full bg-slate-800 flex items-center justify-center text-white shadow">
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
               </div>
               External Ticketing System
            </div>
            
            <label class="relative inline-flex items-center cursor-pointer mt-2">
              <input type="checkbox" bind:checked={config.enable_ticketing} class="sr-only peer">
              <div class="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-slate-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-slate-800"></div>
            </label>
          </div>
          <p class="text-sm text-slate-500 ml-[56px] -mt-1">Automatically open incidents in Jira, Zendesk, or ServiceNow via n8n.</p>
        </div>

        <!-- Submission -->
        <div class="pt-4 border-t border-slate-100 flex items-center justify-between">
          {#if saveSuccess}
            <span class="text-green-600 font-medium text-sm flex items-center gap-2 fade-in">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
              Saved successfully!
            </span>
          {:else}
            <div></div> <!-- Spacer -->
          {/if}
          
          <button 
            type="submit" 
            disabled={isSaving}
            class="bg-blue-600 hover:bg-blue-700 text-white font-medium py-3 px-8 rounded-xl shadow-md shadow-blue-500/20 transition-all hover:-translate-y-0.5 disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none flex items-center gap-2"
          >
            {#if isSaving}
              <svg class="animate-spin h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
              Updating...
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path><polyline points="17 21 17 13 7 13 7 21"></polyline><polyline points="7 3 7 8 15 8"></polyline></svg>
              Save Configuration
            {/if}
          </button>
        </div>

      </form>
    </div>
  </div>
</div>
