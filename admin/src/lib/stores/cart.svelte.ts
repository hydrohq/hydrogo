export interface CartItem {
  id: string;
  name: string;
  priceCents: number;
  currency: string;
  quantity: number;
  image?: string;
  sku?: string;
}

class CartStore {
  items = $state<CartItem[]>(this.loadStorage());
  isOpen = $state(false);

  private loadStorage(): CartItem[] {
    if (typeof window === 'undefined') return [];
    try {
      const saved = localStorage.getItem('hydro_cart');
      return saved ? JSON.parse(saved) : [];
    } catch {
      return [];
    }
  }

  private saveStorage() {
    if (typeof window === 'undefined') return;
    try {
      localStorage.setItem('hydro_cart', JSON.stringify(this.items));
    } catch {}
  }

  totalCount = $derived(this.items.reduce((acc, item) => acc + item.quantity, 0));
  totalCents = $derived(this.items.reduce((acc, item) => acc + item.priceCents * item.quantity, 0));

  formattedTotal = $derived(
    (this.totalCents / 100).toLocaleString('tr-TR', {
      style: 'currency',
      currency: this.items[0]?.currency || 'TRY',
      minimumFractionDigits: 2,
    })
  );

  addItem(product: { id: string; name: string; priceCents: number; currency?: string; image?: string; sku?: string }) {
    const existing = this.items.find((i) => i.id === product.id);
    if (existing) {
      existing.quantity += 1;
    } else {
      this.items.push({
        id: product.id,
        name: product.name,
        priceCents: product.priceCents,
        currency: product.currency || 'TRY',
        quantity: 1,
        image: product.image,
        sku: product.sku,
      });
    }
    this.saveStorage();
    this.isOpen = true;
  }

  updateQuantity(id: string, delta: number) {
    const item = this.items.find((i) => i.id === id);
    if (!item) return;
    item.quantity += delta;
    if (item.quantity <= 0) {
      this.items = this.items.filter((i) => i.id !== id);
    }
    this.saveStorage();
  }

  removeItem(id: string) {
    this.items = this.items.filter((i) => i.id !== id);
    this.saveStorage();
  }

  clear() {
    this.items = [];
    this.saveStorage();
  }

  toggle(open?: boolean) {
    if (open !== undefined) {
      this.isOpen = open;
    } else {
      this.isOpen = !this.isOpen;
    }
  }
}

export const cart = new CartStore();
