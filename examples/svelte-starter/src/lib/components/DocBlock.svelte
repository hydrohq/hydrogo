<script lang="ts">
  import type { Block } from '../docs/types';
  import { docsHref } from '../router.svelte';
  import Inline from './Inline.svelte';
  import CodeBlock from './CodeBlock.svelte';
  import CodeTabs from './CodeTabs.svelte';
  import Callout from './Callout.svelte';
  import Diagram from './Diagram.svelte';

  let { block, section, page }: { block: Block; section: string; page: string } = $props();
</script>

{#if block.type === 'p'}
  <p class="my-4 leading-7 text-muted [&_strong]:text-fg"><Inline text={block.text} /></p>
{:else if block.type === 'h2'}
  <h2 id={block.id} class="doc-h group mt-12 mb-3 border-t border-line pt-8 text-2xl font-semibold tracking-tight text-fg first:mt-0 first:border-0 first:pt-0">
    <a href={docsHref(section, page, block.id)} class="no-underline">{block.text}<span class="ml-2 text-accent opacity-0 group-hover:opacity-60 focus:opacity-100" aria-hidden="true">#</span><span class="sr-only"> (link to this section)</span></a>
  </h2>
{:else if block.type === 'h3'}
  <h3 id={block.id} class="doc-h group mt-8 mb-2 text-lg font-semibold text-fg">
    <a href={docsHref(section, page, block.id)} class="no-underline">{block.text}<span class="ml-2 text-accent opacity-0 group-hover:opacity-60 focus:opacity-100" aria-hidden="true">#</span><span class="sr-only"> (link to this section)</span></a>
  </h3>
{:else if block.type === 'code'}
  <CodeBlock code={block.code} lang={block.lang} title={block.title} />
{:else if block.type === 'tabs'}
  <CodeTabs tabs={block.tabs} />
{:else if block.type === 'callout'}
  <Callout kind={block.kind} title={block.title} text={block.text} />
{:else if block.type === 'list'}
  {#if block.ordered}
    <ol class="my-4 list-decimal space-y-2 pl-6 leading-7 text-muted marker:text-faint">
      {#each block.items as item}<li><Inline text={item} /></li>{/each}
    </ol>
  {:else}
    <ul class="my-4 list-disc space-y-2 pl-6 leading-7 text-muted marker:text-faint">
      {#each block.items as item}<li><Inline text={item} /></li>{/each}
    </ul>
  {/if}
{:else if block.type === 'table'}
  <div class="thin-scroll my-5 overflow-x-auto rounded-lg border border-line">
    <table class="w-full border-collapse text-left text-sm">
      <thead class="bg-subtle text-xs uppercase tracking-wide text-faint">
        <tr>
          {#each block.head as h}<th scope="col" class="whitespace-nowrap border-b border-line px-3 py-2 font-semibold">{h}</th>{/each}
        </tr>
      </thead>
      <tbody>
        {#each block.rows as row}
          <tr class="border-b border-line last:border-0 even:bg-subtle/50">
            {#each row as cell, i}
              <td class="px-3 py-2 align-top text-muted {i === 0 ? 'font-medium text-fg' : ''}"><Inline text={cell} /></td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{:else if block.type === 'cards'}
  <div class="my-5 grid gap-3 sm:grid-cols-2">
    {#each block.items as c}
      <a href={c.href} class="rounded-lg border border-line bg-elevated p-4 no-underline transition-colors hover:border-accent">
        <span class="block font-semibold text-fg">{c.title}</span>
        <span class="mt-1 block text-sm leading-6 text-muted">{c.text}</span>
      </a>
    {/each}
  </div>
{:else if block.type === 'diagram'}
  <Diagram name={block.name} />
{/if}
