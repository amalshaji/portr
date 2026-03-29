/// <reference types="vite/client" />

declare module "vite/modulepreload-polyfill"

declare module "react-syntax-highlighter/dist/esm/languages/hljs/json" {
  const language: unknown
  export default language
}

declare module "react-syntax-highlighter/dist/esm/styles/hljs" {
  export const atomOneDark: Record<string, React.CSSProperties>
  export const atomOneLight: Record<string, React.CSSProperties>
}
