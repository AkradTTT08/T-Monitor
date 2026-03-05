const fs = require('fs');
const file = 'd:/TTT-Monitor-API-n8n/T-Monitor/frontend/src/routes/dashboard/projects/[id]/+page.svelte';
let content = fs.readFileSync(file, 'utf8');

const map = {
    'bg-blue-50': 'bg-cyan-950/30',
    'border-blue-200': 'border-cyan-500/30',
    'text-blue-700': 'text-cyan-400',
    'hover:bg-blue-100': 'hover:bg-cyan-900/50',
    'bg-red-50': 'bg-red-950/30',
    'border-red-200': 'border-red-500/30',
    'text-red-700': 'text-red-400',
    'hover:bg-red-100': 'hover:bg-red-900/50',
    'bg-slate-50': 'bg-slate-900/50',
    'border-slate-200': 'border-slate-700/50',
    'text-slate-700': 'text-cyan-50',
    'text-slate-800': 'text-cyan-300',
    'bg-slate-100': 'bg-slate-800',
    'text-slate-500': 'text-cyan-500/80',
    'bg-blue-600': 'bg-cyan-600',
    'hover:bg-blue-700': 'hover:bg-cyan-700',
    'text-slate-600': 'text-slate-400',
    'border-slate-100': 'border-slate-800',
    'text-blue-600': 'text-cyan-400',
    'hover:text-blue-700': 'hover:text-cyan-300',
    'text-slate-900': 'text-cyan-50',
    'bg-slate-200': 'bg-slate-700',
    'focus:ring-blue-500': 'focus:ring-cyan-500',
    'focus:border-blue-500': 'focus:border-cyan-500',
    'shadow-blue-500': 'shadow-cyan-500',
    'text-slate-400': 'text-slate-500',
    'bg-white': 'bg-slate-900/40',
    'border-slate-300': 'border-slate-600',
    'text-amber-700': 'text-amber-400',
    'bg-amber-100': 'bg-amber-900/30',
};

// Also let's fix the button sizes
const sizeMap = {
    'px-6 py-2.5': 'px-4 py-2',
    'px-5 py-2.5': 'px-4 py-2 text-xs',
    'text-sm disabled\\:opacity-50': 'text-xs disabled:opacity-50'
};

for (const [key, val] of Object.entries(map)) {
    const safeStr = key.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&');
    const regex = new RegExp(`(?<![a-zA-Z0-9-])` + safeStr + `(?![a-zA-Z0-9-])`, 'g');
    content = content.replace(regex, val);
}

for (const [key, val] of Object.entries(sizeMap)) {
    const regex = new RegExp(key, 'g');
    content = content.replace(regex, val);
}

fs.writeFileSync(file, content);
console.log('Updated +page.svelte globally');
