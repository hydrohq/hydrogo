<script lang="ts">
  import '../../app.css';
  import Header from '$lib/components/Header.svelte';
  import Footer from '$lib/components/Footer.svelte';

  let { data, children } = $props();

  // app.html's lang="%lang%" placeholder only gets replaced in a per-request
  // server render; this is a static SPA, so set it client-side instead.
  $effect(() => {
    document.documentElement.lang = data.locale;
  });
</script>

<a class="skip" href="#content">Skip to content</a>
<Header siteName={data.siteName} navigation={data.navigation} locale={data.locale} locales={data.locales} />
<main id="content">{@render children()}</main>
<Footer siteName={data.siteName} text={data.footer} />
