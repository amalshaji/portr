import Root from "./checkbox.svelte";
import type { Checkbox as CheckboxPrimitive } from "bits-ui";

type Props = CheckboxPrimitive.Props;
type Events = CheckboxPrimitive.Events;

export {
	Root,
	type Props,
	type Events,
	//
	Root as Checkbox,
	type Props as CheckboxProps,
	type Events as CheckboxEvents,
};
