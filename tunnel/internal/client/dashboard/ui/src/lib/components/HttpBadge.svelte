<script lang="ts">
  export let method:
    | "GET"
    | "POST"
    | "PUT"
    | "DELETE"
    | "PATCH"
    | "OPTIONS"
    | "HEAD"
    | "CONNECT"
    | "TRACE";

  const getColorByMethod = (method: string) => {
    switch (method) {
      case "GET":
        return "bg-green-500";
      case "POST":
        return "bg-blue-500";
      case "PUT":
        return "bg-yellow-500";
      case "DELETE":
        return "bg-red-500";
      case "PATCH":
        return "bg-purple-500";
      case "OPTIONS":
        return "bg-gray-500";
      case "HEAD":
        return "bg-gray-500";
      case "CONNECT":
        return "bg-gray-500";
      case "TRACE":
        return "bg-gray-500";
      default:
        return "bg-gray-500";
    }
  };
</script>

<span
  class={`rounded-sm px-2 py-1 text-white text-xs font-light ${getColorByMethod(method)}`}
>
  {method}
</span>
