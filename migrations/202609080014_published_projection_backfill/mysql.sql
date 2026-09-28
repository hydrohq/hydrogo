UPDATE pages
SET published_revision = revision,
    published_locale = locale,
    published_slug = slug,
    published_title = title,
    published_body = body
WHERE status = 'published'
  AND published_revision IS NULL;
