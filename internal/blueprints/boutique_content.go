package blueprints

import (
	"encoding/json"
	"strings"
)

// NicheEditorial carries full rich storytelling, craft pillars, testimonials, and FAQs for a boutique.
type NicheEditorial struct {
	HeroBadge      string                 `json:"hero_badge"`
	HeroTitle      string                 `json:"hero_title"`
	HeroSubtitle   string                 `json:"hero_subtitle"`
	HeroImage      string                 `json:"hero_image"`
	StoryBadge     string                 `json:"story_badge"`
	StoryTitle     string                 `json:"story_title"`
	StoryText      string                 `json:"story_text"`
	StoryQuote     string                 `json:"story_quote"`
	StoryImage     string                 `json:"story_image"`
	PillarsTitle   string                 `json:"pillars_title"`
	Pillars        []EditorialPillar      `json:"pillars"`
	Testimonials   []EditorialTestimonial `json:"testimonials"`
	FAQs           []EditorialFAQ         `json:"faqs"`
	AtelierAddress string                 `json:"atelier_address"`
	AtelierHours   string                 `json:"atelier_hours"`
	Instagram      string                 `json:"instagram"`
}

// EditorialPillar represents a single craft/quality guarantee card.
type EditorialPillar struct {
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

// EditorialTestimonial represents authentic customer social proof.
type EditorialTestimonial struct {
	Customer string `json:"customer"`
	Location string `json:"location"`
	Quote    string `json:"quote"`
	Item     string `json:"item"`
}

// EditorialFAQ represents an accordion item.
type EditorialFAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// GetNicheEditorialContent inspects the store name, prompt, and catalog to derive deep editorial content.
func GetNicheEditorialContent(brandName, prompt string) NicheEditorial {
	haystack := strings.ToLower(brandName + " " + prompt)

	isCeramic := strings.Contains(haystack, "seramik") || strings.Contains(haystack, "çömlek") ||
		strings.Contains(haystack, "fincan") || strings.Contains(haystack, "tabak") ||
		strings.Contains(haystack, "vazo") || strings.Contains(haystack, "kupa") ||
		strings.Contains(haystack, "clay") || strings.Contains(haystack, "ceramic")

	isLeather := strings.Contains(haystack, "deri") || strings.Contains(haystack, "cüzdan") ||
		strings.Contains(haystack, "çanta") || strings.Contains(haystack, "kemer") ||
		strings.Contains(haystack, "saraç") || strings.Contains(haystack, "leather")

	if isCeramic {
		return NicheEditorial{
			HeroBadge:    "2026 GÜZ KOLEKSİYONU · EL TORNASI ATÖLYE SERİSİ",
			HeroTitle:    "Topraktan Sofraya, Sabırla Şekillenen Formlar",
			HeroSubtitle: "Doğal killer ve kurşunsuz sırlarla atölyemizde tek tek el tornasında çekilen, 1240 derecede pişirilmiş zamansız stoneware parçalar.",
			HeroImage:    "https://images.unsplash.com/photo-1565193566173-7a0ee3dbe261?auto=format&fit=crop&w=1200&q=80",
			StoryBadge:   "ZANAAT VE SABIR",
			StoryTitle:   "Çamurun 14 Günlük Dönüşüm Hikayesi",
			StoryText:    "Atölyemizde hiçbir nesne seri üretimin aceleciliğiyle üretilmez. Doğal çamurun el tornasında çekilmesi, yavaşça kuruması, 950 derecede ilk bisküvi fırınlaması, doğal mineral sırlarla elde sırlanması ve son olarak 1240 derecelik nihai fırın yolculuğu... Masanıza gelen her tabak ve kupa, iki haftalık sabırlı bir zanaat emeğini barındırır.",
			StoryQuote:   "Kusursuz simetri yerine, doğallığın ve ustanın el izinin sıcaklığına inanıyoruz.",
			StoryImage:   "https://images.unsplash.com/photo-1576020799627-aeac76d580dc?auto=format&fit=crop&w=1000&q=80",
			PillarsTitle: "Atölye Sözümüz & Zanaat İlkelerimiz",
			Pillars: []EditorialPillar{
				{
					Title: "1240°C Stoneware Mukavemeti",
					Desc:  "Yüksek ısı fırınlaması sayesinde porselen sertliğinde, bulaşık makinesi ve fırına tam dayanıklı.",
				},
				{
					Title: "Kurşunsuz Doğal Sırlar",
					Desc:  "Gıda ile temasa %100 uygun, kimyasal içermeyen mineral bazlı doğal atölye sırları.",
				},
				{
					Title: "%100 Kırılma Değişim Garantisi",
					Desc:  "Çift katlı mukavva ve petek korumalı ambalaj. Olası kargo hasarında anında koşulsuz yenisi gönderilir.",
				},
				{
					Title: "Tekil El Zanaatı İmzası",
					Desc:  "Seri üretim kalıpları yerine her parça elde şekillenir; her fincan kendine has küçük dokunuşlar taşır.",
				},
			},
			Testimonials: []EditorialTestimonial{
				{
					Customer: "Ceren A.",
					Location: "Kadıköy / İstanbul",
					Quote:    "Espresso fincanları elime ulaştığında dokusuna hayran kaldım. Masif ahşap altlıkları ve mat sırın hissiyatı mükemmel.",
					Item:     "Espresso Fincan Seti",
				},
				{
					Customer: "Murat & Selin K.",
					Location: "Karşıyaka / İzmir",
					Quote:    "Salata kasesi soframızın baş tacı oldu. Ağır, tok ve gerçek bir sanat eseri gibi duruyor. Paketleme de inanılmaz özenliydi.",
					Item:     "Ayaklı Salata Kasesi",
				},
				{
					Customer: "Zeynep T.",
					Location: "Çankaya / Ankara",
					Quote:    "Arkadaşıma ev hediyesi olarak aldım, bayıldı. Sipariş verdikten 10 dakika sonra WhatsApp'tan kargo sürecini bildirdiler.",
					Item:     "Stoneware Kupa",
				},
			},
			FAQs: []EditorialFAQ{
				{
					Question: "Ürünler bulaşık makinesinde ve mikrodalgada yıkanabilir mi?",
					Answer:   "Evet, tüm ürünlerimiz 1240°C stoneware pişiriminden geçtiği için bulaşık makinesinde, fırında ve mikrodalgada güvenle kullanılabilir.",
				},
				{
					Question: "Kargoda kırılma riski var mı, hasar durumunda ne yapılıyor?",
					Answer:   "Paketlememiz %100 korumalı mukavva ve petek kraft kağıtlarla yapılır. Ancak kargo sürecinde bir aksilik olursa, WhatsApp hattımıza tek bir fotoğraf göndermeniz yeterlidir; hiçbir ücret talep etmeden anında yenisini kargoluyoruz.",
				},
				{
					Question: "WhatsApp üzerinden sipariş nasıl işliyor?",
					Answer:   "Beğendiğiniz ürünün altındaki 'Hızlı Sipariş' butonuna tıklayıp adres bilgilerinizi girmeniz yeterlidir. Sistem otomatik sipariş kodu oluşturup sizi doğrudan atölye hattımıza bağlar, böylece siparişinizi anında onaylarız.",
				},
				{
					Question: "Özel hediye paketi ve el yazısı not ekleyebilir miyim?",
					Answer:   "Elbette. Siparişinizi oluştururken veya WhatsApp üzerinden bize ileterek el yazısı not kartı ve özel kraft hediye paketi talep edebilirsiniz.",
				},
			},
			AtelierAddress: "Moda Zanaatkarlar Caddesi No:18, Kadıköy / İstanbul",
			AtelierHours:   "Pazartesi - Cumartesi: 10:00 - 19:00",
			Instagram:      "@veraseramik",
		}
	}

	if isLeather {
		return NicheEditorial{
			HeroBadge:    "ÖMÜR BOYU DİKİŞ GARANTİSİ · HAKİKİ VAKETA DERİ",
			HeroTitle:    "Zamanla Güzelleşen Hakiki Deri Zanaatı",
			HeroSubtitle: "Geleneksel çift iğne saraç dikişi ve tam damarlı vaketa deri ile nesiller boyu kullanılmak üzere tasarlanan aksesuarlar.",
			HeroImage:    "https://images.unsplash.com/photo-1473187983305-f615310e7daa?auto=format&fit=crop&w=1200&q=80",
			StoryBadge:   "SARAÇ GELENEĞİ",
			StoryTitle:   "Mumlu İplikle Elde Atılan Düğümler",
			StoryText:    "Makinelerin seri üretim soğukluğuna karşı, her parçayı iki iğne ve mumlu saraç ipiyle elde dikiyoruz. Gerçek deri yaşlandıkça karakter kazanır, rengi derinleşir ve sizin hikayenizin bir parçası olur.",
			StoryQuote:   "Gerçek deri eskidikçe yıpranmaz; yaşayan bir malzeme gibi patina kazanır.",
			StoryImage:   "https://images.unsplash.com/photo-1553062407-98eeb64c6a62?auto=format&fit=crop&w=1000&q=80",
			PillarsTitle: "Saraç Atölyesi Standartlarımız",
			Pillars: []EditorialPillar{
				{
					Title: "Tam Damarlı Hakiki Deri",
					Desc:  "Sentetik kaplama içermeyen, doğal dokusu ve dayanıklılığı korunmuş 1. sınıf vaketa deri.",
				},
				{
					Title: "Elde Saraç Dikişi",
					Desc:  "İki iğneyle çift düğümlü dikiş; makine dikişi gibi sökülmez ve ömür boyu formunu korur.",
				},
				{
					Title: "Doğal Patina Karakteri",
					Desc:  "Güneş, temas ve kullanım ile zaman içinde kişiselleşen, eşsiz vintage tonlar.",
				},
				{
					Title: "Ömür Boyu Dikiş Bakımı",
					Desc:  "Atölyemizden çıkan tüm deri parçalar için ömür boyu ücretsiz dikiş ve bakım güvencesi.",
				},
			},
			Testimonials: []EditorialTestimonial{
				{
					Customer: "Emre B.",
					Location: "Beşiktaş / İstanbul",
					Quote:    "Kartlığın deri kokusu ve dikiş hassasiyeti harika. 6 aydır her gün kullanıyorum, patina aldıkça daha da güzelleşti.",
					Item:     "Hakiki Deri Kartlık",
				},
				{
					Customer: "Caner S.",
					Location: "Alsancak / İzmir",
					Quote:    "Laptop çantasının derisi çok kaliteli ve tok. Bölmeleri kullanışlı, dikişleri tam bir saraç ustalığı.",
					Item:     "Deri Laptop Çantası",
				},
				{
					Customer: "Burcu D.",
					Location: "Nilüfer / Bursa",
					Quote:    "Eşime doğum günü hediyesi olarak aldım. Özel hediye kutusu ve notuyla birlikte çok hızlı ulaştı.",
					Item:     "Pasaport & Seyahat Kılıfı",
				},
			},
			FAQs: []EditorialFAQ{
				{
					Question: "Kullandığınız deriler hakiki mi?",
					Answer:   "Evet, tüm ürünlerimizde %100 bitkisel tabaklanmış (vaketa) 1. sınıf hakiki sığır derisi kullanılmaktadır.",
				},
				{
					Question: "Deri bakımı nasıl yapılmalıdır?",
					Answer:   "Doğal deri nemi sever. Yılda 1-2 kez doğal badem yağı veya renksiz deri kremi ile yumuşak bir bezle silmeniz derinin ömrünü uzatır.",
				},
				{
					Question: "Dikişlerde açılma olursa ne yapılıyor?",
					Answer:   "El dikişlerimiz sökülmeye karşı ömür boyu atölyemiz garantisi altındadır. Olası bir durumda ücretsiz onarım sağlıyoruz.",
				},
				{
					Question: "Üzerine isim veya baş harf basılabiliyor mu?",
					Answer:   "Evet, sıcak pres tekniğiyle kişiselleştirme yapabiliyoruz. Siparişinizi WhatsApp üzerinden onaylarken belirtmeniz yeterlidir.",
				},
			},
			AtelierAddress: "Tarihi Saraçhane Hanı No:42, Fatih / İstanbul",
			AtelierHours:   "Hafta içi 09:30 - 18:30",
			Instagram:      "@deriatolyesi",
		}
	}

	// Default Boutique (Fashion, Jewelry, Lifestyle)
	return NicheEditorial{
		HeroBadge:    "SEÇKİN BUTİK KOLEKSİYONU · 2026 SERİSİ",
		HeroTitle:    "Zamansız Zarafet & Özenli Detaylar",
		HeroSubtitle: "Doğal materyaller, rafine kesimler ve günlük hayatınıza şıklık katan özenle seçilmiş butik tasarımlar.",
		HeroImage:    "https://images.unsplash.com/photo-1441986300917-64674bd600d8?auto=format&fit=crop&w=1200&q=80",
		StoryBadge:   "BUTİK ANLAYIŞI",
		StoryTitle:   "Az, Öz ve Nitelikli Tasarım",
		StoryText:    "Hızlı tüketim yerine zamansız parçalara odaklanıyoruz. Koleksiyonlarımızdaki her ürün, malzeme kalitesinden dikiş titizliğine kadar en yüksek standartlarda butik olarak hazırlanır.",
		StoryQuote:   "Zarafet, göze çarpmak değil; akılda kalmaktır.",
		StoryImage:   "https://images.unsplash.com/photo-1490481651871-ab68de25d43d?auto=format&fit=crop&w=1000&q=80",
		PillarsTitle: "Butik Güvencelerimiz",
		Pillars: []EditorialPillar{
			{
				Title: "Seçkin Kumaş & Materyal",
				Desc:  "Nefes alan doğal lifler, kaliteli dokular ve özenle işlenmiş aksesuarlar.",
			},
			{
				Title: "Hızlı ve Sigortalı Kargo",
				Desc:  "Siparişleriniz aynı gün özenle hazırlanır ve sigortalı teslimatla kapınıza ulaşır.",
			},
			{
				Title: "Kolay Beden & Ürün Değişimi",
				Desc:  "Beğenmediğiniz veya bedeni uymayan ürünlerde WhatsApp üzerinden zahmetsiz değişim.",
			},
			{
				Title: "Birebir Butik Danışmanlığı",
				Desc:  "Kombin önerileri ve beden seçimleri için WhatsApp üzerinden kesintisiz stil danışmanlığı.",
			},
		},
		Testimonials: []EditorialTestimonial{
			{
				Customer: "Selin Y.",
				Location: "Nişantaşı / İstanbul",
				Quote:    "Kumaşın kalitesi ve dökümü fotoğraflardakinden bile daha güzel. Butik hizmet kalitesi için teşekkür ederim.",
				Item:     "İpek Gömlek",
			},
			{
				Customer: "Merve K.",
				Location: "Bostanlı / İzmir",
				Quote:    "Sipariş verdim ertesi gün teslim edildi. Paketlemedeki lavanta kokusu ve zarif not çok ince bir detaydı.",
				Item:     "Keten Palazzo Pantolon",
			},
			{
				Customer: "Ece A.",
				Location: "Tunalı / Ankara",
				Quote:    "Kalıpları tam oturuyor, kumaşı terletmiyor. WhatsApp'tan beden konusunda çok yardımcı oldular.",
				Item:     "Doğal Keten Elbise",
			},
		},
		FAQs: []EditorialFAQ{
			{
				Question: "Siparişim ne zaman kargoya verilir?",
				Answer:   "Hafta içi saat 15:00'e kadar verilen siparişler aynı gün, sonraki siparişler ise en geç 24 saat içinde kargoya teslim edilir.",
			},
			{
				Question: "Değişim veya iade imkanı var mı?",
				Answer:   "Evet, teslim aldığınız tarihten itibaren 14 gün içinde faturasıyla birlikte kolayca değişim yapabilirsiniz.",
			},
			{
				Question: "WhatsApp siparişlerinde ödeme nasıl yapılıyor?",
				Answer:   "WhatsApp üzerinden dilerseniz Havale/EFT, dilerseniz güvenli kredi kartı linki (Shopier) ile ödemenizi tamamlayabilirsiniz.",
			},
			{
				Question: "Kargo ücreti ne kadar?",
				Answer:   "1500 TL ve üzeri tüm siparişlerde kargo tamamen ücretsizdir.",
			},
		},
		AtelierAddress: "Abdi İpekçi Caddesi No:24, Nişantaşı / İstanbul",
		AtelierHours:   "Pazartesi - Cumartesi 10:30 - 19:30",
		Instagram:      "@butikatolye",
	}
}

// ConvertToSettingsMap merges the editorial content into a store_settings data map.
func (e NicheEditorial) MergeIntoSettings(settings map[string]any) {
	settings["hero_badge"] = e.HeroBadge
	settings["hero_title"] = e.HeroTitle
	settings["hero_subtitle"] = e.HeroSubtitle
	settings["hero_image"] = e.HeroImage
	settings["story_badge"] = e.StoryBadge
	settings["story_title"] = e.StoryTitle
	settings["story_text"] = e.StoryText
	settings["story_quote"] = e.StoryQuote
	settings["story_image"] = e.StoryImage
	settings["pillars_title"] = e.PillarsTitle

	if pJSON, err := json.Marshal(e.Pillars); err == nil {
		settings["pillars_json"] = string(pJSON)
	}
	if tJSON, err := json.Marshal(e.Testimonials); err == nil {
		settings["testimonials_json"] = string(tJSON)
	}
	if fJSON, err := json.Marshal(e.FAQs); err == nil {
		settings["faqs_json"] = string(fJSON)
	}
	settings["atelier_address"] = e.AtelierAddress
	settings["atelier_hours"] = e.AtelierHours
	settings["instagram"] = e.Instagram
}
