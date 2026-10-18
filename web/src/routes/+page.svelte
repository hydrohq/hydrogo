<script lang="ts">
  import type { PageProps } from './$types';
  import BreakingNewsTicker from '#lib/components/BreakingNewsTicker.svelte';
  import FeaturedCarousel from '#lib/components/FeaturedCarousel.svelte';
  import SurManset from '#lib/components/SurManset.svelte';
  import CategoryGrid from '#lib/components/CategoryGrid.svelte';
  import ColumnistsSection from '#lib/components/ColumnistsSection.svelte';
  import MediaGallerySection from '#lib/components/MediaGallerySection.svelte';
  import AdSlot from '#lib/components/AdSlot.svelte';

  let { data }: PageProps = $props();
</script>

<!-- Breaking News Bar -->
<div class="-mx-3 sm:-mx-4 -mt-4 mb-4">
  <BreakingNewsTicker items={data.breakingNews} />
</div>

<!-- Header Top Banner Ad -->
<AdSlot ad={data.headerAd} zone="header_top" />

<!-- Main Headline Section -->
<div class="my-4">
  <FeaturedCarousel items={data.featured} />
</div>

<!-- 4-Card Sur Manset Showcase -->
<SurManset items={data.surManset} />

<!-- Under Headline Banner Ad -->
<AdSlot ad={data.mansetAltiAd} zone="manset_alti" />

<!-- Gündem Category Section -->
<CategoryGrid title="Gündem" categorySlug="gundem" items={data.gundemNews} />

<!-- Columnists Band -->
<ColumnistsSection columnists={data.columnists} />

<!-- Dünya Category Section -->
<CategoryGrid title="Dünya" categorySlug="dunya" items={data.dunyaNews} />

<!-- Rich Media (Photo & Video Galleries) -->
<MediaGallerySection galleries={data.galleries} videos={data.videos} />

<!-- Ekonomi Category Section -->
<CategoryGrid title="Ekonomi" categorySlug="ekonomi" items={data.ekonomiNews} />
