// Seeds the navigation menu and site settings content types so the header,
// footer and site copy are editable from the HydroGo admin panel instead of
// being hardcoded in the Svelte components.
//
// Usage: node scripts/seed-site-content.mjs

const BASE = process.env.HYDRO_BASE || 'http://127.0.0.1:8090';
const EMAIL = process.env.HYDRO_ADMIN_EMAIL || 'admin@haberdiyari.com';
const PASSWORD = process.env.HYDRO_ADMIN_PASSWORD || 'HydroNews2026!SecureKey';

let cookieJar = [];
let csrfToken = '';

function storeCookies(res) {
  const raw = res.headers.getSetCookie ? res.headers.getSetCookie() : res.headers.raw?.()['set-cookie'] || [];
  for (const c of raw) {
    const pair = c.split(';')[0];
    const name = pair.split('=')[0];
    cookieJar = cookieJar.filter((e) => !e.startsWith(`${name}=`));
    cookieJar.push(pair);
  }
}

async function request(path, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
    Cookie: cookieJar.join('; '),
    ...(options.headers || {}),
  };
  if (['POST', 'PUT', 'DELETE', 'PATCH'].includes((options.method || 'GET').toUpperCase())) {
    headers['X-CSRF-Token'] = csrfToken;
  }
  const res = await fetch(`${BASE}${path}`, { ...options, headers });
  storeCookies(res);
  const text = await res.text();
  const body = text ? JSON.parse(text) : {};
  if (!res.ok) {
    throw new Error(`${options.method || 'GET'} ${path} -> ${res.status}: ${JSON.stringify(body)}`);
  }
  return body;
}

async function login() {
  const csrf = await request('/csrf-token');
  csrfToken = csrf.token;
  await request('/auth/login', { method: 'POST', body: JSON.stringify({ email: EMAIL, password: PASSWORD }) });
  console.log('Logged in as admin.');
}

async function clearExisting(type) {
  const existing = await request(`/api/v1/admin/content/${type}?limit=100&locale=tr`);
  for (const item of existing.data || []) {
    await request(`/api/v1/admin/content/${type}/${item.id}`, { method: 'DELETE' });
  }
}

async function createAndPublish(type, fields, { drafts = true } = {}) {
  const created = await request(`/api/v1/admin/content/${type}`, {
    method: 'POST',
    body: JSON.stringify({ locale: 'tr', ...fields }),
  });
  if (drafts) {
    await request(`/api/v1/admin/content/${type}/${created.data.id}/workflow`, {
      method: 'POST',
      body: JSON.stringify({ target: 'published', version: created.data.version }),
    });
  }
  return created.data;
}

const NAV_ITEMS = [
  { label: 'Gündem', href: '/gundem', placement: 'header', sort_order: 1 },
  { label: 'Dünya', href: '/dunya', placement: 'header', sort_order: 2 },
  { label: 'Ekonomi', href: '/ekonomi', placement: 'header', sort_order: 3 },
  { label: 'Teknoloji', href: '/teknoloji', placement: 'header', sort_order: 4 },
  { label: 'Spor', href: '/spor', placement: 'header', sort_order: 5 },
  { label: 'Resmi İlanlar', href: '/resmi-ilanlar', placement: 'header', sort_order: 7 },
  { label: 'Künye', href: '/kunye', placement: 'header', sort_order: 9 },
  { label: 'Son Dakika', href: '/son-dakika', placement: 'header', sort_order: 0 },

  { label: 'Anasayfa', href: '/', placement: 'footer_sayfalar', sort_order: 1 },
  { label: 'Haberler', href: '/gundem', placement: 'footer_sayfalar', sort_order: 2 },
  { label: 'Künye', href: '/kunye', placement: 'footer_sayfalar', sort_order: 3 },
  { label: 'İletişim', href: '/iletisim', placement: 'footer_sayfalar', sort_order: 4 },
  { label: 'Gizlilik Politikası', href: '/sayfa/gizlilik-politikasi', placement: 'footer_sayfalar', sort_order: 5 },
  { label: 'Çerez Politikası', href: '/sayfa/cerez-politikasi', placement: 'footer_sayfalar', sort_order: 6 },
  { label: 'Yayın İlkeleri', href: '/sayfa/yayin-ilkeleri', placement: 'footer_sayfalar', sort_order: 7 },

  { label: 'X', href: 'https://x.com/hydropost', placement: 'footer_sosyal', sort_order: 1 },
  { label: 'Instagram', href: 'https://instagram.com/hydropost', placement: 'footer_sosyal', sort_order: 2 },
];

const COLUMNISTS = [
  { name: 'Elif Arslan', title: 'Ekonomi Yorumcusu', latest_article_title: 'Faiz kararının piyasalara etkisi', sort_order: 1 },
  { name: 'Kerem Doğan', title: 'Dış Politika Editörü', latest_article_title: 'Bölgesel dengeler yeniden şekilleniyor', sort_order: 2 },
  { name: 'Selin Kaya', title: 'Spor Yazarı', latest_article_title: 'Sezonun ikinci yarısında kritik haftalar', sort_order: 3 },
  { name: 'Burak Yıldız', title: 'Teknoloji Editörü', latest_article_title: 'Yapay zekâ düzenlemelerinde yeni dönem', sort_order: 4 },
];

async function run() {
  await login();

  console.log('Clearing existing site_navigation / site_settings / columnists / polls / homepage_sections records...');
  await clearExisting('site_navigation');
  await clearExisting('site_settings');
  await clearExisting('columnists');
  await clearExisting('polls');
  await clearExisting('homepage_sections');

  const HOMEPAGE_SECTIONS = [
    { key: 'hero', visible: true, sort_order: 1 },
    { key: 'son_haberler', visible: true, sort_order: 2 },
    { key: 'kategori_vitrini', category_filter: 'Gündem', visible: true, sort_order: 3 },
    { key: 'kategori_vitrini', category_filter: 'Ekonomi', visible: true, sort_order: 4 },
    { key: 'bulten', visible: true, sort_order: 5 },
    { key: 'kategori_vitrini', category_filter: 'Teknoloji', visible: true, sort_order: 6 },
    { key: 'kategori_vitrini', category_filter: 'Dünya', visible: true, sort_order: 7 },
    { key: 'ikinci_grid', visible: true, sort_order: 8 },
    { key: 'kategori_vitrini', category_filter: 'Spor', visible: true, sort_order: 9 },
    { key: 'kategori_vitrini', category_filter: 'Sağlık', visible: true, sort_order: 10 },
    { key: 'kategori_vitrini', category_filter: 'Yaşam', visible: true, sort_order: 11 },
    { key: 'cok_okunan', visible: true, sort_order: 12 },
    { key: 'galeri', visible: true, sort_order: 13 },
  ];
  for (const s of HOMEPAGE_SECTIONS) {
    await createAndPublish('homepage_sections', s, { drafts: false });
  }
  console.log(`Seeded ${HOMEPAGE_SECTIONS.length} homepage sections.`);

  await createAndPublish('polls', {
    question: 'Sizce Süper Lig 2026 sezonunda şampiyonluğun en güçlü adayı kim?',
    options: 'Galatasaray,Fenerbahçe,Beşiktaş,Trabzonspor,Diğer',
    category: 'spor',
    total_votes: 1482,
    votes_data: JSON.stringify({ Galatasaray: 620, Fenerbahçe: 480, Beşiktaş: 230, Trabzonspor: 110, Diğer: 42 }),
    is_active: true,
  });
  console.log('Seeded 1 active poll.');

  for (const c of COLUMNISTS) {
    await createAndPublish('columnists', c);
  }
  console.log(`Seeded ${COLUMNISTS.length} columnists.`);

  for (const item of NAV_ITEMS) {
    await createAndPublish('site_navigation', item);
  }
  console.log(`Seeded ${NAV_ITEMS.length} navigation items.`);

  await createAndPublish('site_settings', {
    site_name: 'HydroPost',
    tagline: 'Bağımsız Haber Portalı',
    footer_description: 'HydroGo motoru üzerinde çalışan bağımsız, ultra hızlı ve güvenilir haber portalı.',
    footer_highlight_title: 'HydroPost',
    footer_highlight_text: 'Türkiye ve dünya gündemini bir araya getiriyor; hızlı ve güvenilir şekilde okuyucularımıza ulaştırıyoruz.',
    footer_highlight_cta_text: 'Haberleri Oku',
    footer_highlight_cta_href: '/gundem',
    copyright_text: 'Tüm hakları saklıdır.',
    seo_title: 'HydroPost — Bağımsız ve Güçlü Haber Portalı',
    seo_description: 'HydroGo tekil binary mimarisiyle çalışan, ışık hızında haber portalı.',
    comments_enabled: true,
    comments_require_email: false,
  }, { drafts: false });
  console.log('Seeded site_settings.');

  const LEGAL_PAGES = [
    {
      slug: 'gizlilik-politikasi',
      title: 'Gizlilik Politikası ve KVKK Aydınlatma Metni',
      body: `<h2>1. Veri Sorumlusu ve Kapsam</h2><p>HydroPost ("Portal"), 6698 sayılı Kişisel Verilerin Korunması Kanunu ("KVKK") uyarınca veri sorumlusu sıfatıyla, ziyaretçilerimizin kişisel verilerinin güvenliğine ve gizliliğine azami hassasiyet göstermektedir. Bu metin, sitemizi ziyaret eden kullanıcılarımızın verilerinin toplanma yöntemleri, işlenme amaçları ve yasal hakları konusunda bilgilendirilmesi amacıyla hazırlanmıştır.</p><h2>2. İşlenen Kişisel Veriler</h2><p>Haber portalımızı kullanımınız esnasında; IP adresi, tarayıcı bilgileri, ziyaret edilen sayfalar, erişim saatleri ve yorum alanlarına isteğe bağlı girilen isim ve e-posta adresleri gibi sınırlı teknik veriler işlenebilmektedir.</p><h2>3. Kişisel Verilerin İşlenme Amaçları</h2><p>Toplanan kişisel veriler; 5651 sayılı İnternet Ortamında Yapılan Yayınların Düzenlenmesi Hakkında Kanun kapsamındaki yasal yükümlülüklerin yerine getirilmesi, haber platformunun güvenliğinin ve performansının sağlanması ve kullanıcı deneyiminin iyileştirilmesi amacıyla işlenmektedir.</p><h2>4. Veri Aktarımı ve Güvenlik</h2><p>Kişisel verileriniz, yasal zorunluluklar veya adli makamların resmi talepleri haricinde hiçbir üçüncü şahıs veya kuruma ticari amaçla satılmaz veya devredilmez. Verileriniz HydroGo'nun yüksek güvenlik standartlarına sahip sunucularında korunmaktadır.</p><h2>5. İlgili Kişinin Hakları</h2><p>KVKK'nın 11. maddesi uyarınca veri sahipleri; kişisel verilerinin işlenip işlenmediğini öğrenme, işlenmişse buna ilişkin bilgi talep etme ve verilerin silinmesini veya düzeltilmesini isteme hakkına sahiptir.</p>`,
    },
    {
      slug: 'cerez-politikasi',
      title: 'Çerez (Cookie) Politikası',
      body: `<h2>1. Çerez Nedir?</h2><p>Çerezler (cookies), bir internet sitesini ziyaret ettiğinizde tarayıcınız aracılığıyla cihazınıza veya ağ sunucusuna depolanan küçük metin dosyalarıdır. Çerezler sitenin daha verimli çalışmasını ve tercihlerinize uygun içerik sunulmasını sağlar.</p><h2>2. Kullanılan Çerez Türleri</h2><p><strong>Zorunlu Çerezler:</strong> Web sitemizin düzgün çalışması, oturum yönetimi ve güvenlik doğrulamaları için kesinlikle gerekli olan çerezlerdir.</p><p><strong>İşlevsel Çerezler:</strong> Yazı boyutu tercihiniz, tema seçiminiz ve anket oylama durumunuz gibi tercihlerin hatırlanmasını sağlayan çerezlerdir.</p><p><strong>Analitik Çerezler:</strong> Hangi haberlerin daha çok okunduğunu, site trafiğini ve ziyaretçi akışını anonim olarak ölçümleyen çerezlerdir.</p><h2>3. Çerezlerin Kontrolü ve Silinmesi</h2><p>Tarayıcınızın ayarlarını değiştirerek çerezlere ilişkin tercihlerinizi kişiselleştirebilirsiniz. Çerezleri devre dışı bırakmanız durumunda web sitesinin bazı işlevleri kısıtlanabilir.</p>`,
    },
    {
      slug: 'yayin-ilkeleri',
      title: 'Yayın İlkeleri ve Editoryal Bağımsızlık',
      body: `<h2>1. Doğruluk ve Güvenilirlik</h2><p>HydroPost; haberin doğruluğunu teyit etmeden yayınlamamayı, teyit edilmemiş iddiaları kesin olgular gibi sunmamayı temel yayın ilkesi olarak kabul eder.</p><h2>2. Tarafsızlık ve Çoğulculuk</h2><p>Haber bültenlerimizde her türlü siyasi, ekonomik ve ideolojik baskıdan uzak, kamuoyunun haber alma hakkına saygılı ve tarafsız bir editoryal duruş sergilenir.</p><h2>3. Masumiyet Karinesi ve Kişilik Hakları</h2><p>Yargı kararı kesinleşmedikçe hiç kimse suçlu ilan edilemez. Özel hayatın gizliliğine, kişilik haklarına ve insan onuruna saygı gösterilir.</p><h2>4. Düzeltme ve Cevap Hakkı</h2><p>Hatalı veya eksik yayınlanan bilgiler, tespit edildiği anda gecikmeksizin şeffaf bir şekilde düzeltilir. İlgili kişi ve kurumların yasal tekzip ve cevap haklarına titizlikle riayet edilir.</p>`,
    },
  ];

  for (const page of LEGAL_PAGES) {
    try {
      const existing = await request(`/api/v1/pages/tr/${page.slug}`);
      if (existing?.data?.id) {
        console.log(`Page '${page.slug}' already exists.`);
        continue;
      }
    } catch {
      // not found, proceed to create
    }
    try {
      const created = await request('/api/v1/admin/pages', {
        method: 'POST',
        body: JSON.stringify({
          locale: 'tr',
          slug: page.slug,
          title: page.title,
          body: page.body,
          blocks: [],
        }),
      });
      if (created?.data?.id) {
        await request(`/api/v1/admin/pages/${created.data.id}/publish`, {
          method: 'POST',
          body: JSON.stringify({ version: created.data.version }),
        });
        console.log(`Created and published page '${page.slug}'.`);
      }
    } catch (e) {
      console.warn(`Could not seed page '${page.slug}':`, e.message);
    }
  }

  console.log('Done.');
}

run().catch((err) => {
  console.error(err);
  process.exit(1);
});
