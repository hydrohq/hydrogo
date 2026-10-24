<?php
/**
 * Server-side Meta Tag & OpenGraph Renderer for Social Media Crawlers & Search Bots.
 * Serves complete og:*, twitter:*, canonical and NewsArticle JSON-LD tags to crawlers
 * that do not execute JavaScript (WhatsApp, Facebook, X/Twitter, LinkedIn, Telegram, etc.).
 *
 * Reads through HydroGo's public JSON API -- never the SQLite file directly.
 * The database schema is an internal implementation detail; the API is the
 * only stable contract, and this script should keep working across a
 * HydroGo upgrade the same way any other frontend would.
 */

$category = isset($_GET['category']) ? trim($_GET['category']) : '';
$slug = isset($_GET['slug']) ? trim($_GET['slug']) : '';
$locale = isset($_GET['locale']) ? trim($_GET['locale']) : 'tr';

$siteName = 'HydroPost';
$siteDesc = 'HydroGo tekil binary mimarisiyle çalışan, ışık hızında haber portalı.';
// Set these for the actual deployment; they are not committed per-site values.
$publicOrigin = getenv('HYDROPOST_PUBLIC_ORIGIN') ?: 'http://127.0.0.1:8090';
// The API origin HydroGo itself answers on -- usually the same host (hydro.cgi
// serves /api/ on this same domain), but kept separate in case a deployment
// ever points bot-meta.php at a different HydroGo instance than the one the
// public site's own domain resolves to.
$apiOrigin = getenv('HYDROPOST_API_ORIGIN') ?: $publicOrigin;
$article = null;

/**
 * Minimal, dependency-free JSON GET against HydroGo's public API.
 * Returns the decoded `data` field, or null on any failure -- a slow or
 * down backend must never turn into a PHP fatal error for a crawler.
 */
function hydro_api_get(string $origin, string $path): ?array {
    $ctx = stream_context_create([
        'http' => ['method' => 'GET', 'timeout' => 3, 'ignore_errors' => true],
    ]);
    $raw = @file_get_contents(rtrim($origin, '/') . $path, false, $ctx);
    if ($raw === false) {
        return null;
    }
    $decoded = json_decode($raw, true);
    if (!is_array($decoded) || !array_key_exists('data', $decoded)) {
        return null;
    }
    return $decoded['data'];
}

// 1. Site settings (name, default SEO description) -- best-effort, falls
// back to the defaults above if the content type isn't seeded yet.
$settingsList = hydro_api_get($apiOrigin, '/api/v1/content/site_settings?limit=1');
if (is_array($settingsList) && count($settingsList) > 0) {
    $settings = $settingsList[0];
    if (!empty($settings['site_name'])) $siteName = $settings['site_name'];
    if (!empty($settings['seo_description'])) $siteDesc = $settings['seo_description'];
}

// 2. The article itself, by locale + slug.
if ($slug !== '') {
    $data = hydro_api_get($apiOrigin, '/api/v1/news/' . rawurlencode($locale) . '/' . rawurlencode($slug));
    if (is_array($data) && !empty($data['status']) && $data['status'] === 'published') {
        $article = $data;
    }
}

if ($article) {
    $title = !empty($article['seo_title']) ? $article['seo_title'] : $article['title'];
    $rawDesc = !empty($article['seo_description']) ? $article['seo_description'] : (!empty($article['excerpt']) ? $article['excerpt'] : $article['title']);
    $desc = mb_substr(trim(strip_tags($rawDesc)), 0, 200, 'UTF-8');
    $pageTitle = (str_contains($title, $siteName)) ? $title : ($title . ' — ' . $siteName);
    $articleCat = !empty($article['category']) ? strtolower($article['category']) : ($category ?: 'haber');
    $canonicalUrl = $publicOrigin . '/' . rawurlencode($articleCat) . '/' . rawurlencode($slug);

    $cover = $article['cover_media_id'] ?? '';
    if ($cover !== '' && !str_starts_with($cover, 'demo-')) {
        // cover_media_id is sometimes a full external image URL (an editor
        // pasting a source's own image link) rather than an internal
        // HydroGo media asset id -- only build a /media/ URL for the
        // latter, never rewrite an already-absolute URL into a broken one.
        $imageUrl = (str_starts_with($cover, 'http://') || str_starts_with($cover, 'https://'))
            ? $cover
            : $publicOrigin . '/media/' . rawurlencode($cover);
    } else {
        $imageUrl = $publicOrigin . '/images/editorial/gundem-1.jpg';
    }

    $author = !empty($article['author_name']) ? $article['author_name'] : $siteName;
    $publishedTime = !empty($article['created_at']) ? $article['created_at'] : date('c');
    $modifiedTime = !empty($article['updated_at']) ? $article['updated_at'] : $publishedTime;

    $jsonLd = [
        '@context' => 'https://schema.org',
        '@type' => 'NewsArticle',
        'mainEntityOfPage' => [
            '@type' => 'WebPage',
            '@id' => $canonicalUrl,
        ],
        'headline' => $title,
        'description' => $desc,
        'image' => [$imageUrl],
        'datePublished' => $publishedTime,
        'dateModified' => $modifiedTime,
        'author' => [
            '@type' => 'Person',
            'name' => $author,
        ],
        'publisher' => [
            '@type' => 'Organization',
            'name' => $siteName,
            'logo' => [
                '@type' => 'ImageObject',
                'url' => $publicOrigin . '/favicon.svg',
            ],
        ],
    ];
} else {
    $pageTitle = $siteName . ' — Güncel Haberler';
    $desc = $siteDesc;
    $canonicalUrl = $publicOrigin . ($category ? '/' . rawurlencode($category) : '');
    $imageUrl = $publicOrigin . '/images/editorial/gundem-1.jpg';
    $jsonLd = null;
}

header('Content-Type: text/html; charset=utf-8');
?>
<!doctype html>
<html lang="tr">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title><?= htmlspecialchars($pageTitle, ENT_QUOTES, 'UTF-8') ?></title>
    <meta name="description" content="<?= htmlspecialchars($desc, ENT_QUOTES, 'UTF-8') ?>">
    <link rel="canonical" href="<?= htmlspecialchars($canonicalUrl, ENT_QUOTES, 'UTF-8') ?>">

    <!-- OpenGraph (Facebook, WhatsApp, LinkedIn) -->
    <meta property="og:site_name" content="<?= htmlspecialchars($siteName, ENT_QUOTES, 'UTF-8') ?>">
    <meta property="og:type" content="<?= $article ? 'article' : 'website' ?>">
    <meta property="og:title" content="<?= htmlspecialchars($pageTitle, ENT_QUOTES, 'UTF-8') ?>">
    <meta property="og:description" content="<?= htmlspecialchars($desc, ENT_QUOTES, 'UTF-8') ?>">
    <meta property="og:url" content="<?= htmlspecialchars($canonicalUrl, ENT_QUOTES, 'UTF-8') ?>">
    <meta property="og:image" content="<?= htmlspecialchars($imageUrl, ENT_QUOTES, 'UTF-8') ?>">
    <meta property="og:image:width" content="1200">
    <meta property="og:image:height" content="630">
    <meta property="og:locale" content="tr_TR">

    <!-- Twitter Card (X) -->
    <meta name="twitter:card" content="summary_large_image">
    <meta name="twitter:title" content="<?= htmlspecialchars($pageTitle, ENT_QUOTES, 'UTF-8') ?>">
    <meta name="twitter:description" content="<?= htmlspecialchars($desc, ENT_QUOTES, 'UTF-8') ?>">
    <meta name="twitter:image" content="<?= htmlspecialchars($imageUrl, ENT_QUOTES, 'UTF-8') ?>">

    <?php if ($jsonLd): ?>
    <script type="application/ld+json">
    <?= json_encode($jsonLd, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) ?>
    </script>
    <?php endif; ?>

    <!-- Fallback client redirect if accessed directly in a regular browser -->
    <noscript>
        <meta http-equiv="refresh" content="0; url=<?= htmlspecialchars($canonicalUrl, ENT_QUOTES, 'UTF-8') ?>">
    </noscript>
    <script>
        window.location.replace("<?= htmlspecialchars($canonicalUrl, ENT_QUOTES, 'UTF-8') ?>");
    </script>
</head>
<body style="font-family: system-ui, sans-serif; padding: 40px; text-align: center; color: #333;">
    <h1><?= htmlspecialchars($pageTitle, ENT_QUOTES, 'UTF-8') ?></h1>
    <p><?= htmlspecialchars($desc, ENT_QUOTES, 'UTF-8') ?></p>
    <p><a href="<?= htmlspecialchars($canonicalUrl, ENT_QUOTES, 'UTF-8') ?>">Habere Git &rarr;</a></p>
</body>
</html>
