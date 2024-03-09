<script lang="ts">
  // @ts-expect-error
  import { Render, Subscribe } from "svelte-headless-table";
  import * as Table from "$lib/components/ui/table";
  import DataTableSkeleton from "./data-table-skeleton.svelte";

  export let table, columns, isLoading: boolean;

  const { tableAttrs, tableBodyAttrs, pageRows, headerRows } =
    table.createViewModel(columns);
</script>

<div class="rounded-sm border">
  <Table.Root {...$tableAttrs}>
    <Table.Header>
      {#each $headerRows as headerRow}
        <Subscribe rowAttrs={headerRow.attrs()}>
          <Table.Row>
            {#each headerRow.cells as cell (cell.id)}
              <Subscribe
                attrs={cell.attrs()}
                let:attrs
                props={cell.props()}
                let:props
              >
                <Table.Head {...attrs}>
                  <div class="px-2">
                    <Render of={cell.render()} />
                  </div>
                </Table.Head>
              </Subscribe>
            {/each}
          </Table.Row>
        </Subscribe>
      {/each}
    </Table.Header>

    <Table.Body {...$tableBodyAttrs}>
      {#if isLoading}
        <Table.Row>
          <Table.Cell colspan={$headerRows[0].cells.length}>
            <DataTableSkeleton />
          </Table.Cell>
        </Table.Row>
      {:else if $pageRows.length === 0}
        <Table.Row>
          <Table.Cell colspan={$headerRows[0].cells.length}>
            <div class="flex flex-col items-center justify-center py-10">
              <p class="text-gray-500">No data available</p>
            </div>
          </Table.Cell>
        </Table.Row>
      {:else}
        {#each $pageRows as row (row.id)}
          <Subscribe rowAttrs={row.attrs()}>
            <Table.Row>
              {#each row.cells as cell (cell.id)}
                <Subscribe
                  attrs={cell.attrs()}
                  let:attrs
                  props={cell.props()}
                  let:props
                >
                  <Table.Cell {...attrs}>
                    <div class="px-2 py-2">
                      <Render of={cell.render()} />
                    </div>
                  </Table.Cell>
                </Subscribe>
              {/each}
            </Table.Row>
          </Subscribe>
        {/each}
      {/if}
    </Table.Body>
  </Table.Root>
</div>
