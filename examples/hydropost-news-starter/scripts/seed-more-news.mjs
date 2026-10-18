// Adds a much larger batch of demo articles (10 more per category) on top
// of seed-demo-news.mjs + seed-extra-news.mjs, so the site has enough
// volume for pagination, "Son Haberler", "Çok Okunanlar" and the category
// manşet widgets to feel like a real, living archive instead of a thin demo.
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

function slugOf(title) {
  return title.toLowerCase().replace(/ğ/g, 'g').replace(/ü/g, 'u').replace(/ş/g, 's').replace(/ı/g, 'i').replace(/ö/g, 'o').replace(/ç/g, 'c').replace(/'/g, '').replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
}

const ARTICLES = [
  // Gündem
  { category: 'Gündem', title: 'Deprem tatbikatı tüm okullarda gerçekleştirildi', excerpt: 'İl genelindeki okullarda eş zamanlı deprem tatbikatı yapıldı, öğrenciler tahliye prosedürünü uyguladı.', body: ['Tatbikat Tamamlandı', 'İl genelindeki tüm okullarda eş zamanlı deprem tatbikatı gerçekleştirildi.', 'Yetkililer, tatbikatın sorunsuz tamamlandığını açıkladı.'] },
  { category: 'Gündem', title: 'Yeni toplu konut projesi için başvurular başladı', excerpt: 'Düşük gelirli ailelere yönelik toplu konut projesinde başvuru süreci bugün itibarıyla açıldı.', body: ['Başvurular Açıldı', 'Toplu konut projesi kapsamında başvurular bugün itibarıyla alınmaya başlandı.', 'Başvuruların bir ay içinde sonuçlanması bekleniyor.'] },
  { category: 'Gündem', title: 'Şehir merkezinde trafik düzenlemesi hayata geçti', excerpt: 'Yoğun saatlerde yaşanan trafik sıkışıklığını azaltmak için yeni şerit düzenlemesi uygulanmaya başlandı.', body: ['Yeni Düzenleme', 'Şehir merkezinde yeni trafik düzenlemesi bugün yürürlüğe girdi.', 'Sürücülerin yeni tabelalara dikkat etmesi istendi.'] },
  { category: 'Gündem', title: 'Kütüphane haftası etkinlikleri start aldı', excerpt: 'Kütüphane haftası kapsamında şehir genelinde söyleşi ve atölye etkinlikleri düzenlenecek.', body: ['Etkinlikler Başladı', 'Kütüphane haftası etkinlikleri bu hafta boyunca sürecek.', 'Etkinliklere katılım ücretsiz.'] },
  { category: 'Gündem', title: 'Belediye otobüs filosuna yeni araçlar eklendi', excerpt: 'Toplu taşıma kapasitesini artırmak amacıyla filoya yeni düşük emisyonlu otobüsler katıldı.', body: ['Filo Güçlendi', 'Belediye otobüs filosuna yeni araçlar eklendi.', 'Yeni araçların düşük emisyon standartlarına uygun olduğu belirtildi.'] },
  { category: 'Gündem', title: 'Kan bağışı kampanyasına yoğun katılım', excerpt: 'Şehir meydanında düzenlenen kan bağışı kampanyasına vatandaşlardan yoğun ilgi gösterildi.', body: ['Yoğun Katılım', 'Kan bağışı kampanyasına beklenenin üzerinde katılım oldu.', 'Kampanya önümüzdeki hafta farklı bir noktada devam edecek.'] },
  { category: 'Gündem', title: 'Sahil şeridinde yürüyüş yolu yenilendi', excerpt: 'Sahil şeridindeki yürüyüş yolu bakım çalışmasının ardından yeniden vatandaşların kullanımına açıldı.', body: ['Yenileme Tamamlandı', 'Sahil şeridindeki yürüyüş yolu yenileme çalışması tamamlandı.', 'Bölge, özellikle hafta sonları yoğun ilgi görüyor.'] },
  { category: 'Gündem', title: 'Kamu binalarında enerji verimliliği çalışması başladı', excerpt: 'Kamu binalarının enerji tüketimini azaltmak amacıyla yalıtım ve aydınlatma yenileme çalışması başlatıldı.', body: ['Verimlilik Çalışması', 'Kamu binalarında enerji verimliliği çalışmaları başladı.', 'Çalışmanın yıl sonuna kadar tamamlanması planlanıyor.'] },
  { category: 'Gündem', title: 'Yerel pazarlarda esnaf denetimleri sıklaştırıldı', excerpt: 'Gıda güvenliği kapsamında semt pazarlarında yapılan denetimlerin sıklığı artırıldı.', body: ['Denetimler Sıklaştı', 'Semt pazarlarında gıda güvenliği denetimleri artırıldı.', 'Denetimlerde usulsüzlük tespit edilen işletmelere ceza uygulanıyor.'] },
  { category: 'Gündem', title: 'Şehir müzesinde yeni sergi açıldı', excerpt: 'Şehir müzesinde yerel tarihe ışık tutan yeni bir fotoğraf sergisi ziyarete açıldı.', body: ['Sergi Açıldı', 'Şehir müzesindeki yeni sergi ziyaretçilerle buluştu.', 'Sergi, önümüzdeki iki ay boyunca gezilebilecek.'] },

  // Dünya
  { category: 'Dünya', title: 'Asya-Pasifik ekonomik forumu sona erdi', excerpt: 'Forumda bölgesel ticaret iş birliğini güçlendirecek yeni adımlar üzerinde duruldu.', body: ['Forum Sona Erdi', 'Asya-Pasifik ekonomik forumu üç günün ardından sona erdi.', 'Katılımcı ülkeler, ticaret iş birliğini artırma konusunda mutabık kaldı.'] },
  { category: 'Dünya', title: 'Afrika\'da tarım teknolojileri zirvesi düzenlendi', excerpt: 'Zirvede kıtadaki tarımsal verimliliği artıracak teknoloji transferi konuşuldu.', body: ['Zirve Gerçekleşti', 'Afrika\'da düzenlenen tarım teknolojileri zirvesine çok sayıda ülke katıldı.', 'Zirvede teknoloji transferi projeleri masaya yatırıldı.'] },
  { category: 'Dünya', title: 'Kuzey Avrupa\'da yenilenebilir enerji yatırımları arttı', excerpt: 'Bölge ülkeleri, rüzgar ve güneş enerjisi kapasitesini genişletmek için yeni yatırım planları açıkladı.', body: ['Yatırımlar Artıyor', 'Kuzey Avrupa ülkelerinde yenilenebilir enerji yatırımları hız kazandı.', 'Hedef, önümüzdeki on yılda enerji ithalatını azaltmak.'] },
  { category: 'Dünya', title: 'Güney Amerika\'da orman koruma projesi genişliyor', excerpt: 'Uluslararası destekli proje kapsamında korunan orman alanı miktarı artırıldı.', body: ['Proje Genişliyor', 'Güney Amerika\'daki orman koruma projesi kapsam genişletti.', 'Proje, yerel toplulukların da desteğiyle yürütülüyor.'] },
  { category: 'Dünya', title: 'Uluslararası uzay istasyonunda yeni deney başladı', excerpt: 'Astronotlar, mikrogravite ortamında malzeme bilimi alanında yeni bir deney dizisine başladı.', body: ['Deneyler Başladı', 'Uluslararası uzay istasyonunda yeni bilimsel deneyler başlatıldı.', 'Sonuçların önümüzdeki aylarda paylaşılması bekleniyor.'] },
  { category: 'Dünya', title: 'Okyanus temizliği girişimi destek buldu', excerpt: 'Plastik atıkları temizlemeyi hedefleyen uluslararası girişim yeni bağışçılardan destek aldı.', body: ['Yeni Destek', 'Okyanus temizliği girişimi yeni finansman kaynakları buldu.', 'Girişim, önümüzdeki yıl faaliyet alanını genişletmeyi planlıyor.'] },
  { category: 'Dünya', title: 'Küresel sağlık örgütü yeni rapor yayınladı', excerpt: 'Raporda bulaşıcı olmayan hastalıklarla mücadelede küresel iş birliğinin önemine dikkat çekildi.', body: ['Yeni Rapor', 'Küresel sağlık örgütü yıllık raporunu yayınladı.', 'Rapor, önleyici sağlık politikalarına vurgu yapıyor.'] },
  { category: 'Dünya', title: 'Sınır ötesi demiryolu projesi ilerliyor', excerpt: 'İki ülkeyi birbirine bağlayacak demiryolu hattının inşaat çalışmalarında önemli bir aşamaya gelindi.', body: ['İnşaat İlerliyor', 'Sınır ötesi demiryolu projesinde inşaat çalışmaları hızla sürüyor.', 'Hattın iki yıl içinde tamamlanması öngörülüyor.'] },
  { category: 'Dünya', title: 'Dünya genelinde su tasarrufu kampanyası başladı', excerpt: 'Kampanya, kentsel su kaynaklarının sürdürülebilir kullanımına dikkat çekmeyi amaçlıyor.', body: ['Kampanya Başladı', 'Su tasarrufuna dikkat çeken küresel kampanya start aldı.', 'Kampanya kapsamında çok sayıda şehir etkinliklere ev sahipliği yapacak.'] },
  { category: 'Dünya', title: 'Uluslararası sanat bienali kapılarını açtı', excerpt: 'Bienalde dünyanın dört bir yanından sanatçıların eserleri sergileniyor.', body: ['Bienal Başladı', 'Uluslararası sanat bienali ziyaretçilerini ağırlamaya başladı.', 'Bienal, iki ay boyunca açık kalacak.'] },

  // Ekonomi
  { category: 'Ekonomi', title: 'KOBİ\'lere yeni destek paketi açıklandı', excerpt: 'Küçük ve orta ölçekli işletmelere yönelik yeni finansman desteği devreye alındı.', body: ['Yeni Destek Paketi', 'KOBİ\'lere yönelik yeni finansman desteği açıklandı.', 'Başvurular önümüzdeki hafta başlayacak.'] },
  { category: 'Ekonomi', title: 'Sanayi üretiminde aylık artış kaydedildi', excerpt: 'Sanayi üretim endeksi bir önceki aya göre belirgin bir artış gösterdi.', body: ['Üretim Arttı', 'Sanayi üretiminde aylık bazda artış kaydedildi.', 'Artışta imalat sektörünün payı öne çıktı.'] },
  { category: 'Ekonomi', title: 'Turizm gelirlerinde rekor bekleniyor', excerpt: 'Sektör temsilcileri, bu yılki turizm gelirlerinin önceki yılları geride bırakmasını bekliyor.', body: ['Rekor Beklentisi', 'Turizm sektöründe bu yıl rekor gelir bekleniyor.', 'Doluluk oranlarının geçen yıla göre arttığı belirtildi.'] },
  { category: 'Ekonomi', title: 'Yerli üretim teşvikleri genişletildi', excerpt: 'Yerli üretimi desteklemek amacıyla vergi teşviklerinin kapsamı genişletildi.', body: ['Teşvikler Genişledi', 'Yerli üretime yönelik teşvik kapsamı genişletildi.', 'Yeni teşviklerden daha fazla sektörün yararlanması bekleniyor.'] },
  { category: 'Ekonomi', title: 'Lojistik sektöründe dijitalleşme hız kazandı', excerpt: 'Firmalar, tedarik zinciri yönetiminde dijital çözümlere geçişi hızlandırdı.', body: ['Dijitalleşme Hızlandı', 'Lojistik sektöründe dijital dönüşüm hız kazandı.', 'Dijital çözümler, teslimat sürelerini kısaltmaya yardımcı oluyor.'] },
  { category: 'Ekonomi', title: 'Enerji ithalatında maliyet düşüşü gözlendi', excerpt: 'Küresel fiyatlardaki gerileme, enerji ithalat maliyetlerine olumlu yansıdı.', body: ['Maliyetler Geriledi', 'Enerji ithalat maliyetlerinde gerileme kaydedildi.', 'Uzmanlar, bu eğilimin cari açığa olumlu katkı sağlayacağını belirtiyor.'] },
  { category: 'Ekonomi', title: 'Girişimcilik fonu yeni başvuruları açtı', excerpt: 'Erken aşama girişimlere yönelik fon, yeni dönem başvurularını almaya başladı.', body: ['Başvurular Açıldı', 'Girişimcilik fonu yeni dönem başvurularını açtı.', 'Seçilen girişimlere mentorluk desteği de sağlanacak.'] },
  { category: 'Ekonomi', title: 'Perakende sektöründe istihdam arttı', excerpt: 'Yıl sonu kampanya dönemiyle birlikte perakende sektöründe istihdam rakamları yükseldi.', body: ['İstihdam Arttı', 'Perakende sektöründe istihdam rakamlarında artış gözlendi.', 'Sektör temsilcileri, artışın mevsimsel olduğunu belirtti.'] },
  { category: 'Ekonomi', title: 'Tarım ürünleri ihracatında yeni pazarlar', excerpt: 'İhracatçılar, tarım ürünleri için yeni pazarlara açılma hedeflerini açıkladı.', body: ['Yeni Pazarlar', 'Tarım ürünleri ihracatında yeni pazar arayışları hızlandı.', 'Hedef pazarlar arasında Orta Doğu ve Kuzey Afrika öne çıkıyor.'] },
  { category: 'Ekonomi', title: 'Finans sektöründe dijital dönüşüm yatırımları', excerpt: 'Bankalar, mobil bankacılık altyapısına yönelik yatırımlarını artırdı.', body: ['Yatırımlar Artıyor', 'Finans sektöründe dijital dönüşüm yatırımları hız kazandı.', 'Mobil bankacılık kullanım oranlarının arttığı kaydedildi.'] },

  // Teknoloji
  { category: 'Teknoloji', title: 'Yapay zekâ destekli tarım uygulaması geliştirildi', excerpt: 'Uygulama, çiftçilere verim tahmini ve sulama planlaması konusunda destek sağlıyor.', body: ['Yeni Uygulama', 'Yerli bir girişim, yapay zekâ destekli tarım uygulamasını tanıttı.', 'Uygulama, çiftçilere verim tahmini sunuyor.'] },
  { category: 'Teknoloji', title: '5G altyapısı kırsal bölgelere ulaşıyor', excerpt: 'Operatörler, 5G şebeke yatırımlarını kırsal bölgeleri de kapsayacak şekilde genişletiyor.', body: ['Altyapı Genişliyor', '5G altyapı yatırımları kırsal bölgelere ulaşmaya başladı.', 'Hedef, önümüzdeki iki yılda ülke genelinde kapsama sağlamak.'] },
  { category: 'Teknoloji', title: 'Yerli oyun stüdyosundan yeni proje', excerpt: 'Stüdyo, uluslararası pazara yönelik yeni bir mobil oyun projesini duyurdu.', body: ['Yeni Proje', 'Yerli oyun stüdyosu yeni projesini tanıttı.', 'Oyunun önümüzdeki yıl piyasaya sürülmesi planlanıyor.'] },
  { category: 'Teknoloji', title: 'Siber güvenlik eğitimleri yaygınlaştırılıyor', excerpt: 'Kamu kurumlarında çalışanlara yönelik siber güvenlik farkındalık eğitimleri artırıldı.', body: ['Eğitimler Yaygınlaşıyor', 'Siber güvenlik eğitimleri kamu kurumlarında yaygınlaştırılıyor.', 'Eğitimler, temel güvenlik farkındalığını artırmayı hedefliyor.'] },
  { category: 'Teknoloji', title: 'Akıllı şehir sensör ağı genişliyor', excerpt: 'Trafik ve hava kalitesi verilerini toplayan sensör ağı şehrin yeni bölgelerine yayılıyor.', body: ['Ağ Genişliyor', 'Akıllı şehir sensör ağı yeni bölgeleri kapsayacak şekilde genişletildi.', 'Veriler, trafik yönetimini iyileştirmek için kullanılacak.'] },
  { category: 'Teknoloji', title: 'Bulut bilişim yatırımları hız kazandı', excerpt: 'Şirketler, veri altyapısını bulut tabanlı çözümlere taşıma sürecini hızlandırdı.', body: ['Yatırımlar Hızlandı', 'Bulut bilişim yatırımları önemli ölçüde arttı.', 'Geçiş sürecinin maliyet avantajı sağladığı belirtiliyor.'] },
  { category: 'Teknoloji', title: 'Mobil uygulama pazarında yerli başarı', excerpt: 'Yerli bir uygulama, kategori sıralamasında uluslararası rakiplerini geride bıraktı.', body: ['Yerli Başarı', 'Yerli mobil uygulama, kategorisinde zirveye yerleşti.', 'Geliştirici ekip, kullanıcı geri bildirimlerine göre güncellemeler yapmayı sürdürüyor.'] },
  { category: 'Teknoloji', title: 'Robotik otomasyon fabrikalarda artıyor', excerpt: 'Üretim tesisleri, verimliliği artırmak için robotik otomasyon yatırımlarını hızlandırdı.', body: ['Otomasyon Artıyor', 'Fabrikalarda robotik otomasyon kullanımı yaygınlaşıyor.', 'Yatırımların üretim hızını artırdığı gözlemleniyor.'] },
  { category: 'Teknoloji', title: 'Veri merkezi yatırımı tamamlandı', excerpt: 'Yeni veri merkezi, artan bulut hizmeti talebini karşılamak üzere devreye alındı.', body: ['Yatırım Tamamlandı', 'Yeni veri merkezi hizmete açıldı.', 'Merkez, enerji verimliliği standartlarına uygun olarak inşa edildi.'] },
  { category: 'Teknoloji', title: 'Açık kaynak donanım projesi tanıtıldı', excerpt: 'Proje, geliştiricilere düşük maliyetli ve özelleştirilebilir donanım çözümleri sunmayı hedefliyor.', body: ['Proje Tanıtıldı', 'Açık kaynak donanım projesi geliştirici topluluğuyla buluştu.', 'Proje ekibi, katkı sağlamak isteyenlere kapılarını açık tutuyor.'] },

  // Spor
  { category: 'Spor', title: 'Atletizmde yeni şampiyon belli oldu', excerpt: '100 metre finalinde genç sporcu beklenmedik bir performansla şampiyon oldu.', body: ['Yeni Şampiyon', 'Atletizm şampiyonasında 100 metre finalinde yeni şampiyon belli oldu.', 'Sporcu, başarısını uzun süredir devam eden çalışmasına bağladı.'] },
  { category: 'Spor', title: 'Basketbolda final heyecanı sürüyor', excerpt: 'Play-off serisinde iki takım da birer galibiyetle final yarışında eşitliği koruyor.', body: ['Final Serisi', 'Basketbol liginde final serisi heyecanla devam ediyor.', 'Üçüncü maçın bu hafta sonu oynanması bekleniyor.'] },
  { category: 'Spor', title: 'Yüzme federasyonundan gençlik programı', excerpt: 'Federasyon, yetenekli genç yüzücüleri desteklemek için yeni bir burs programı başlattı.', body: ['Yeni Program', 'Yüzme federasyonu gençlik destek programını duyurdu.', 'Program kapsamında seçilen sporculara antrenman desteği sağlanacak.'] },
  { category: 'Spor', title: 'Tenis turnuvasında sürpriz sonuçlar', excerpt: 'Turnuvanın ilk turunda favoriler elenirken sürpriz sonuçlar gündeme oturdu.', body: ['Sürpriz Sonuçlar', 'Tenis turnuvasının ilk turunda sürpriz eleme sonuçları yaşandı.', 'Turnuvanın geri kalanının da heyecanlı geçmesi bekleniyor.'] },
  { category: 'Spor', title: 'Masa tenisinde milli başarı', excerpt: 'Milli sporcu, uluslararası turnuvada çeyrek finale yükselerek başarılı bir performans sergiledi.', body: ['Milli Başarı', 'Masa tenisinde milli sporcu çeyrek finale yükseldi.', 'Sporcu, bir sonraki tura hazırlıklarını sürdürüyor.'] },
  { category: 'Spor', title: 'Judo milli takımı kampa girdi', excerpt: 'Milli takım, yaklaşan Avrupa şampiyonası öncesi hazırlık kampına başladı.', body: ['Kamp Başladı', 'Judo milli takımı hazırlık kampına girdi.', 'Teknik heyet, kamp programını sıkı tutacaklarını belirtti.'] },
  { category: 'Spor', title: 'Bisiklet turunda yeni parkur açıklandı', excerpt: 'Bu yılki bisiklet turunun parkuru, daha zorlu dağ etaplarını içerecek şekilde güncellendi.', body: ['Yeni Parkur', 'Bisiklet turunun bu yılki parkuru açıklandı.', 'Parkurda daha zorlu dağ etaplarına yer verildi.'] },
  { category: 'Spor', title: 'Güreşte Avrupa şampiyonası hazırlıkları', excerpt: 'Milli güreşçiler, Avrupa şampiyonası öncesi son hazırlık etaplarına girdi.', body: ['Hazırlıklar Sürüyor', 'Güreş milli takımı Avrupa şampiyonası hazırlıklarını sürdürüyor.', 'Kafilenin önümüzdeki hafta yola çıkması planlanıyor.'] },
  { category: 'Spor', title: 'Engelli sporcular için yeni tesis açıldı', excerpt: 'Şehirdeki yeni spor tesisi, engelli sporcuların antrenman ihtiyaçlarına özel olarak tasarlandı.', body: ['Yeni Tesis', 'Engelli sporculara yönelik yeni spor tesisi açıldı.', 'Tesis, çeşitli branşlarda antrenman imkânı sunuyor.'] },
  { category: 'Spor', title: 'Amatör spor kulüplerine destek paketi', excerpt: 'Gençlik ve spor kurumu, amatör kulüplere yönelik yeni bir malzeme ve altyapı destek paketi açıkladı.', body: ['Destek Paketi', 'Amatör spor kulüplerine yönelik yeni destek paketi açıklandı.', 'Destek, malzeme ve saha bakımı giderlerini kapsayacak.'] },
];

async function run() {
  const csrf = await request('/csrf-token');
  csrfToken = csrf.token;
  await request('/auth/login', { method: 'POST', body: JSON.stringify({ email: EMAIL, password: PASSWORD }) });
  console.log(`Logged in. Publishing ${ARTICLES.length} articles...`);

  for (const article of ARTICLES) {
    const slug = slugOf(article.title);
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
