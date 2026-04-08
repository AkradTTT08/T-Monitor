<script lang="ts">
	import { slide, fade } from 'svelte/transition';

	let isOpen = $state(false);
	let query = $state('');
	let isTyping = $state(false);
	let chatHistory = $state<{ role: 'user' | 'ai'; text: string }[]>([]);

	let chatContainerElement = $state<HTMLElement | null>(null);

	function toggleChat() {
		isOpen = !isOpen;
	}

	async function sendMessage() {
		if (!query.trim()) return;

		const userMessage = query;
		chatHistory.push({ role: 'user', text: userMessage });
		query = '';
		isTyping = true;
		
		// Auto scroll
		setTimeout(() => {
			if (chatContainerElement) {
				chatContainerElement.scrollTop = chatContainerElement.scrollHeight;
			}
		}, 50);

		try {
			const token = localStorage.getItem("monitor_token");
			const res = await fetch('http://localhost:8082/api/v1/ai/chat', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({ 
					query: userMessage,
					history: chatHistory.slice(-6) // Send last 6 messages for context
				})
			});

			const data = await res.json().catch(() => ({}));

			if (!res.ok) {
				throw new Error(data.error || data.message || 'Failed to fetch from AI');
			}

			chatHistory.push({ role: 'ai', text: data.answer || data.error });
		} catch (error: any) {
			chatHistory.push({ role: 'ai', text: `ขออภัย เกิดข้อผิดพลาด: ${error.message}` });
		} finally {
			isTyping = false;
			// Auto scroll
			setTimeout(() => {
				if (chatContainerElement) {
					chatContainerElement.scrollTop = chatContainerElement.scrollHeight;
				}
			}, 50);
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			sendMessage();
		}
	}
</script>

<div class="fixed bottom-6 right-6 z-50 flex flex-col items-end">
	<!-- Chat Window -->
	{#if isOpen}
		<div
			class="mb-4 flex h-[500px] w-[380px] flex-col overflow-hidden rounded-2xl bg-white shadow-2xl ring-1 ring-gray-900/5 dark:bg-gray-800 dark:ring-white/10"
			transition:slide={{ duration: 300, axis: 'y' }}
		>
			<!-- Header -->
			<div
				class="flex items-center justify-between bg-gradient-to-r from-blue-600 to-cyan-500 p-4 text-white"
			>
				<div class="flex items-center gap-2">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="24"
						height="24"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<path
							d="M12 8V4H8a2 2 0 0 0-2 2v4a2 2 0 0 0 2 2h4a2 2 0 0 0 2-2v-4a2 2 0 0 0-2-2h-4"
						/>
						<path
							d="M20 8v-4h-4a2 2 0 0 0-2 2v4a2 2 0 0 0 2 2h4a2 2 0 0 0 2-2v-4a2 2 0 0 0-2-2h-4"
						/>
						<path
							d="M12 20v-4h-4a2 2 0 0 0-2 2v4a2 2 0 0 0 2 2h4a2 2 0 0 0 2-2v-4a2 2 0 0 0-2-2h-4"
						/>
						<path
							d="M20 20v-4h-4a2 2 0 0 0-2 2v4a2 2 0 0 0 2 2h4a2 2 0 0 0 2-2v-4a2 2 0 0 0-2-2h-4"
						/>
					</svg>
					<div>
						<h3 class="font-bold leading-none">T-Monitor AI</h3>
						<span class="text-xs text-blue-100">Powered by Gemini</span>
					</div>
				</div>
				<button
					class="rounded-full p-1 text-white hover:bg-white/20 transition-colors"
					onclick={toggleChat}
					aria-label="Close Chat"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="20"
						height="20"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<path d="M18 6 6 18" />
						<path d="m6 6 12 12" />
					</svg>
				</button>
			</div>

			<!-- Chat History -->
			<div class="flex-1 overflow-y-auto p-4 space-y-4" bind:this={chatContainerElement}>
				{#if chatHistory.length === 0}
					<div class="flex h-full flex-col items-center justify-center text-center text-gray-500">
						<div class="mb-3 rounded-full bg-blue-100 p-3 text-blue-600 dark:bg-blue-900/30">
							<svg
								xmlns="http://www.w3.org/2000/svg"
								width="32"
								height="32"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<circle cx="12" cy="12" r="10" />
								<path d="M12 16v-4" />
								<path d="M12 8h.01" />
							</svg>
						</div>
						<p class="text-sm">สอบถามข้อมูลเกี่ยวกับระบบ API ของคุุณได้เลย!<br />เช่น "โปรเจกต์ไหนมี API ล่มเยอะสุด"</p>
					</div>
				{/if}

				{#each chatHistory as msg}
					<div
						class="flex w-full {msg.role === 'user' ? 'justify-end' : 'justify-start'}"
						transition:fade={{ duration: 150 }}
					>
						<div
							class="max-w-[85%] rounded-2xl px-4 py-2.5 text-sm whitespace-pre-wrap {msg.role ===
							'user'
								? 'bg-blue-600 text-white rounded-tr-sm'
								: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200 rounded-tl-sm'}"
						>
							{msg.text}
						</div>
					</div>
				{/each}

				{#if isTyping}
					<div class="flex w-full justify-start" transition:fade={{ duration: 150 }}>
						<div
							class="flex items-center gap-1 rounded-2xl rounded-tl-sm bg-gray-100 px-4 py-3 dark:bg-gray-700"
						>
							<div
								class="h-2 w-2 animate-bounce rounded-full bg-gray-400"
								style="animation-delay: 0ms"
							></div>
							<div
								class="h-2 w-2 animate-bounce rounded-full bg-gray-400"
								style="animation-delay: 150ms"
							></div>
							<div
								class="h-2 w-2 animate-bounce rounded-full bg-gray-400"
								style="animation-delay: 300ms"
							></div>
						</div>
					</div>
				{/if}
			</div>

			<!-- Input Area -->
			<div class="border-t border-gray-100 p-3 dark:border-gray-700">
				<div class="relative flex items-center">
					<textarea
						bind:value={query}
						onkeydown={handleKeydown}
						placeholder="ถามอะไรสักอย่างสิ..."
						class="max-h-32 min-h-[44px] w-full resize-none rounded-xl border border-gray-200 bg-gray-50 py-2.5 pl-4 pr-12 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:focus:border-blue-400"
						rows="1"
					></textarea>
					<button
						onclick={sendMessage}
						disabled={!query.trim() || isTyping}
						aria-label="Send Message"
						class="absolute right-2 flex h-8 w-8 items-center justify-center rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 transition-colors"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="16"
							height="16"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<path d="m22 2-7 20-4-9-9-4Z" />
							<path d="M22 2 11 13" />
						</svg>
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- FAB -->
	<button
		class="relative flex h-14 w-14 items-center justify-center rounded-full bg-gradient-to-tr from-blue-600 to-cyan-400 text-white shadow-[0_0_20px_rgba(34,211,238,0.4)] hover:scale-105 hover:shadow-[0_0_30px_rgba(34,211,238,0.6)] active:scale-95 transition-all duration-300"
		onclick={toggleChat}
		aria-label="Toggle AI Chat"
	>
		{#if isOpen}
			<svg
				xmlns="http://www.w3.org/2000/svg"
				width="24"
				height="24"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<path d="M18 6 6 18" />
				<path d="m6 6 12 12" />
			</svg>
		{:else}
			<svg 
				xmlns="http://www.w3.org/2000/svg" 
				viewBox="0 0 24 24" 
				width="28" 
				height="28" 
				fill="currentColor"
				class="animate-pulse duration-[3000ms]"
			>
  				<path d="M12 2C6.477 2 2 6.145 2 11.258c0 2.9 1.488 5.485 3.82 7.157v3.585l3.486-1.922c.86.239 1.764.368 2.694.368 5.523 0 10-4.145 10-9.258S17.523 2 12 2zm1.093 12.392-2.825-3.003-5.508 3.003 6.044-6.427 2.89 3.004 5.442-3.004-6.043 6.427z"/>
			</svg>
		{/if}
	</button>
</div>
