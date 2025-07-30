<script lang="ts">
  import * as Tooltip from "$lib/components/ui/tooltip";
  import { Button } from "$lib/components/ui/button";
  import moment from "moment";

  export let Date: string;
</script>

<Tooltip.Root>
  <Tooltip.Trigger asChild let:builder>
    <Button
      builders={[builder]}
      variant="ghost"
      class="p-0 hover:bg-transparent"
    >
      {moment(Date).fromNow()}</Button
    >
  </Tooltip.Trigger>
  <Tooltip.Content>
    <p>{moment(Date).format("DD MMM YYYY, HH:mm:ss")}</p>
  </Tooltip.Content>
</Tooltip.Root>
