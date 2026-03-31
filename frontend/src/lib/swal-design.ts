import Swal from 'sweetalert2';

export const systemAlert = Swal.mixin({
  background: '#0f172a', // Slate-900
  color: '#cbd5e1', // Slate-300
  confirmButtonColor: '#0891b2', // Cyan-600
  cancelButtonColor: '#334155', // Slate-700
  customClass: {
    popup: 'bg-slate-900 border border-slate-700/50 shadow-[0_0_50px_-12px_rgba(6,182,212,0.5)] rounded-2xl backdrop-blur-xl',
    title: 'text-xl font-bold font-mono tracking-tighter text-white uppercase pt-6',
    htmlContainer: 'text-sm font-medium text-slate-400 py-4',
    confirmButton: 'px-8 py-3 bg-cyan-600 hover:bg-cyan-500 text-white font-bold rounded-xl transition-all uppercase tracking-widest text-[10px] m-1',
    cancelButton: 'px-8 py-3 bg-slate-800 hover:bg-slate-700 text-slate-400 font-bold rounded-xl transition-all uppercase tracking-widest text-[10px] border border-slate-700 m-1',
    actions: 'pb-6 px-6',
    icon: 'border-cyan-500/30'
  },
  buttonsStyling: false,
});

export const systemToast = systemAlert.mixin({
  toast: true,
  position: 'top-end',
  showConfirmButton: false,
  timer: 3000,
  timerProgressBar: true,
  customClass: {
    popup: 'bg-slate-900/90 border border-slate-700/50 shadow-2xl rounded-2xl backdrop-blur-lg',
    title: 'text-xs font-bold font-mono tracking-tight text-white uppercase',
    htmlContainer: 'text-[10px] font-medium text-slate-400',
  }
});
