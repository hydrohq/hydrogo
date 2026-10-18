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

export function getLiveMarkets(): MarketItem[] {
  return [
    { code: 'USD/TRY', name: 'Dolar', buying: '34.22', selling: '34.28', changeRate: 0.18 },
    { code: 'EUR/TRY', name: 'Euro', buying: '37.05', selling: '37.15', changeRate: -0.24 },
    { code: 'GA', name: 'Gram Altın', buying: '2.980', selling: '3.015', changeRate: 0.65 },
    { code: 'BIST100', name: 'Borsa İst.', buying: '9.145', selling: '9.150', changeRate: 1.12 },
    { code: 'BTC/USD', name: 'Bitcoin', buying: '67.450', selling: '67.500', changeRate: 2.40 },
  ];
}

export function getPrayerTimes(city: string = 'İstanbul'): PrayerTimes {
  return {
    city,
    fajr: '05:42',
    sunrise: '07:08',
    dhuhr: '13:01',
    asr: '16:04',
    maghrib: '18:42',
    isha: '20:02',
    nextPrayer: 'İkindi',
    remainingTime: '01:24',
  };
}

export function getWeather(city: string = 'İstanbul'): WeatherInfo {
  return {
    city,
    temp: 21,
    condition: 'Parçalı Bulutlu',
    icon: '⛅',
  };
}

export function getDutyPharmacies(city: string = 'İstanbul'): Pharmacy[] {
  return [
    {
      id: '1',
      name: 'Merkez Şifa Eczanesi',
      district: 'Kadıköy',
      city,
      address: 'Caferağa Mah. Moda Cad. No: 42/A',
      phone: '0 (216) 345 67 89',
      note: 'Moda Tramvay Durağı Karşısı',
    },
    {
      id: '2',
      name: 'Boğaziçi Eczanesi',
      district: 'Beşiktaş',
      city,
      address: 'Sinanpaşa Mah. Şair Nedim Cad. No: 18',
      phone: '0 (212) 234 56 78',
      note: 'Ihlamur Kasrı Civarı',
    },
    {
      id: '3',
      name: 'Çankaya Hayat Eczanesi',
      district: 'Çankaya',
      city: 'Ankara',
      address: 'Tunalı Hilmi Cad. No: 95',
      phone: '0 (312) 428 10 20',
      note: 'Kuğulu Park Yanı',
    },
    {
      id: '4',
      name: 'Kordon Eczanesi',
      district: 'Konak',
      city: 'İzmir',
      address: 'Alsancak Mah. Atatürk Cad. No: 140',
      phone: '0 (232) 464 12 34',
      note: 'Gündoğdu Meydanı Yakını',
    },
  ];
}
