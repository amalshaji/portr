<script lang="ts">
	import { Checkbox as CheckboxPrimitive } from "bits-ui";
	import { Check, Minus } from "radix-icons-svelte";
	import { cn } from "$lib/utils";

	type $$Props = CheckboxPrimitive.Props;
	type $$Events = CheckboxPrimitive.Events;

	let className: $$Props["class"] = undefined;
	export let checked: $$Props["checked"] = false;
	export { className as class };
</script>

<CheckboxPrimitive.Root
	class={cn(
		"box-content peer h-4 w-4 shrink-0 rounded-sm-sm border border-primary shadow focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground data-[disabled=true]:cursor-not-allowed data-[disabled=true]:opacity-50",
		className
	)}
	bind:checked
	on:click
	{...$$restProps}
>
	<CheckboxPrimitive.Indicator
		class={cn("flex items-center justify-center text-current h-4 w-4")}
		let:isChecked
		let:isIndeterminate
	>
		{#if isIndeterminate}
			<Minus class="h-3.5 w-3.5" />
		{:else}
			<Check class={cn("h-3.5 w-3.5", !isChecked && "text-transparent")} />
		{/if}
	</CheckboxPrimitive.Indicator>
</CheckboxPrimitive.Root>
