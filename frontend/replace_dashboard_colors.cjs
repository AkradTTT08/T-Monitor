const fs = require('fs');
const file = 'd:/TTT-Monitor-API-n8n/T-Monitor/frontend/src/routes/dashboard/+page.svelte';
let content = fs.readFileSync(file, 'utf8');

const splitString = '<!-- Create Project Modal -->';
let parts = content.split(splitString);
if (parts.length < 2) {
    console.log('Split string not found!');
    process.exit(1);
}

let top = parts[0];
let bottom = parts[1];

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
    'bg-white': 'bg-slate-800/80',
    'border-slate-100': 'border-slate-800',
    'text-blue-600': 'text-cyan-400',
    'hover:text-blue-700': 'hover:text-cyan-300',
    'border-slate-300': 'border-slate-700',
    'text-slate-900': 'text-cyan-50',
    'bg-slate-200': 'bg-slate-700',
    'focus:ring-blue-500': 'focus:ring-cyan-500',
    'focus:border-blue-500': 'focus:border-cyan-500',
    'shadow-blue-500': 'shadow-cyan-500',
    'text-white': 'text-cyan-50',
    'bg-red-600': 'bg-red-500/20 text-red-400 border border-red-500/30',
    'hover:bg-red-700': 'hover:bg-red-500/30 hover:text-red-300'
};

for (const [key, val] of Object.entries(map)) {
    const regex = new RegExp(`(?<=\\s|"|'|\`)${key.replace(/:/g, '\\\\:').replace(/\//g, '\\\\/')}(?=\\s|"|'|\`)`, 'g');
    bottom = bottom.replace(regex, val);
}

fs.writeFileSync(file, top + splitString + bottom);
console.log('updated dashboard/+page.svelte modals');
