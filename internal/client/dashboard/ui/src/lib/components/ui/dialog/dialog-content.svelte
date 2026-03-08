<script lang="ts">
	import { Dialog as DialogPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";
	import DialogOverlay from "./dialog-overlay.svelte";
	import type { ContentProps } from "./index.js";

	type $$Props = ContentProps;

	let className: $$Props["class"] = undefined;
	export { className as class };
</script>

<DialogPrimitive.Portal>
	<DialogOverlay />
	<DialogPrimitive.Content
		class={cn(
			"fixed left-1/2 top-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-lg border bg-background p-6 shadow-lg",
			className
		)}
		{...$$restProps}
		on:pointerdown
		on:pointerup
		on:pointermove
		on:touchend
		on:touchstart
		on:touchcancel
		on:touchmove
	>
		<slot />
	</DialogPrimitive.Content>
</DialogPrimitive.Portal>
