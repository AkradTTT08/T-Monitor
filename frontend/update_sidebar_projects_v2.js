import fs from 'fs';

let content = fs.readFileSync('src/routes/dashboard/+layout.svelte', 'utf-8');

// 1. Add scripts to load projects and handle selection
const scriptInjection = `
  let projects: any[] = [];
  let selectedProjectId = '';

  // Listen to page changes to auto-select the right project ID
  $: {
    if ($page.params.id) {
      selectedProjectId = $page.params.id;
      if (typeof window !== 'undefined') {
         localStorage.setItem('monitor_selected_project', selectedProjectId);
      }
    }
  }

  onMount(async () => {
    // Load projects for the switcher
    try {
      const token = localStorage.getItem('monitor_token');
      const res = await fetch('http://localhost:5273/api/v1/projects', {
        headers: { 'Authorization': \`Bearer \${token}\` }
      });
      if (res.ok) {
        projects = await res.json();
        // check local storage if no route param
        if (!selectedProjectId) {
           const saved = localStorage.getItem('monitor_selected_project');
           if (saved) selectedProjectId = saved;
           else if (projects.length > 0) selectedProjectId = projects[0].id.toString();
        }
      }
    } catch (err) {
      console.error('Failed to load projects for sidebar', err);
    }
  });

  function handleProjectChange(e: Event) {
    const target = e.target as HTMLSelectElement;
    const newId = target.value;
    if (newId) {
      selectedProjectId = newId;
      localStorage.setItem('monitor_selected_project', newId);
      goto(\`/dashboard/projects/\${newId}\`);
    } else {
      goto('/dashboard');
    }
  }
`;

if (!content.includes('let selectedProjectId')) {
    content = content.replace("export let data;", "export let data;\n" + scriptInjection);
    if (!content.includes('goto')) {
        content = content.replace(`import { page } from "$app/stores";`, `import { page } from "$app/stores";\n  import { goto } from "$app/navigation";`);
    }
}

// 2. Add Project Switcher UI under T-Monitor Enterprise
const switcherUI = `
            {#if !isSidebarCollapsed}
              <div class="flex flex-col min-w-0 transition-opacity">
                <span class="font-bold text-slate-900 text-sm leading-tight truncate">T-Monitor</span>
                <span class="text-[10px] uppercase font-bold tracking-wider text-slate-400 truncate mt-0.5">Enterprise</span>
              </div>
            {/if}
          </div>

          {#if !isSidebarCollapsed}
          <div class="mt-4 px-1 w-full relative group/switcher">
            <select
              bind:value={selectedProjectId}
              on:change={handleProjectChange}
              class="w-full appearance-none bg-slate-100 border border-slate-200 text-slate-700 text-xs font-semibold rounded-lg px-3 py-2 pr-8 focus:outline-none focus:ring-2 focus:ring-blue-500 cursor-pointer shadow-sm truncate transition-colors hover:bg-slate-200"
            >
              <option value="" disabled>Select a project...</option>
              {#each projects as project}
                <option value={project.id.toString()}>{project.name}</option>
              {/each}
              <option value="">-- Manage Projects --</option>
            </select>
            <div class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-3 text-slate-500 group-hover/switcher:text-slate-700">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>
            </div>
          </div>
          {/if}
`;

content = content.replace(
    `            <div class="flex flex-col overflow-hidden">
              <span
                class="font-bold text-slate-900 text-sm leading-tight truncate"
                >Pantera Capital</span
              >
              <span class="text-xs text-slate-500 truncate"
                >Workspace • Main</span
              >
            </div>
          </div>`,
    switcherUI
).replace(/justify-between p-2/g, "flex-col items-start p-2");

// 3. Fix Project APIs Link
content = content.replace(
    `href="/dashboard"`,
    `href={selectedProjectId ? \`/dashboard/projects/\${selectedProjectId}\` : '/dashboard'}`
);

// 4. Fix current path checking for Project APIs
content = content.replace(
    `currentPath ===\n            '/dashboard'`,
    `($page.params.id ? currentPath === \`/dashboard/projects/\${$page.params.id}\` : currentPath === '/dashboard') && !$page.url.pathname.includes('/notifications')`
);

// 5. Add Projects Admin link
const projectsAdminLink = `
              <!-- Projects Admin Link -->
              <a href="/dashboard" on:click={() => isMobileMenuOpen = false} title="Projects" class="w-full flex items-center {isSidebarCollapsed ? 'justify-center px-0' : 'justify-between px-3'} py-3 rounded-xl text-sm font-semibold transition-all {currentPath === '/dashboard' && !$page.params.id ? 'bg-[#ecff82] text-slate-900 shadow-sm' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}">
                <div class="flex items-center gap-3">
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" class="{currentPath === '/dashboard' && !$page.params.id ? 'text-slate-900' : 'text-slate-500'}" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>
                  {#if !isSidebarCollapsed}<span>All Projects</span>{/if}
                </div>
              </a>
`;

if (!content.includes('<!-- Projects Admin Link -->')) {
    content = content.replace(
        `<!-- Manage Users Link -->`,
        projectsAdminLink + `\n              <!-- Manage Users Link -->`
    );
}

fs.writeFileSync('src/routes/dashboard/+layout.svelte', content);
console.log("Updated layout sidebar successfully");
