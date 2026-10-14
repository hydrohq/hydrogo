(function(){
  const translations = {
    'Geliştirici & Entegrasyon Konsolu':'Developer & Integration Console',
    'Geliştirici &amp; Entegrasyon Konsolu':'Developer &amp; Integration Console',
    'Geliştirici sekmeleri':'Developer tabs', 'API & Entegrasyon':'API & Integration',
    'Sistem Kaynakları':'System Resources', 'Yönetim Araçları':'Administration Tools',
    'Güvenlik & Yetkiler':'Security & Permissions', 'Güvenlik &amp; Yetkiler':'Security &amp; Permissions',
    'Sistem kaynakları':'System resources', 'Process belleği':'Process memory', 'HTTP ve trafik':'HTTP & traffic',
    'Veritabanı havuzu':'Database pool', 'Arka plan işleri':'Background jobs', 'Heap kullanımı':'Heap usage',
    'İstek / gecikme':'Requests / latency', 'Ortalama':'Average', 'Başlangıç:':'Started:',
    'Toplam istek':'Total requests', 'Aktif istek':'Active requests', 'Sayfa görüntüleme':'Page views',
    'Benzersiz ziyaretçi':'Unique visitors', 'Bekleyen / lease':'Pending / leased',
    'Dead-letter':'Dead-letter', 'Worker paralelliği':'Worker concurrency', 'İşlenen / başarısız':'Processed / failed',
    'Hatalı İstekler':'Failed requests', 'Başarı Oranı':'Success rate', 'Hata & Güvenlik Blokajı':'Errors & security blocks',
    'En Çok Hata Alan Rotalar':'Routes with most errors', 'Hata sayısına göre sıralı':'Sorted by error count',
    'Hata Sayısı':'Error count', 'Hata kaydedilen rota bulunmuyor (Sistem kusursuz çalışıyor).':'No routes with errors recorded (the system is healthy).',
    'Göster':'Show', 'Tümü':'All', 'Görüntülenme Olanlar':'With views', 'Hata Alanlar':'With errors',
    'Zaman Aralığı:':'Time range:', 'Son 7 Gün':'Last 7 days', 'Son 30 Gün':'Last 30 days',
    'Tüm Zamanlar':'All time', 'CSV Dışa Aktar':'Export CSV', 'JSON Dışa Aktar':'Export JSON',
    'Canlı Proses':'Live process', 'Veri Zamanı:':'Data time:', 'Bilgi':'Information',
    'Medya kütüphanesi':'Media library', 'Görsel seç':'Select image', 'Kapat':'Close',
    'Yalnız hazır görseller listelenir. Yeni görseli Medya alanından yükleyin.':'Only ready images are listed. Upload new images from Media.',
    'Seçilebilecek hazır görsel yok.':'No ready images available.', 'Medya kütüphanesi açılamadı.':'Media library could not be opened.',
    'Medyadan seçin':'Choose from media', 'Kütüphaneden seç':'Choose from library', 'Madde ekle':'Add item',
    'İçerik dili':'Content language', 'Seçin':'Select', 'Başlık boyutu':'Heading size', 'Normal Metin':'Normal text',
    'Başlık 1':'Heading 1', 'Başlık 2':'Heading 2', 'Başlık 3':'Heading 3', 'Başlık 4':'Heading 4',
    'Kalın':'Bold', 'İtalik':'Italic', 'Altı çizili':'Underline', 'Üstü çizili':'Strike-through',
    'Bağlantı ekle':'Add link', 'Görsel ekle':'Add image', 'Alıntı':'Blockquote', 'Kod bloğu':'Code block',
    'Numaralı liste':'Ordered list', 'Madde listesi':'Bulleted list', 'Biçimlendirmeyi temizle':'Clear formatting',
    'İşlem başarılı!':'Operation completed successfully!', 'İşlem başarısız.':'Operation failed.',
    'İşlem tamamlanamadı.':'The operation could not be completed.', 'Güvenlik doğrulaması başarısız.':'Security verification failed.',
    'Lütfen sayfayı yenileyin.':'Please refresh the page.', 'Hata: İstek tamamlanamadı.':'Error: The request could not be completed.',
    'Tekrar Dene':'Try again', 'Sistem kusursuz çalışıyor':'The system is healthy',
    'İşlem Başarılı':'Operation successful', 'Hata':'Error', 'Yükleme Başarılı':'Upload successful',
    'Yükleme Başarısız':'Upload failed', 'Davet Gönderildi':'Invitation sent', 'İşlem sırasında bir hata oluştu.':'An error occurred while processing the request.',
    'Medya başarıyla yüklendi ve işleme alındı.':'Media uploaded and queued for processing.', 'Medya yüklenirken hata oluştu.':'Media upload failed.',
    'Davet e-postası başarıyla gönderildi.':'Invitation email sent successfully.', 'Davet gönderilemedi.':'The invitation could not be sent.',
    'Sayfalar':'Pages', 'Haberler':'News', 'Ürünler':'Products', 'Varyantlar':'Variants', 'Siparişler':'Orders', 'Üyeler':'Members',
    'Başlık':'Title', 'İçerik':'Content', 'Durum':'Status', 'Dil':'Language', 'Özet':'Summary', 'Kategori':'Category',
    'Yazar adı':'Author name', 'Etiketler':'Tags', 'Öne çıkan':'Featured', 'Açıklama':'Description', 'Ürün adı':'Product name',
    'Fiyat (kuruş)':'Price (cents)', 'Para birimi':'Currency', 'Stok':'Stock', 'Kaydet':'Save', 'Güncelle':'Update',
    'Oluştur':'Create', 'Düzenle':'Edit', 'Sil':'Delete', 'Yayınla':'Publish', 'İptal':'Cancel', 'Yeni':'New',
    'Taslak':'Draft', 'Yayınlandı':'Published', 'Aktif':'Active', 'Pasif':'Inactive'
  };
  const translate = root => {
    const walker = document.createTreeWalker(root || document.body, NodeFilter.SHOW_TEXT);
    const nodes=[]; while(walker.nextNode()) nodes.push(walker.currentNode);
    nodes.forEach(node=>{const key=node.nodeValue.trim(); if(translations[key]) node.nodeValue=node.nodeValue.replace(key, translations[key]);});
    (root || document).querySelectorAll?.('[aria-label],[title],[placeholder]').forEach(el=>['aria-label','title','placeholder'].forEach(attr=>{const key=el.getAttribute(attr); if(translations[key]) el.setAttribute(attr, translations[key]);}));
  };
  const start=()=>{translate(document.body); new MutationObserver(records=>records.forEach(record=>record.addedNodes.forEach(node=>{if(node.nodeType===1)translate(node);}))).observe(document.body,{childList:true,subtree:true});};
  if(document.readyState==='loading') document.addEventListener('DOMContentLoaded',start); else start();
})();
