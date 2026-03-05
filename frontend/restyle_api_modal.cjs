const fs = require('fs');
const file = 'd:/TTT-Monitor-API-n8n/T-Monitor/frontend/src/routes/dashboard/apis/+page.svelte';
let content = fs.readFileSync(file, 'utf8');

// URL bar container
content = content.replace(
    /class="bg-slate-50 border border-slate-200 rounded-xl p-4 flex items-center justify-between"/g,
    'class="bg-slate-900/60 border border-slate-700/60 rounded-xl p-4 flex items-center justify-between shadow-[inset_0_0_30px_rgba(0,0,0,0.3)]"'
);

// Method badge colors - replace light versions with dark neon equivalents
content = content.replace(/'bg-green-100 text-green-700'/g, "'bg-green-950/60 text-green-400 border border-green-500/30'");
content = content.replace(/'bg-blue-100 text-blue-700'/g, "'bg-cyan-950/60 text-cyan-400 border border-cyan-500/30'");
content = content.replace(/'bg-yellow-100 text-yellow-700'/g, "'bg-amber-950/60 text-amber-400 border border-amber-500/30'");
content = content.replace(/'bg-red-100 text-red-700'/g, "'bg-red-950/60 text-red-400 border border-red-500/30'");
content = content.replace(/'bg-slate-200 text-slate-700'\}/g, "'bg-slate-800 text-slate-400 border border-slate-600'}");

// Copy URL button
content = content.replace(
    /class="opacity-0 group-hover\/copy:opacity-100 transition-opacity p-2 bg-white border border-slate-200 shadow-sm rounded-md text-slate-500 hover:text-slate-800 hover:bg-slate-50 cursor-pointer shrink-0"/g,
    'class="opacity-0 group-hover/copy:opacity-100 transition-opacity p-2 bg-slate-800 border border-slate-700 rounded-md text-slate-400 hover:text-cyan-400 hover:border-cyan-500/40 cursor-pointer shrink-0"'
);

// Panel borders for Headers, Params, Body
content = content.replace(
    /class="border border-slate-200 rounded-xl overflow-hidden flex flex-col h-48"/g,
    'class="border border-slate-700/60 rounded-xl overflow-hidden flex flex-col h-48 bg-slate-950/30"'
);

// Section label headers (bg-slate-50 border-b border-slate-200) for 2-arg version
content = content.replace(
    /class="bg-slate-50 border-b border-slate-200 px-3 py-2 flex justify-between items-center"/g,
    'class="bg-slate-800/70 border-b border-slate-700/60 px-3 py-2 flex justify-between items-center"'
);
// body header
content = content.replace(
    /class="bg-slate-50 border-b border-slate-200 px-3 py-2 flex items-center justify-between"/g,
    'class="bg-slate-800/70 border-b border-slate-700/60 px-3 py-2 flex items-center justify-between"'
);

// Section label text
content = content.replace(
    /class="text-xs font-bold text-slate-600 uppercase tracking-widest"\s*>Headers/g,
    'class="text-xs font-bold text-cyan-500/80 uppercase tracking-widest font-mono" >Headers'
);
content = content.replace(
    /class="text-xs font-bold text-slate-600 uppercase tracking-widest"\s*>Query Params/g,
    'class="text-xs font-bold text-amber-400/80 uppercase tracking-widest font-mono" >Query Params'
);
content = content.replace(
    /class="text-xs font-bold text-slate-600 uppercase tracking-widest"\s*>Request Body/g,
    'class="text-xs font-bold text-indigo-400/80 uppercase tracking-widest font-mono" >Request Body'
);

// Copy buttons inside panels
content = content.replace(
    /class="p-1 text-slate-400 hover:text-slate-700 transition-colors"/g,
    'class="p-1 text-slate-500 hover:text-cyan-400 transition-colors"'
);

// Raw JSON badge
content = content.replace(
    /class="text-\[10px\] bg-slate-200 text-slate-600 px-2 py-0\.5 rounded uppercase font-bold"/g,
    'class="text-[10px] bg-slate-700 text-indigo-300 border border-slate-600 px-2 py-0.5 rounded uppercase font-mono font-bold"'
);

// Close & Send Request buttons (in flex justify-between)
content = content.replace(
    /class="px-5 py-2\.5 text-slate-600 bg-white border border-slate-300 rounded-xl hover:bg-slate-50 font-bold transition-colors text-sm"\s*>Close/g,
    'class="px-4 py-2 text-slate-400 bg-slate-800 border border-slate-700 rounded-xl hover:bg-slate-700 hover:text-cyan-400 font-bold transition-colors text-xs" >Close'
);
content = content.replace(
    /class="px-6 py-2\.5 bg-blue-600 text-white rounded-xl hover:bg-blue-700 font-bold transition-all shadow-md text-sm flex items-center gap-2 outline-none focus:ring-4 focus:ring-blue-500\/30 disabled:opacity-75 relative overflow-hidden"/g,
    'class="px-4 py-2 bg-cyan-600 text-cyan-50 rounded-xl hover:bg-cyan-700 font-bold transition-all shadow-[0_0_15px_rgba(6,182,212,0.3)] text-xs flex items-center gap-2 outline-none focus:ring-4 focus:ring-cyan-500/30 disabled:opacity-75 relative overflow-hidden"'
);

// spinner text-white
content = content.replace(/class="animate-spin h-4 w-4 text-white"/g, 'class="animate-spin h-4 w-4 text-cyan-50"');

// Test Result section border-top
content = content.replace(
    /class="mt-8 border-t border-slate-200 pt-6 animate-fade-in"/g,
    'class="mt-8 border-t border-slate-700 pt-6 animate-fade-in"'
);

// RESPONSE label
content = content.replace(
    /class="text-sm font-black text-slate-800 tracking-wide"/g,
    'class="text-sm font-black text-slate-400 tracking-widest font-mono uppercase"'
);

// Status badges (green/amber/red) for test Result status
content = content.replace(/'bg-green-100 text-green-700 border border-green-200'/g, "'bg-green-950/50 text-green-400 border border-green-500/30'");
content = content.replace(/'bg-amber-100 text-amber-700 border border-amber-200'/g, "'bg-amber-950/50 text-amber-400 border border-amber-500/30'");
content = content.replace(/'bg-red-100 text-red-700 border border-red-200'/g, "'bg-red-950/50 text-red-400 border border-red-500/30'");
content = content.replace(/'bg-slate-100 text-slate-700'/g, "'bg-slate-800 text-slate-400'");

// Latency badge
content = content.replace(
    /class="px-2\.5 py-1 rounded text-\[11px\] font-black tracking-widest font-mono bg-blue-50 text-blue-700 border border-blue-200"/g,
    'class="px-2.5 py-1 rounded text-[11px] font-black tracking-widest font-mono bg-cyan-950/50 text-cyan-400 border border-cyan-500/30"'
);

// Failed to connect badge
content = content.replace(
    /class="px-2\.5 py-1 rounded text-\[11px\] font-black tracking-widest font-mono bg-red-100 text-red-700 border border-red-200"\s*>\s*FAILED TO CONNECT/g,
    'class="px-2.5 py-1 rounded text-[11px] font-black tracking-widest font-mono bg-red-950/50 text-red-400 border border-red-500/30" > FAILED TO CONNECT'
);

fs.writeFileSync(file, content);
console.log('Done - API Details & Testing modal updated to dark neon theme');
