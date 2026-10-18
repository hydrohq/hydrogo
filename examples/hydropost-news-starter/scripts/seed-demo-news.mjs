// Seeds realistic-looking Turkish news demo content for the HydroPost news
// starter, replacing HydroGo's own generic product-feature demo articles
// (cache strategies, database portability, ...) that ship with `hydro seed`.
//
// Usage: node scripts/seed-demo-news.mjs
// Requires the backend running on http://127.0.0.1:8090 with the admin
// credentials below (see ../README.md).

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

function quillBody(paragraphs) {
  const ops = [];
  paragraphs.forEach((p, i) => {
    if (i === 0) ops.push({ insert: `${p}\n`, attributes: { header: 1 } });
    else ops.push({ insert: `${p}\n` });
  });
  return JSON.stringify({ ops });
}

// Deliberately generic entities (no real public figures or institutions) so
// this seed data is safe to ship as starter/demo content.
const ARTICLES = [
  {
    category: 'Gündem',
    title: 'Büyükşehirlerde toplu taşıma ücretlerine güncelleme geldi',
    excerpt: 'Belediye meclisi, artan işletme maliyetleri gerekçesiyle toplu taşıma tarifelerinde yeni dönem ücretlerini onayladı.',
    body: ['Toplu Taşımada Yeni Tarife Dönemi', 'Belediye meclisi dün yaptığı oturumda, 2026 yılı için toplu taşıma ücret tarifesini güncelledi. Yeni tarife önümüzdeki ay yürürlüğe girecek.', 'Yetkililer, güncellemenin artan enerji ve bakım maliyetlerini karşılamak amacıyla yapıldığını belirtti.'],
    featured: true,
  },
  {
    category: 'Gündem',
    title: 'Kentsel dönüşüm projesinde yeni etap başlıyor',
    excerpt: 'Deprem riski taşıyan bölgelerde sürdürülen kentsel dönüşüm çalışmalarının ikinci etabı için ihale süreci tamamlandı.',
    body: ['Dönüşümde İkinci Etap', 'Kentsel dönüşüm kapsamında belirlenen risk bölgelerinde ikinci etap çalışmaları başlıyor.', 'Proje kapsamında çok sayıda bina güçlendirilecek ya da yeniden inşa edilecek.'],
  },
  {
    category: 'Gündem',
    title: 'Hava durumu uyarısı: Kuvvetli yağış bekleniyor',
    excerpt: 'Meteoroloji, önümüzdeki 48 saat içinde birçok ilde kuvvetli yağış ve fırtına uyarısında bulundu.',
    body: ['Sağanak Uyarısı', 'Meteoroloji Genel Müdürlüğü, önümüzdeki iki gün içinde etkili olacak sağanak yağış için sarı kodlu uyarı yayınladı.', 'Vatandaşların ani sel baskınlarına karşı tedbirli olması istendi.'],
  },
  {
    category: 'Dünya',
    title: 'Avrupa\'da enerji fiyatları yeniden gündemde',
    excerpt: 'Kış ayı yaklaşırken enerji tedarik güvenliği ve fiyat istikrarı konusu Avrupa başkentlerinde yeniden tartışılıyor.',
    body: ['Kış Öncesi Enerji Tartışması', 'Avrupa ülkeleri, kış aylarına girerken enerji arzının sürekliliğini sağlamak için yeni tedbirler değerlendiriyor.', 'Uzmanlar, depolama kapasitelerinin geçen yıla göre daha iyi durumda olduğunu belirtiyor.'],
  },
  {
    category: 'Dünya',
    title: 'Asya borsalarında karışık seyir',
    excerpt: 'Bölgedeki büyük borsalar, küresel büyüme endişeleri ile teknoloji hisselerindeki toparlanma arasında karışık bir seyir izledi.',
    body: ['Karışık Kapanış', 'Asya borsaları günü karışık bir seyirle tamamladı.', 'Yatırımcılar bu hafta açıklanacak büyüme verilerini yakından takip ediyor.'],
  },
  {
    category: 'Dünya',
    title: 'Uluslararası zirvede iklim taahhütleri masada',
    excerpt: 'Katılımcı ülkeler, emisyon azaltım hedeflerini güncellemek üzere bir araya geldi.',
    body: ['İklim Zirvesi', 'Çok sayıda ülkenin temsilcisi, emisyon azaltım taahhütlerini gözden geçirmek üzere bir araya geldi.', 'Zirve sonunda ortak bir eylem planı açıklanması bekleniyor.'],
  },
  {
    category: 'Ekonomi',
    title: 'Merkez Bankası faiz kararını açıkladı',
    excerpt: 'Para Politikası Kurulu, politika faizinde değişikliğe gitmeyerek piyasa beklentileriyle uyumlu bir karar aldı.',
    body: ['Faiz Sabit Kaldı', 'Merkez Bankası Para Politikası Kurulu, bu ayki toplantısında politika faizini sabit tuttu.', 'Kurul, enflasyon görünümüne bağlı olarak politika duruşunun sıkı tutulmaya devam edileceğini vurguladı.'],
    featured: true,
  },
  {
    category: 'Ekonomi',
    title: 'Perakende sektöründe yılın son çeyreği hareketli geçecek',
    excerpt: 'Sektör temsilcileri, kampanya dönemleriyle birlikte son çeyrekte satışlarda belirgin bir artış bekliyor.',
    body: ['Son Çeyrek Beklentisi', 'Perakende sektörü temsilcileri, yıl sonu kampanyalarıyla birlikte satışlarda artış öngörüyor.', 'E-ticaret hacminin de bu dönemde payını artırması bekleniyor.'],
  },
  {
    category: 'Ekonomi',
    title: 'Tarımsal üretimde verim artışı hedefleniyor',
    excerpt: 'Bakanlık, yeni destekleme paketiyle birlikte temel ürünlerde verim artışı hedeflediğini açıkladı.',
    body: ['Yeni Destekleme Paketi', 'Tarım alanında yeni bir destekleme paketi devreye alındı.', 'Paket, özellikle su kullanımında verimliliği artıran yöntemleri teşvik ediyor.'],
  },
  {
    category: 'Teknoloji',
    title: 'Yerli yazılım şirketlerinden yapay zekâ iş birliği',
    excerpt: 'İki yerli teknoloji şirketi, yapay zekâ destekli ürün geliştirmek için güçlerini birleştirdiğini duyurdu.',
    body: ['Ortak Geliştirme Süreci', 'Yerli iki teknoloji şirketi, yapay zekâ destekli yeni bir ürün için iş birliği anlaşması imzaladı.', 'Ürünün önümüzdeki yıl kullanıma sunulması planlanıyor.'],
    featured: true,
  },
  {
    category: 'Teknoloji',
    title: 'Elektrikli araç şarj ağı genişliyor',
    excerpt: 'Şehirlerarası yollarda şarj istasyonu sayısının artırılmasına yönelik yeni yatırım planı açıklandı.',
    body: ['Şarj Altyapısı Büyüyor', 'Elektrikli araç kullanıcıları için şarj ağı genişletiliyor.', 'Yeni istasyonların büyük bölümü şehirlerarası ana güzergahlara kurulacak.'],
  },
  {
    category: 'Teknoloji',
    title: 'Siber güvenlikte yeni nesil tehditlere karşı önlem çağrısı',
    excerpt: 'Uzmanlar, kurumları yapay zekâ destekli siber saldırılara karşı altyapılarını güncellemeye çağırdı.',
    body: ['Artan Tehdit Ortamı', 'Siber güvenlik uzmanları, yapay zekâ destekli saldırı tekniklerinin yaygınlaştığı konusunda uyardı.', 'Kurumlara çok katmanlı güvenlik mimarisine geçiş önerildi.'],
  },
  {
    category: 'Spor',
    title: 'Süper Lig\'de zirve yarışı kızışıyor',
    excerpt: 'Ligde oynanan maçların ardından puan farkı daralırken şampiyonluk yarışı son haftalara damgasını vuruyor.',
    body: ['Zirvede Gerilim', 'Süper Lig\'de üst sıralardaki takımlar arasındaki puan farkı iyice azaldı.', 'Taraftarlar, sezonun geri kalanının oldukça çekişmeli geçmesini bekliyor.'],
    featured: true,
  },
  {
    category: 'Spor',
    title: 'Milli takım hazırlık kampına başladı',
    excerpt: 'A Milli Takım, önümüzdeki maçlar öncesi hazırlık kampına girdi.',
    body: ['Kamp Başladı', 'Milli takım, yaklaşan karşılaşmalar öncesinde hazırlık kampına başladı.', 'Teknik heyet, kamp boyunca fiziksel hazırlığa ağırlık vereceğini açıkladı.'],
  },
  {
    category: 'Spor',
    title: 'Genç sporculardan uluslararası turnuvada başarı',
    excerpt: 'Genç millî takım, uluslararası turnuvada madalya kazanarak başarılı bir performans sergiledi.',
    body: ['Madalya Sevinci', 'Genç millî takım, katıldığı uluslararası turnuvada madalya kazandı.', 'Sporcular, başarıyı emek veren herkese armağan ettiklerini söyledi.'],
  },
  {
    category: 'Gündem',
    title: 'Şehir içi bisiklet yolları ağı genişletiliyor',
    excerpt: 'Belediye, sürdürülebilir ulaşım hedefi kapsamında bisiklet yolu uzunluğunu iki katına çıkarmayı planlıyor.',
    body: ['Bisiklet Yolu Seferberliği', 'Belediye, şehir genelinde bisiklet yolu ağını genişletme kararı aldı.', 'Çalışmaların önümüzdeki yıl tamamlanması hedefleniyor.'],
  },
  {
    category: 'Gündem',
    title: 'Okullarda yeni eğitim dönemi hazırlıkları tamamlandı',
    excerpt: 'İl Millî Eğitim Müdürlüğü, yeni dönem öncesi okullarda son hazırlıkların tamamlandığını açıkladı.',
    body: ['Hazırlıklar Tamam', 'Okullarda yeni eğitim dönemi öncesi bakım ve onarım çalışmaları sona erdi.', 'Yetkililer, tüm okulların eğitime hazır olduğunu belirtti.'],
  },
  {
    category: 'Dünya',
    title: 'Avrupa Birliği göç politikasında yeni adım attı',
    excerpt: 'Üye ülkeler, ortak göç yönetimi çerçevesinde yeni bir mutabakata vardı.',
    body: ['Yeni Mutabakat', 'Avrupa Birliği üyesi ülkeler, göç yönetimi konusunda ortak bir çerçevede anlaştı.', 'Mutabakatın önümüzdeki aylarda yürürlüğe girmesi bekleniyor.'],
  },
  {
    category: 'Dünya',
    title: 'Küresel tedarik zincirinde toparlanma sinyalleri',
    excerpt: 'Uzmanlar, deniz taşımacılığındaki gecikmelerin geçen yıla göre azaldığını belirtiyor.',
    body: ['Toparlanma Sinyalleri', 'Küresel tedarik zincirinde son aylarda belirgin bir toparlanma gözlemleniyor.', 'Deniz taşımacılığı maliyetleri de gerileme eğiliminde.'],
  },
  {
    category: 'Ekonomi',
    title: 'İhracatçılardan yeni pazar atılımı',
    excerpt: 'Sektör temsilcileri, Uzak Doğu pazarına yönelik yeni ihracat hedefleri belirledi.',
    body: ['Yeni Pazar Hedefi', 'İhracatçı birlikleri, Uzak Doğu pazarına yönelik atılım kararı aldı.', 'Hedef, önümüzdeki üç yılda ihracatı ikiye katlamak.'],
  },
  {
    category: 'Ekonomi',
    title: 'Konut kredisi faizlerinde hareketlilik',
    excerpt: 'Bankalar, konut kredisi faiz oranlarını güncelledi; uzmanlar talebin artmasını bekliyor.',
    body: ['Faizlerde Güncelleme', 'Bankalar konut kredisi faiz oranlarında güncellemeye gitti.', 'Sektör temsilcileri, talebin önümüzdeki aylarda artmasını bekliyor.'],
  },
  {
    category: 'Teknoloji',
    title: 'Yerli uydu projesinde yeni aşamaya geçildi',
    excerpt: 'Proje ekibi, uydunun fırlatma öncesi test süreçlerinin tamamlandığını duyurdu.',
    body: ['Test Süreci Tamamlandı', 'Yerli uydu projesinde fırlatma öncesi test aşaması tamamlandı.', 'Uydunun önümüzdeki aylarda yörüngeye yerleştirilmesi planlanıyor.'],
  },
  {
    category: 'Teknoloji',
    title: 'Açık kaynak yazılım topluluğu büyüyor',
    excerpt: 'Yerli geliştiriciler, açık kaynak projelere katkı sayısında rekor bir artış kaydetti.',
    body: ['Katkılar Artıyor', 'Yerli yazılım geliştiricilerin açık kaynak projelere katkısı son bir yılda belirgin şekilde arttı.', 'Topluluk, yeni geliştiricileri desteklemek için mentorluk programı başlattı.'],
  },
];

async function run() {
  await login();

  // Retract HydroGo's own generic product-demo articles so they stop
  // appearing on the portal; they remain in the DB as drafts/history.
  const existing = await request('/api/v1/admin/news?limit=100&locale=tr');
  for (const item of existing.data || []) {
    if (item.status === 'published' && item.id?.startsWith('demo-news-')) {
      await request(`/api/v1/admin/news/${item.id}/retract`, {
        method: 'POST',
        body: JSON.stringify({ version: item.version }),
      });
      console.log(`Retracted ${item.id}`);
    }
  }

  for (const article of ARTICLES) {
    const slug = article.title
      .toLowerCase()
      .replace(/ğ/g, 'g').replace(/ü/g, 'u').replace(/ş/g, 's')
      .replace(/ı/g, 'i').replace(/ö/g, 'o').replace(/ç/g, 'c')
      .replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');

    const created = await request('/api/v1/admin/news', {
      method: 'POST',
      body: JSON.stringify({
        locale: 'tr',
        slug,
        title: article.title,
        body: quillBody(article.body),
        excerpt: article.excerpt,
        category: article.category,
        author_name: 'Haber Merkezi',
        featured: Boolean(article.featured),
        tags: article.category,
      }),
    });

    await request(`/api/v1/admin/news/${created.data.id}/publish`, {
      method: 'POST',
      body: JSON.stringify({ version: created.data.version }),
    });
    console.log(`Published: ${article.title}`);
  }

  console.log('Done.');
}

run().catch((err) => {
  console.error(err);
  process.exit(1);
});
