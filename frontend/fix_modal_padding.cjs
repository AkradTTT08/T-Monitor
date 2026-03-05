const fs = require('fs');
const files = [
    'd:/TTT-Monitor-API-n8n/T-Monitor/frontend/src/routes/dashboard/projects/[id]/+page.svelte',
    'd:/TTT-Monitor-API-n8n/T-Monitor/frontend/src/routes/dashboard/+page.svelte'
];

for (const file of files) {
    let content = fs.readFileSync(file, 'utf8');

    // Fix footer paddings/margins
    // -mb-6 and px-6 pb-6 etc. were extending the footer and making it tall. 
    // Let's replace the whole footer container class if it contains pb-6 mt-4.
    content = content.replace(/pt-4 flex justify-end gap-3 border-t border-([^ ]+) sticky bottom-0 bg-([^ ]+) z-10 -mx-6 px-6 -mb-6 pb-6 mt-4/g, 'pt-3 flex justify-end gap-3 border-t border-$1 sticky bottom-0 bg-$2 z-10 -mx-6 px-6 -mb-6 pb-4 mt-2 backdrop-blur-md');

    // Same for relative or absolute footers
    content = content.replace(/-mb-6 pb-6 mt-4/g, '-mb-6 pb-4 mt-2');
    content = content.replace(/pb-6 mt-4/g, 'pb-4 mt-2');

    // Create Folder/Delete Folder footers that used absolute / pt-4
    content = content.replace(/pt-4 border-t/g, 'pt-3 border-t');

    // Fix button sizes. Removing mixed text-sm and text-xs
    content = content.replace(/text-xs([^>]*?)\s?text-sm/g, 'text-xs$1');
    content = content.replace(/text-sm([^>]*?)\s?text-xs/g, 'text-sm$1');

    // Extra fixes: "px-5 py-2.5 text-xs text-sm" which was weird
    content = content.replace(/px-5 py-2\.5/g, 'px-4 py-2');

    fs.writeFileSync(file, content);
}
console.log('Fixed modal padding and button text sizes');
