const fs = require('fs');
const file = 'd:/T-Monitor/frontend/src/routes/dashboard/apis/+page.svelte';
let content = fs.readFileSync(file, 'utf8');
content = content.replace('import { page } from "$app/stores"', 'import { page } from "$app/state"');
content = content.replace('$page.url', 'page.url');
fs.writeFileSync(file, content, 'utf8');
console.log('Fixed stores in apis/+page.svelte');
