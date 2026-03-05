const fs = require('fs');
const file = 'd:/TTT-Monitor-API-n8n/T-Monitor/frontend/src/routes/dashboard/projects/[id]/+page.svelte';
let content = fs.readFileSync(file, 'utf8');

// The original was class="space-y-4" and I changed to class="space-y-4 pb-20".
// Revert that back entirely.
content = content.replace(/class="space-y-4 pb-20"/g, 'class="space-y-4"');

// And the space-y-4 pb-12, space-y-4 pb-16 just in case any remained.
content = content.replace(/class="space-y-4 pb-[0-9]+"/g, 'class="space-y-4"');

// Wait, the real problem is that the expected HTTP status code is covered by the footer. 
// A better way to avoid that is putting a spacer div before the footer.
// e.g. <div class="pb-16"></div> before <div class="pt-3 flex justify-end gap-3 border-t border-slate-800 ...">
content = content.replace(/(\s*)(<div[^>]*class="[^"]*sticky bottom-[0-9]+[^"]*"[^>]*>)/g, '$1<div class="pb-12"></div>$1$2');

fs.writeFileSync(file, content);
console.log('Fixed sticky footer overlap gap by moving padding before the footer instead of the bottom of the container');
