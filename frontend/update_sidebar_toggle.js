import fs from 'fs';

let content = fs.readFileSync('src/routes/dashboard/+layout.svelte', 'utf-8');

const startMarker = '        <!-- Top Profile/Org block -->';
const endMarker = '        <!-- Main Navigation -->';

const startIdx = content.indexOf(startMarker);
const endIdx = content.indexOf(endMarker);

if (startIdx === -1 || endIdx === -1) {
  console.error('❌ Could not find block boundaries');
  console.log('startMarker found at:', startIdx);
  console.log('endMarker found at:', endIdx);
  process.exit(1);
}

const newBlock = `        <!-- Top Profile/Org block -->
        <div class="flex items-center p-2 mb-6 {isSidebarCollapsed ? 'justify-center' : 'gap-3'}">
          <div class="w-10 h-10 rounded-xl bg-slate-900 flex shrink-0 items-center justify-center text-white shadow-sm">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.2 7.8l-7.7 7.7-4-4-5.7 5.7" /><path d="M15 7h6v6" /></svg>
          </div>
          {#if !isSidebarCollapsed}
            <div class="flex flex-col overflow-hidden min-w-0 flex-1">
              <span class="font-bold text-slate-900 text-sm leading-tight truncate">T-Monitor</span>
              <span class="text-xs text-slate-500 truncate">Enterprise</span>
            </div>
          {/if}
        </div>

`;

content = content.slice(0, startIdx) + newBlock + content.slice(endIdx);
fs.writeFileSync('src/routes/dashboard/+layout.svelte', content);
console.log('✅ Done: header block simplified');
