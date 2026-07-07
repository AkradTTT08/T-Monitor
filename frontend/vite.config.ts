import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			// Suppress a11y warnings that are treated as errors in vite-plugin-svelte v6+
			onwarn(warning, defaultHandler) {
				// Skip all a11y accessibility warnings
				if (warning.code?.startsWith('a11y_')) return;
				// Skip unused CSS selector warnings
				if (warning.code === 'css_unused_selector') return;
				defaultHandler(warning);
			}
		})
	]
});
