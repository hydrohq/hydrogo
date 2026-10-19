<script lang="ts">
  let { siteName, text }: { siteName: string; text: string } = $props();
</script>

<footer class="site-footer">
  <p>{text || `© ${new Date().getFullYear()} ${siteName}`}</p>
  <p><a href="/rss.xml">RSS</a> · <a href="/sitemap.xml">Sitemap</a></p>
</footer>
