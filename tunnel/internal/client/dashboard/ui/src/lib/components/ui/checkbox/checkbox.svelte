<script lang="ts">
	import { Checkbox as CheckboxPrimitive } from "bits-ui";
	import { Check } from "lucide-svelte";
	import { cn } from "$lib/utils.js";
	import type { Props, Events } from "./index.js";

	type $$Props = Props;
	type $$Events = Events;

	let className: $$Props["class"] = undefined;
	export let checked: $$Props["checked"] = false;
	export let disabled: $$Props["disabled"] = undefined;
	export { className as class };
</script>

<CheckboxPrimitive.Root
	{checked}
	{disabled}
	class={cn(
		"peer h-4 w-4 shrink-0 rounded-sm border border-input ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground",
		className
	)}
	{...$$restProps}
	on:click
	on:keydown
>
	<CheckboxPrimitive.Indicator class="data-[state=unchecked]:hidden flex items-center justify-center text-current">
		<Check class="h-3.5 w-3.5" />
	</CheckboxPrimitive.Indicator>
</CheckboxPrimitive.Root>
