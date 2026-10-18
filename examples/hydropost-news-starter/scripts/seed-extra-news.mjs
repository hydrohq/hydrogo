// One-off: adds the 10 category-fill articles appended to seed-demo-news.mjs
// without re-running the full script (which would duplicate the first 15).
// Safe to delete after running once against a fresh DB that already has the
// original 15 from seed-demo-news.mjs.
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
  const headers = { 'Content-Type': 'application/json', Accept: 'application/json', Cookie: cookieJar.join('; '), ...(options.headers || {}) };
  if (['POST', 'PUT', 'DELETE', 'PATCH'].includes((options.method || 'GET').toUpperCase())) headers['X-CSRF-Token'] = csrfToken;
  const res = await fetch(`${BASE}${path}`, { ...options, headers });
  storeCookies(res);
  const text = await res.text();
  const body = text ? JSON.parse(text) : {};
  if (!res.ok) throw new Error(`${options.method || 'GET'} ${path} -> ${res.status}: ${JSON.stringify(body)}`);
  return body;
}

function quillBody(paragraphs) {
  const ops = [];
  paragraphs.forEach((p, i) => (i === 0 ? ops.push({ insert: `${p}\n`, attributes: { header: 1 } }) : ops.push({ insert: `${p}\n` })));
  return JSON.stringify({ ops });
}

const ARTICLES = [
  { category: 'Gündem', title: 'Şehir içi bisiklet yolları ağı genişletiliyor', excerpt: 'Belediye, sürdürülebilir ulaşım hedefi kapsamında bisiklet yolu uzunluğunu iki katına çıkarmayı planlıyor.', body: ['Bisiklet Yolu Seferberliği', 'Belediye, şehir genelinde bisiklet yolu ağını genişletme kararı aldı.', 'Çalışmaların önümüzdeki yıl tamamlanması hedefleniyor.'] },
  { category: 'Gündem', title: 'Okullarda yeni eğitim dönemi hazırlıkları tamamlandı', excerpt: 'İl Millî Eğitim Müdürlüğü, yeni dönem öncesi okullarda son hazırlıkların tamamlandığını açıkladı.', body: ['Hazırlıklar Tamam', 'Okullarda yeni eğitim dönemi öncesi bakım ve onarım çalışmaları sona erdi.', 'Yetkililer, tüm okulların eğitime hazır olduğunu belirtti.'] },
  { category: 'Dünya', title: 'Avrupa Birliği göç politikasında yeni adım attı', excerpt: 'Üye ülkeler, ortak göç yönetimi çerçevesinde yeni bir mutabakata vardı.', body: ['Yeni Mutabakat', 'Avrupa Birliği üyesi ülkeler, göç yönetimi konusunda ortak bir çerçevede anlaştı.', 'Mutabakatın önümüzdeki aylarda yürürlüğe girmesi bekleniyor.'] },
  { category: 'Dünya', title: 'Küresel tedarik zincirinde toparlanma sinyalleri', excerpt: 'Uzmanlar, deniz taşımacılığındaki gecikmelerin geçen yıla göre azaldığını belirtiyor.', body: ['Toparlanma Sinyalleri', 'Küresel tedarik zincirinde son aylarda belirgin bir toparlanma gözlemleniyor.', 'Deniz taşımacılığı maliyetleri de gerileme eğiliminde.'] },
  { category: 'Ekonomi', title: 'İhracatçılardan yeni pazar atılımı', excerpt: 'Sektör temsilcileri, Uzak Doğu pazarına yönelik yeni ihracat hedefleri belirledi.', body: ['Yeni Pazar Hedefi', 'İhracatçı birlikleri, Uzak Doğu pazarına yönelik atılım kararı aldı.', 'Hedef, önümüzdeki üç yılda ihracatı ikiye katlamak.'] },
  { category: 'Ekonomi', title: 'Konut kredisi faizlerinde hareketlilik', excerpt: 'Bankalar, konut kredisi faiz oranlarını güncelledi; uzmanlar talebin artmasını bekliyor.', body: ['Faizlerde Güncelleme', 'Bankalar konut kredisi faiz oranlarında güncellemeye gitti.', 'Sektör temsilcileri, talebin önümüzdeki aylarda artmasını bekliyor.'] },
  { category: 'Teknoloji', title: 'Yerli uydu projesinde yeni aşamaya geçildi', excerpt: 'Proje ekibi, uydunun fırlatma öncesi test süreçlerinin tamamlandığını duyurdu.', body: ['Test Süreci Tamamlandı', 'Yerli uydu projesinde fırlatma öncesi test aşaması tamamlandı.', 'Uydunun önümüzdeki aylarda yörüngeye yerleştirilmesi planlanıyor.'] },
  { category: 'Teknoloji', title: 'Açık kaynak yazılım topluluğu büyüyor', excerpt: 'Yerli geliştiriciler, açık kaynak projelere katkı sayısında rekor bir artış kaydetti.', body: ['Katkılar Artıyor', 'Yerli yazılım geliştiricilerin açık kaynak projelere katkısı son bir yılda belirgin şekilde arttı.', 'Topluluk, yeni geliştiricileri desteklemek için mentorluk programı başlattı.'] },
  { category: 'Spor', title: 'Yüzmede yeni Türkiye rekoru kırıldı', excerpt: 'Milli yüzücü, 200 metre serbest stilde yeni Türkiye rekoruna imza attı.', body: ['Yeni Rekor', 'Milli yüzücü, 200 metre serbest stilde Türkiye rekorunu kırdı.', 'Sporcu, hedefinin uluslararası arenada da başarı elde etmek olduğunu söyledi.'] },
  { category: 'Spor', title: 'Voleybolda play-off heyecanı başlıyor', excerpt: 'Ligde play-off eşleşmeleri belli oldu, ilk maçlar bu hafta sonu oynanacak.', body: ['Play-off Başlıyor', 'Voleybol liginde play-off eşleşmeleri açıklandı.', 'İlk maçların bu hafta sonu oynanması bekleniyor.'] },
];

async function run() {
  const csrf = await request('/csrf-token');
  csrfToken = csrf.token;
  await request('/auth/login', { method: 'POST', body: JSON.stringify({ email: EMAIL, password: PASSWORD }) });
  console.log('Logged in as admin.');

  for (const article of ARTICLES) {
    const slug = article.title.toLowerCase().replace(/ğ/g, 'g').replace(/ü/g, 'u').replace(/ş/g, 's').replace(/ı/g, 'i').replace(/ö/g, 'o').replace(/ç/g, 'c').replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
    const created = await request('/api/v1/admin/news', {
      method: 'POST',
      body: JSON.stringify({ locale: 'tr', slug, title: article.title, body: quillBody(article.body), excerpt: article.excerpt, category: article.category, author_name: 'Haber Merkezi', featured: false, tags: article.category }),
    });
    await request(`/api/v1/admin/news/${created.data.id}/publish`, { method: 'POST', body: JSON.stringify({ version: created.data.version }) });
    console.log(`Published: ${article.title}`);
  }
  console.log('Done.');
}

run().catch((err) => {
  console.error(err);
  process.exit(1);
});
