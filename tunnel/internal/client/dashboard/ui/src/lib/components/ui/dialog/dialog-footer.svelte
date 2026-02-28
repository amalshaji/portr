<script lang="ts">
	import { cn } from "$lib/utils.js";

	type $$Props = {
		class?: string;
	};

	let className: $$Props["class"] = undefined;
	export { className as class };
</script>

<div
	class={cn(
		"mt-6 flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2",
		className
	)}
	{...$$restProps}
>
	<slot />
</div>
