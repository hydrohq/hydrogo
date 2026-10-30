export interface WhatsAppOrderItem {
  title: string;
  sku?: string;
  size: string;
  color: string;
  price: number;
  quantity: number;
}

export function buildWhatsAppOrderUrl(
  whatsappNumber: string,
  storeName: string,
  item: WhatsAppOrderItem,
  customerName = '',
  shippingAddress = ''
): string {
  const digits = whatsappNumber.replace(/[^0-9]/g, '');
  const total = (item.price * item.quantity).toLocaleString('tr-TR', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  });
  const lines = [
    `Merhaba *${storeName}* 👋`,
    `Web sitenizden sipariş vermek istiyorum:`,
    ``,
    `🛍️ *Ürün:* ${item.title}${item.sku ? ` (${item.sku})` : ''}`,
    `📏 *Beden:* ${item.size || 'Standart'}`,
    `🎨 *Renk:* ${item.color || 'Standart'}`,
    `🔢 *Adet:* ${item.quantity}`,
    `💳 *Toplam Tutar:* ${total} TL`
  ];
  if (customerName.trim()) {
    lines.push(`👤 *Ad Soyad:* ${customerName.trim()}`);
  }
  if (shippingAddress.trim()) {
    lines.push(`📍 *Teslimat Adresi:* ${shippingAddress.trim()}`);
  }
  const text = encodeURIComponent(lines.join('\n'));
  return `https://wa.me/${digits}?text=${text}`;
}
