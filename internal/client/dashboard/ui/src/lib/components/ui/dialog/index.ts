import { Dialog as DialogPrimitive } from "bits-ui";

import Content from "./dialog-content.svelte";
import Description from "./dialog-description.svelte";
import Footer from "./dialog-footer.svelte";
import Header from "./dialog-header.svelte";
import Overlay from "./dialog-overlay.svelte";
import Title from "./dialog-title.svelte";

const Root = DialogPrimitive.Root;
const Trigger = DialogPrimitive.Trigger;
const Close = DialogPrimitive.Close;
const Portal = DialogPrimitive.Portal;

type RootProps = DialogPrimitive.Props;
type TriggerProps = DialogPrimitive.TriggerProps;
type CloseProps = DialogPrimitive.CloseProps;
type PortalProps = DialogPrimitive.PortalProps;
type ContentProps = DialogPrimitive.ContentProps;
type OverlayProps = DialogPrimitive.OverlayProps;
type TitleProps = DialogPrimitive.TitleProps;
type DescriptionProps = DialogPrimitive.DescriptionProps;

export {
	Root,
	Trigger,
	Close,
	Portal,
	Content,
	Overlay,
	Header,
	Footer,
	Title,
	Description,
	type RootProps,
	type TriggerProps,
	type CloseProps,
	type PortalProps,
	type ContentProps,
	type OverlayProps,
	type TitleProps,
	type DescriptionProps,
};
