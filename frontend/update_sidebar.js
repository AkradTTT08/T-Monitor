import fs from 'fs';
let content = fs.readFileSync('src/routes/dashboard/+layout.svelte', 'utf-8');

const navStart = content.indexOf('<!-- Main Navigation -->');
const asideEnd = content.indexOf('</aside>', navStart);

const newNav = `<!-- Main Navigation -->
      <div class="flex-1 overflow-y-auto space-y-1.5 {isSidebarCollapsed ? 'px-0' : 'pr-1'} hide-scrollbar">
        <a href="/dashboard" on:click={() => isMobileMenuOpen = false} title="Project APIs" class="w-full flex items-center {isSidebarCollapsed ? 'justify-center px-0' : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath === '/dashboard' ? 'bg-[#ecff82] text-slate-900 shadow-sm' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}">
          <div class="flex items-center gap-3">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" class="{currentPath === '/dashboard' ? 'text-slate-900' : 'text-slate-500'}" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="9" rx="1"/><rect x="14" y="3" width="7" height="5" rx="1"/><rect x="14" y="12" width="7" height="9" rx="1"/><rect x="3" y="16" width="7" height="5" rx="1"/></svg>
            {#if !isSidebarCollapsed}<span>Project APIs</span>{/if}
          </div>
        </a>

        <!-- Open APIs Link -->
        <a href="/dashboard/apis" on:click={() => isMobileMenuOpen = false} title="Open APIs" class="w-full flex items-center {isSidebarCollapsed ? 'justify-center px-0 relative' : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath === '/dashboard/apis' ? 'bg-[#ecff82] text-slate-900 shadow-sm' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}">
          <div class="flex items-center gap-3">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" class="{currentPath === '/dashboard/apis' ? 'text-slate-900' : 'text-slate-500'}" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v20"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
            {#if !isSidebarCollapsed}<span>Open APIs</span>{/if}
          </div>
          {#if !isSidebarCollapsed}
          <span class="bg-slate-900 text-white text-[10px] font-bold px-2 py-0.5 rounded-md">8</span>
          {:else}
          <span class="absolute top-1.5 right-1.5 w-2.5 h-2.5 bg-slate-900 border-2 border-slate-50 rounded-full"></span>
          {/if}
        </a>

        <!-- Status Live Link -->
        <a href="/dashboard/status" on:click={() => isMobileMenuOpen = false} title="Status Live" class="w-full flex items-center {isSidebarCollapsed ? 'justify-center px-0' : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath === '/dashboard/status' ? 'bg-[#ecff82] text-slate-900 shadow-sm' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}">
          <div class="flex items-center gap-3">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" class="{currentPath === '/dashboard/status' ? 'text-slate-900' : 'text-slate-500'}" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>
            {#if !isSidebarCollapsed}<span>Status Live</span>{/if}
          </div>
        </a>

        <!-- Notification Channels Link -->
        <a href={\`/dashboard/projects/\${$page.params.id || '1'}/notifications\`} on:click={() => isMobileMenuOpen = false} title="Alerts & Channels" class="w-full flex items-center {isSidebarCollapsed ? 'justify-center px-0' : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath.includes('notifications') ? 'bg-amber-100 text-amber-900 shadow-sm' : 'text-slate-600 hover:bg-amber-50 hover:text-amber-700'}">
          <div class="flex items-center gap-3">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" class="{currentPath.includes('notifications') ? 'text-amber-700' : 'text-amber-500'}" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>
            {#if !isSidebarCollapsed}<span>Alerts & Channels</span>{/if}
          </div>
        </a>

        {#if user.role === 'admin'}
        <div class="pt-4 pb-1">
          {#if !isSidebarCollapsed}
          <div class="px-3 flex items-center justify-between text-[10px] uppercase font-bold text-slate-400 tracking-wider mb-2">
            <span>Administration</span>
          </div>
          {:else}
          <div class="w-full h-px bg-slate-200 mb-2"></div>
          {/if}
          <!-- Manage Users Link -->
          <a href="/dashboard/users" on:click={() => isMobileMenuOpen = false} title="Users & Roles" class="w-full flex items-center {isSidebarCollapsed ? 'justify-center px-0' : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath === '/dashboard/users' ? 'bg-[#ecff82] text-slate-900 shadow-sm' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}">
            <div class="flex items-center gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" class="{currentPath === '/dashboard/users' ? 'text-slate-900' : 'text-slate-500'}" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
              {#if !isSidebarCollapsed}<span>Users & Roles</span>{/if}
            </div>
          </a>
        </div>
        {/if}
      </div>

      <!-- Bottom User Section -->
      <div class="mt-auto pt-4 border-t border-slate-200 flex flex-col gap-4">
        <!-- Help Center -->
        <a href="#" class="w-full flex items-center {isSidebarCollapsed ? 'justify-center px-0' : 'gap-3 px-3'} py-2 rounded-xl text-sm font-semibold transition-all text-slate-500 hover:bg-slate-100 hover:text-slate-900" title="Help center">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>
          {#if !isSidebarCollapsed}<span>Help center</span>{/if}
        </a>

        <!-- User Info -->
        <div class="w-full flex flex-col {isSidebarCollapsed ? 'pt-0' : 'pt-2'}">
          <button on:click={handleLogout} title={user?.email || 'Logout'} class="w-full flex items-center {isSidebarCollapsed ? 'justify-center p-0' : 'justify-between p-2'} rounded-xl hover:bg-slate-100 transition-all text-left group">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-full bg-slate-900 flex shrink-0 items-center justify-center text-white font-bold text-sm shadow-sm uppercase relative border-2 border-slate-100">
                {user?.email?.charAt(0) || 'U'}
                <span class="absolute bottom-[-2px] right-[-2px] w-3.5 h-3.5 bg-blue-600 text-white rounded-full flex items-center justify-center text-[8px] font-bold border-2 border-white">S</span>
              </div>
              {#if !isSidebarCollapsed}
              <div class="flex flex-col overflow-hidden min-w-0">
                <span class="font-bold text-slate-900 text-sm leading-tight truncate capitalize">{user?.email?.split('@')[0] || 'User'}</span>
                <span class="text-xs text-slate-500 font-medium truncate">{user?.email || 'user@example.com'}</span>
              </div>
              {/if}
            </div>
            {#if !isSidebarCollapsed}
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-slate-400 group-hover:text-red-500 transition-colors"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" x2="9" y1="12" y2="12"/></svg>
            {/if}
          </button>
          
          <!-- Progress Bar (Static mock) -->
          {#if !isSidebarCollapsed}
          <div class="px-2 mt-4 animate-in fade-in duration-300">
            <div class="flex items-center justify-between gap-3 text-[11px] font-bold text-slate-800 mb-1.5">
              <div class="flex-1 h-1.5 bg-slate-200 rounded-full overflow-hidden">
                <div class="h-full bg-emerald-500 rounded-full w-[70%] shadow-sm"></div>
              </div>
              <span>70%</span>
            </div>
            <span class="text-[11px] font-medium text-slate-500">Complete your profile</span>
          </div>
          {/if}
        </div>
      </div>
    
`;

content = content.slice(0, navStart) + newNav + content.slice(asideEnd);
fs.writeFileSync('src/routes/dashboard/+layout.svelte', content);
console.log("Updated layout successfully");
