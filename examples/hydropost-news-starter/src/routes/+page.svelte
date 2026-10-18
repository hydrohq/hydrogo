<script lang="ts">
  import type { PageProps } from './$types';
  import FeaturedCarousel from '#lib/components/FeaturedCarousel.svelte';
  import NewsGrid from '#lib/components/NewsGrid.svelte';
  import NewsletterSignup from '#lib/components/NewsletterSignup.svelte';
  import PopularReadList from '#lib/components/PopularReadList.svelte';
  import MediaGallerySection from '#lib/components/MediaGallerySection.svelte';
  import CategorySection from '#lib/components/CategorySection.svelte';

  let { data }: PageProps = $props();
</script>

<!-- Section order, visibility and inclusion are all driven by the
     "Anasayfa Bölümleri" (homepage_sections) content type in the admin
     panel — this file only knows how to render each known key. -->
{#each data.sections as section (section.id)}
  {#if section.key === 'hero'}
    <FeaturedCarousel items={data.featured} />
  {:else if section.key === 'son_haberler'}
    <section class="my-10">
      <div class="flex items-end justify-between mb-5">
        <h2 class="text-lg md:text-xl font-bold text-[var(--hp-ink)]">Son Haberler</h2>
        <a href="/gundem" class="text-xs font-bold text-[var(--hp-muted)] hover:text-[var(--hp-primary)] flex items-center gap-1.5 transition-colors">Tümünü Gör <span>→</span></a>
      </div>
      <NewsGrid items={data.latestGridA} columns={4} showCategory />
    </section>
  {:else if section.key === 'bulten'}
    <NewsletterSignup />
  {:else if section.key === 'ikinci_grid'}
    <section class="my-10">
      <NewsGrid
        items={data.latestGridB}
        extraCells={[
          { kind: 'ad', ad: data.homeGrid1Ad },
          { kind: 'poll', poll: data.poll },
          { kind: 'ad', ad: data.homeGrid2Ad },
        ]}
      />
    </section>
  {:else if section.key === 'cok_okunan'}
    <PopularReadList items={data.popularNews} />
  {:else if section.key === 'galeri'}
    <MediaGallerySection galleries={data.galleries} videos={data.videos} />
  {:else if section.key === 'kategori_vitrini' && section.category_filter}
    <CategorySection category={section.category_filter} items={data.categoryShowcases[section.category_filter] || []} />
  {/if}
{/each}
