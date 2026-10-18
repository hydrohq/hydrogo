export interface MarketItem {
  code: string;
  name: string;
  buying: string;
  selling: string;
  changeRate: number; // e.g. 0.45 for +0.45%
}

export interface PrayerTimes {
  city: string;
  fajr: string; // İmsak
  sunrise: string; // Güneş
  dhuhr: string; // Öğle
  asr: string; // İkindi
  maghrib: string; // Akşam
  isha: string; // Yatsı
  nextPrayer: string;
  remainingTime: string;
}

export interface WeatherInfo {
  city: string;
  temp: number;
  condition: string;
  icon: string;
}

export interface Pharmacy {
  id: string;
  name: string;
  district: string;
  city: string;
  address: string;
  phone: string;
  note?: string;
}

// All 81 provinces, for city pickers — only a handful have curated demo
// data below (prayer times, weather, duty pharmacies); the rest fall back
// to İstanbul's, same as any unrecognized city already does.
export const TURKISH_PROVINCES = [
  'Adana', 'Adıyaman', 'Afyonkarahisar', 'Ağrı', 'Amasya', 'Ankara', 'Antalya', 'Artvin',
  'Aydın', 'Balıkesir', 'Bilecik', 'Bingöl', 'Bitlis', 'Bolu', 'Burdur', 'Bursa',
  'Çanakkale', 'Çankırı', 'Çorum', 'Denizli', 'Diyarbakır', 'Edirne', 'Elazığ', 'Erzincan',
  'Erzurum', 'Eskişehir', 'Gaziantep', 'Giresun', 'Gümüşhane', 'Hakkâri', 'Hatay', 'Isparta',
  'Mersin', 'İstanbul', 'İzmir', 'Kars', 'Kastamonu', 'Kayseri', 'Kırklareli', 'Kırşehir',
  'Kocaeli', 'Konya', 'Kütahya', 'Malatya', 'Manisa', 'Kahramanmaraş', 'Mardin', 'Muğla',
  'Muş', 'Nevşehir', 'Niğde', 'Ordu', 'Rize', 'Sakarya', 'Samsun', 'Siirt',
  'Sinop', 'Sivas', 'Tekirdağ', 'Tokat', 'Trabzon', 'Tunceli', 'Şanlıurfa', 'Uşak',
  'Van', 'Yozgat', 'Zonguldak', 'Aksaray', 'Bayburt', 'Karaman', 'Kırıkkale', 'Batman',
  'Şırnak', 'Bartın', 'Ardahan', 'Iğdır', 'Yalova', 'Karabük', 'Kilis', 'Osmaniye', 'Düzce',
].sort((a, b) => a.localeCompare(b, 'tr'));

export function getLiveMarkets(): MarketItem[] {
  return [
    { code: 'USD/TRY', name: 'Dolar', buying: '34.22', selling: '34.28', changeRate: 0.18 },
    { code: 'EUR/TRY', name: 'Euro', buying: '37.05', selling: '37.15', changeRate: -0.24 },
    { code: 'GA', name: 'Gram Altın', buying: '2.980', selling: '3.015', changeRate: 0.65 },
    { code: 'BIST100', name: 'Borsa İst.', buying: '9.145', selling: '9.150', changeRate: 1.12 },
    { code: 'BTC/USD', name: 'Bitcoin', buying: '67.450', selling: '67.500', changeRate: 2.40 },
  ];
}

export function normalizeCityName(name: string = ''): string {
  const n = (name || '').toLocaleLowerCase('tr-TR').trim();
  if (n.includes('istan') || n.includes('ıstan')) return 'İstanbul';
  if (n.includes('ankar')) return 'Ankara';
  if (n.includes('izm') || n.includes('ızm')) return 'İzmir';
  if (n.includes('burs')) return 'Bursa';
  if (n.includes('antal')) return 'Antalya';
  if (n.includes('adan')) return 'Adana';
  if (n.includes('kony')) return 'Konya';
  if (n.includes('trabz')) return 'Trabzon';
  return 'İstanbul';
}

const PRAYER_DATABASE: Record<string, PrayerTimes> = {
  İstanbul: {
    city: 'İstanbul',
    fajr: '05:42',
    sunrise: '07:08',
    dhuhr: '13:01',
    asr: '16:04',
    maghrib: '18:42',
    isha: '20:02',
    nextPrayer: 'İkindi',
    remainingTime: '01:24',
  },
  Ankara: {
    city: 'Ankara',
    fajr: '05:28',
    sunrise: '06:54',
    dhuhr: '12:47',
    asr: '15:50',
    maghrib: '18:29',
    isha: '19:49',
    nextPrayer: 'İkindi',
    remainingTime: '01:10',
  },
  İzmir: {
    city: 'İzmir',
    fajr: '05:49',
    sunrise: '07:13',
    dhuhr: '13:08',
    asr: '16:13',
    maghrib: '18:52',
    isha: '20:09',
    nextPrayer: 'İkindi',
    remainingTime: '01:33',
  },
  Bursa: {
    city: 'Bursa',
    fajr: '05:41',
    sunrise: '07:07',
    dhuhr: '13:00',
    asr: '16:04',
    maghrib: '18:42',
    isha: '20:01',
    nextPrayer: 'İkindi',
    remainingTime: '01:24',
  },
  Antalya: {
    city: 'Antalya',
    fajr: '05:35',
    sunrise: '06:58',
    dhuhr: '12:54',
    asr: '16:01',
    maghrib: '18:38',
    isha: '19:55',
    nextPrayer: 'İkindi',
    remainingTime: '01:21',
  },
  Adana: {
    city: 'Adana',
    fajr: '05:22',
    sunrise: '06:44',
    dhuhr: '12:41',
    asr: '15:48',
    maghrib: '18:25',
    isha: '19:42',
    nextPrayer: 'İkindi',
    remainingTime: '01:08',
  },
  Konya: {
    city: 'Konya',
    fajr: '05:30',
    sunrise: '06:53',
    dhuhr: '12:50',
    asr: '15:56',
    maghrib: '18:34',
    isha: '19:50',
    nextPrayer: 'İkindi',
    remainingTime: '01:16',
  },
  Trabzon: {
    city: 'Trabzon',
    fajr: '05:04',
    sunrise: '06:33',
    dhuhr: '12:25',
    asr: '15:25',
    maghrib: '18:04',
    isha: '19:27',
    nextPrayer: 'İkindi',
    remainingTime: '00:45',
  },
};

const WEATHER_DATABASE: Record<string, WeatherInfo> = {
  İstanbul: { city: 'İstanbul', temp: 21, condition: 'Parçalı Bulutlu', icon: '⛅' },
  Ankara: { city: 'Ankara', temp: 17, condition: 'Açık, Güneşli', icon: '☀️' },
  İzmir: { city: 'İzmir', temp: 24, condition: 'Açık, Ilık', icon: '🌤️' },
  Bursa: { city: 'Bursa', temp: 19, condition: 'Az Bulutlu', icon: '⛅' },
  Antalya: { city: 'Antalya', temp: 27, condition: 'Güneşli & Sıcak', icon: '☀️' },
  Adana: { city: 'Adana', temp: 28, condition: 'Açık & Sıcak', icon: '☀️' },
  Konya: { city: 'Konya', temp: 16, condition: 'Hafif Rüzgarlı', icon: '💨' },
  Trabzon: { city: 'Trabzon', temp: 18, condition: 'Sağanak Yağışlı', icon: '🌧️' },
};

const PHARMACY_DATABASE: Record<string, Pharmacy[]> = {
  İstanbul: [
    {
      id: 'ist-1',
      name: 'Moda Sahil Eczanesi',
      district: 'Kadıköy',
      city: 'İstanbul',
      address: 'Caferağa Mah. Moda Cad. No: 42/A',
      phone: '0 (216) 345 67 89',
      note: 'Moda Tramvay Durağı Karşısı',
    },
    {
      id: 'ist-2',
      name: 'Ihlamur Eczanesi',
      district: 'Beşiktaş',
      city: 'İstanbul',
      address: 'Sinanpaşa Mah. Şair Nedim Cad. No: 18',
      phone: '0 (212) 234 56 78',
      note: 'Beşiktaş Çarşı İçi, Ihlamur Kasrı Civarı',
    },
    {
      id: 'ist-3',
      name: 'Nişantaşı Hayat Eczanesi',
      district: 'Şişli',
      city: 'İstanbul',
      address: 'Teşvikiye Mah. Vali Konağı Cad. No: 82',
      phone: '0 (212) 296 44 22',
      note: 'Amerikan Hastanesi Yakını',
    },
    {
      id: 'ist-4',
      name: 'Mihrimah Eczanesi',
      district: 'Üsküdar',
      city: 'İstanbul',
      address: 'Mimar Sinan Mah. Hakimiyeti Milliye Cad. No: 12',
      phone: '0 (216) 553 11 00',
      note: 'Üsküdar Meydanı, Vapur İskelesi Yanı',
    },
    {
      id: 'ist-5',
      name: 'Çapa Şifa Eczanesi',
      district: 'Fatih',
      city: 'İstanbul',
      address: 'Şehremini Mah. Turgut Özal Millet Cad. No: 104',
      phone: '0 (212) 585 90 90',
      note: 'Çapa Tıp Fakültesi Karşısı',
    },
  ],
  Ankara: [
    {
      id: 'ank-1',
      name: 'Tunalı Eczanesi',
      district: 'Çankaya',
      city: 'Ankara',
      address: 'Tunalı Hilmi Cad. No: 95/B',
      phone: '0 (312) 428 10 20',
      note: 'Kuğulu Park Yanı',
    },
    {
      id: 'ank-2',
      name: 'Batıkent Şifa Eczanesi',
      district: 'Yenimahalle',
      city: 'Ankara',
      address: 'Kentkoop Mah. 1864 Cad. No: 8',
      phone: '0 (312) 250 88 44',
      note: 'Batıkent Metro Çıkışı',
    },
    {
      id: 'ank-3',
      name: 'Etlik Hayat Eczanesi',
      district: 'Keçiören',
      city: 'Ankara',
      address: 'Etlik Mah. Kıvrımlı Cad. No: 41',
      phone: '0 (312) 322 15 50',
      note: 'Etlik Şehir Hastanesi Civarı',
    },
    {
      id: 'ank-4',
      name: 'Zafer Eczanesi',
      district: 'Altındağ',
      city: 'Ankara',
      address: 'Anafartalar Mah. Çıkrıkçılar Yokuşu No: 14',
      phone: '0 (312) 310 40 50',
      note: 'Ulus Heykel Meydanı Yakını',
    },
  ],
  İzmir: [
    {
      id: 'izm-1',
      name: 'Kordon Eczanesi',
      district: 'Konak',
      city: 'İzmir',
      address: 'Alsancak Mah. Atatürk Cad. No: 140/A',
      phone: '0 (232) 464 12 34',
      note: 'Gündoğdu Meydanı Yakını',
    },
    {
      id: 'izm-2',
      name: 'Bostanlı Sahil Eczanesi',
      district: 'Karşıyaka',
      city: 'İzmir',
      address: 'Cemal Gürsel Cad. No: 112',
      phone: '0 (232) 362 55 60',
      note: 'Bostanlı Vapur İskelesi Karşısı',
    },
    {
      id: 'izm-3',
      name: 'Ege Üniversitesi Eczanesi',
      district: 'Bornova',
      city: 'İzmir',
      address: 'Kazımdirik Mah. Fevzi Çakmak Cad. No: 19',
      phone: '0 (232) 388 77 99',
      note: 'Küçükpark Meydanı',
    },
    {
      id: 'izm-4',
      name: 'Şirinyer Eczanesi',
      district: 'Buca',
      city: 'İzmir',
      address: 'Hürriyet Mah. Menderes Cad. No: 88',
      phone: '0 (232) 448 30 20',
      note: 'İzban Şirinyer İstasyonu Civarı',
    },
  ],
  Bursa: [
    {
      id: 'bur-1',
      name: 'Heykel Merkez Eczanesi',
      district: 'Osmangazi',
      city: 'Bursa',
      address: 'Atatürk Cad. No: 56',
      phone: '0 (224) 221 40 50',
      note: 'Tarihi Valilik Binası Karşısı',
    },
    {
      id: 'bur-2',
      name: 'Özlüce Deva Eczanesi',
      district: 'Nilüfer',
      city: 'Bursa',
      address: '100. Yıl Mah. Uğur Mumcu Bulvarı No: 28',
      phone: '0 (224) 413 80 90',
      note: 'Özlüce Metro Durağı Çıkışı',
    },
    {
      id: 'bur-3',
      name: 'Yeşil Eczanesi',
      district: 'Yıldırım',
      city: 'Bursa',
      address: 'Yeşil Mah. 1. Yeşil Cad. No: 16',
      phone: '0 (224) 327 45 60',
      note: 'Yeşil Türbe Civarı',
    },
    {
      id: 'bur-4',
      name: 'Mudanya İskele Eczanesi',
      district: 'Mudanya',
      city: 'Bursa',
      address: 'Halitpaşa Mah. 12 Eylül Cad. No: 34',
      phone: '0 (224) 544 12 22',
      note: 'BUDO Deniz Otobüsü İskelesi Yanı',
    },
  ],
  Antalya: [
    {
      id: 'ant-1',
      name: 'Lara Sahil Eczanesi',
      district: 'Muratpaşa',
      city: 'Antalya',
      address: 'Şirinyalı Mah. İsmet Gökşen Cad. No: 48',
      phone: '0 (242) 316 20 30',
      note: 'Dedeman Kavşağı Yakını',
    },
    {
      id: 'ant-2',
      name: 'Altınkum Eczanesi',
      district: 'Konyaaltı',
      city: 'Antalya',
      address: 'Gürsu Mah. Akdeniz Bulvarı No: 102',
      phone: '0 (242) 229 40 50',
      note: 'Konyaaltı Plaj Yolu Karşısı',
    },
    {
      id: 'ant-3',
      name: 'Dokuma Şifa Eczanesi',
      district: 'Kepez',
      city: 'Antalya',
      address: 'Fabrikalar Mah. Fikri Erten Cad. No: 21',
      phone: '0 (242) 334 50 60',
      note: 'Özdilek Park AVM Civarı',
    },
    {
      id: 'ant-4',
      name: 'Damlataş Eczanesi',
      district: 'Alanya',
      city: 'Antalya',
      address: 'Saray Mah. Damlataş Cad. No: 35',
      phone: '0 (242) 513 18 20',
      note: 'Alanya Teleferik Yanı',
    },
  ],
  Adana: [
    {
      id: 'ada-1',
      name: 'Ziyapaşa Eczanesi',
      district: 'Seyhan',
      city: 'Adana',
      address: 'Kurtuluş Mah. Ziyapaşa Bulvarı No: 44/A',
      phone: '0 (322) 458 10 20',
      note: 'Atatürk Parkı Karşısı',
    },
    {
      id: 'ada-2',
      name: 'Barajyolu Hayat Eczanesi',
      district: 'Çukurova',
      city: 'Adana',
      address: 'Beyazevler Mah. Barajyolu Cad. No: 78',
      phone: '0 (322) 235 60 70',
      note: 'Duygu Kafe Civarı',
    },
    {
      id: 'ada-3',
      name: 'Köprülü Eczanesi',
      district: 'Yüreğir',
      city: 'Adana',
      address: 'Kışla Mah. Mustafa Kemal Paşa Bulvarı No: 15',
      phone: '0 (322) 322 40 50',
      note: 'Yüreğir Devlet Hastanesi Yanı',
    },
  ],
  Konya: [
    {
      id: 'kny-1',
      name: 'Nalçacı Eczanesi',
      district: 'Selçuklu',
      city: 'Konya',
      address: 'Nişantaş Mah. Nalçacı Cad. No: 62',
      phone: '0 (332) 236 40 50',
      note: 'Kule Site AVM Civarı',
    },
    {
      id: 'kny-2',
      name: 'Zafer Şifa Eczanesi',
      district: 'Meram',
      city: 'Konya',
      address: 'Şükran Mah. Abdülaziz Cad. No: 18',
      phone: '0 (332) 351 12 30',
      note: 'Zafer Yaya Meydanı',
    },
    {
      id: 'kny-3',
      name: 'Mevlana Deva Eczanesi',
      district: 'Karatay',
      city: 'Konya',
      address: 'Aziziye Mah. Mevlana Cad. No: 34',
      phone: '0 (332) 350 20 40',
      note: 'Mevlana Müzesi Karşısı',
    },
  ],
  Trabzon: [
    {
      id: 'tbz-1',
      name: 'Meydan Eczanesi',
      district: 'Ortahisar',
      city: 'Trabzon',
      address: 'Kemerkaya Mah. Kahramanmaraş Cad. No: 45',
      phone: '0 (462) 321 40 50',
      note: 'Atatürk Parkı Meydan Yanı',
    },
    {
      id: 'tbz-2',
      name: 'Sahil Eczanesi',
      district: 'Akçaabat',
      city: 'Trabzon',
      address: 'Dürbinar Mah. İnönü Cad. No: 22',
      phone: '0 (462) 228 15 30',
      note: 'Sahil Parkı Karşısı',
    },
    {
      id: 'tbz-3',
      name: 'Kaşüstü Hayat Eczanesi',
      district: 'Yomra',
      city: 'Trabzon',
      address: 'Kaşüstü Mah. Devlet Karayolu No: 104',
      phone: '0 (462) 344 20 60',
      note: 'Kanuni Eğitim ve Araştırma Hastanesi Civarı',
    },
  ],
};

export function getPrayerTimes(city: string = 'İstanbul'): PrayerTimes {
  const norm = normalizeCityName(city);
  return PRAYER_DATABASE[norm] || PRAYER_DATABASE['İstanbul'];
}

export function getWeather(city: string = 'İstanbul'): WeatherInfo {
  const norm = normalizeCityName(city);
  return WEATHER_DATABASE[norm] || WEATHER_DATABASE['İstanbul'];
}

export function getDutyPharmacies(city: string = 'İstanbul'): Pharmacy[] {
  const norm = normalizeCityName(city);
  return PHARMACY_DATABASE[norm] || PHARMACY_DATABASE['İstanbul'];
}
