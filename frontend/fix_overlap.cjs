const fs = require('fs');
const file = 'd:/TTT-Monitor-API-n8n/T-Monitor/frontend/src/routes/dashboard/projects/[id]/+page.svelte';
let content = fs.readFileSync(file, 'utf8');

// The modal body is space-y-4 right now:
// <form on:submit|preventDefault={handleAddApiSubmit} class="space-y-4">
// We need it to be space-y-4 pb-12 so there's extra room for the sticky footer.
content = content.replace(/class="space-y-4"/g, 'class="space-y-4 pb-12"');

// And just to be extremely safe, we ensure that Add/Edit API Modals have extra bottom space.
// If pb-12 is not enough, pb-16 will ensure expected status code is completely visible.
content = content.replace(/class="space-y-4 pb-12"/g, 'class="space-y-4 pb-16"');

// Wait, the inner div for expected status code has no particular padding.
// The true issue is that sticky footers need space at the bottom of the scrolling container.
// It's already in a form with space-y-4. Let's add pb-20 to the form.
content = content.replace(/class="space-y-4 pb-16"/g, 'class="space-y-4 pb-20"');

fs.writeFileSync(file, content);
console.log('Added pb-20 to forms so footer does not overlap content');
