<script>
	import '../app.css';
	import AIChat from '$lib/components/AIChat.svelte';
</script>

<div class="min-h-screen flex flex-col items-center justify-center bg-slate-900 relative overflow-hidden text-slate-100">
    <!-- Dark background with subtle gradient -->
    <div class="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-slate-800 via-slate-900 to-black"></div>

    <!-- Glowing orbs representing active API nodes -->
    <div class="absolute top-[10%] left-[10%] w-[500px] h-[500px] bg-cyan-600/20 rounded-full mix-blend-screen filter blur-[100px] animate-pulse"></div>
    <div class="absolute bottom-[10%] right-[10%] w-[600px] h-[600px] bg-blue-600/20 rounded-full mix-blend-screen filter blur-[120px] animate-pulse" style="animation-delay: 2s;"></div>
    
    <!-- Heartbeat Wave Animation with SCSS -->
    <div class="heartbeat-container">
        <!-- The Floating APIs Text Riding the Wave -->
        <div class="floating-text-container">
            <span class="apis-text">/APIs_</span>
        </div>
        
        <!-- SVG Heartbeat Line (Period width = 500. So translating -50% is perfectly seamless) -->
        <svg class="heartbeat-wave" preserveAspectRatio="none" viewBox="0 0 1000 100" fill="none" stroke="currentColor" xmlns="http://www.w3.org/2000/svg">
            <path class="wave-path" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
            d="M 0 50 L 300 50 L 320 20 L 350 90 L 380 10 L 410 70 L 430 50 
               L 800 50 L 820 20 L 850 90 L 880 10 L 910 70 L 930 50 
               L 1000 50" />
        </svg>

        <!-- Secondary, smaller wave (Period width = 500) -->
        <svg class="heartbeat-wave secondary-wave" preserveAspectRatio="none" viewBox="0 0 1000 100" fill="none" stroke="currentColor" xmlns="http://www.w3.org/2000/svg">
            <path class="wave-path-secondary" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"
            d="M 0 50 L 150 50 L 160 30 L 175 80 L 190 20 L 205 60 L 215 50 
               L 650 50 L 660 30 L 675 80 L 690 20 L 705 60 L 715 50 
               L 1000 50" />
        </svg>
    </div>

	<slot />
	<!-- Global AI Chat Widget -->
	<AIChat />
</div>

<style lang="scss">
    .heartbeat-container {
        position: absolute;
        inset: 0;
        top: 50%;
        margin-top: -5rem;
        height: 10rem;
        overflow: visible;
        pointer-events: none;
        opacity: 0.6;
    }

    .heartbeat-wave {
        position: absolute;
        height: 100%;
        width: 200vw; /* Must be 200% of the viewport width */
        left: 0;
        animation: scssWave 6s linear infinite;
    }

    .secondary-wave {
        top: 2rem;
        height: 50%;
        animation: scssWave 4s linear infinite;
        opacity: 0.5;
    }

    .wave-path {
        color: #22d3ee; // cyan-400
        filter: drop-shadow(0 0 10px rgba(34,211,238,0.8));
    }

    .wave-path-secondary {
        color: #3b82f6; // blue-500
        filter: drop-shadow(0 0 5px rgba(59,130,246,0.6));
    }

    .floating-text-container {
        position: absolute;
        bottom: 4rem; /* Sit right above the central wave line */
        width: 100vw; /* Center within the screen, not the extended wave */
        display: flex;
        justify-content: center;
        align-items: flex-end;
    }

    .apis-text {
        font-size: 15vw; /* Responsive huge text */
        font-weight: 900;
        color: rgba(6, 182, 212, 0.04); /* Subtly tinted text */
        font-family: monospace;
        letter-spacing: -0.05em;
        white-space: nowrap;
        animation: float 4s ease-in-out infinite alternate;
        text-shadow: 0 0 40px rgba(6, 182, 212, 0.05); /* Soft glow behind text */
    }

    @keyframes scssWave {
        0% { transform: translateX(0); }
        100% { transform: translateX(-50%); } 
    }

    @keyframes float {
        0% { transform: translateY(0px) rotate(-1deg) scale(1); }
        100% { transform: translateY(-20px) rotate(1deg) scale(1.02); }
    }
</style>
