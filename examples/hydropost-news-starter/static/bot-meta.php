<?php
/**
 * Server-side Meta Tag & OpenGraph Renderer for Social Media Crawlers & Search Bots.
 * Serves complete og:*, twitter:*, canonical and NewsArticle JSON-LD tags to crawlers
 * that do not execute JavaScript (WhatsApp, Facebook, X/Twitter, LinkedIn, Telegram, etc.).
 */

$category = isset($_GET['category']) ? trim($_GET['category']) : '';
$slug = isset($_GET['slug']) ? trim($_GET['slug']) : '';

$siteName = 'HydroPost';
$siteDesc = 'HydroGo tekil binary mimarisiyle çalışan, ışık hızında haber portalı.';
// Set these for the actual deployment; they are not committed per-site values.
$publicOrigin = getenv('HYDROPOST_PUBLIC_ORIGIN') ?: 'http://127.0.0.1:8090';
$dbPath = getenv('HYDROPOST_DB_PATH') ?: __DIR__ . '/../hydro.db';
$article = null;

if (file_exists($dbPath)) {
    try {
        $pdo = new PDO('sqlite:' . $dbPath, null, null, [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_SILENT,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
        ]);

        // 1. Fetch site settings if present
        $st = $pdo->prepare("SELECT data_json FROM hydro_collection_records WHERE collection_id = 'site_settings' LIMIT 1");
        if ($st && $st->execute()) {
            $row = $st->fetch();
            if ($row && !empty($row['data_json'])) {
                $settings = json_decode($row['data_json'], true);
                if (!empty($settings['site_name'])) $siteName = $settings['site_name'];
                if (!empty($settings['seo_description'])) $siteDesc = $settings['seo_description'];
            }
        }

        // 2. Fetch published article by slug
        if ($slug !== '') {
            $stmt = $pdo->prepare("SELECT title, excerpt, body, cover_media_id, category, author_name, created_at, updated_at, seo_title, seo_description FROM news_items WHERE slug = :slug AND status = 'published' LIMIT 1");
            if ($stmt && $stmt->execute([':slug' => $slug])) {
                $article = $stmt->fetch();
            }
        }
    } catch (Exception $e) {
        // Fallback gracefully on DB error
    }
}

if ($article) {
    $title = !empty($article['seo_title']) ? $article['seo_title'] : $article['title'];
    $rawDesc = !empty($article['seo_description']) ? $article['seo_description'] : (!empty($article['excerpt']) ? $article['excerpt'] : $article['title']);
    $desc = mb_substr(trim(strip_tags($rawDesc)), 0, 200, 'UTF-8');
    $pageTitle = (str_contains($title, $siteName)) ? $title : ($title . ' — ' . $siteName);
    $articleCat = !empty($article['category']) ? strtolower($article['category']) : ($category ?: 'haber');
    $canonicalUrl = $publicOrigin . '/' . rawurlencode($articleCat) . '/' . rawurlencode($slug);

    if (!empty($article['cover_media_id']) && !str_starts_with($article['cover_media_id'], 'demo-')) {
        $imageUrl = $publicOrigin . '/media/' . rawurlencode($article['cover_media_id']);
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
