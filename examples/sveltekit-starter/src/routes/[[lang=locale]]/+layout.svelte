<script lang="ts">
  import '../../app.css';
  import Header from '$lib/components/Header.svelte';
  import Footer from '$lib/components/Footer.svelte';

  let { data, children } = $props();
</script>

<a class="skip" href="#content">Skip to content</a>
<Header siteName={data.siteName} navigation={data.navigation} locale={data.locale} locales={data.locales} />
<main id="content">{@render children()}</main>
<Footer siteName={data.siteName} text={data.footer} />
