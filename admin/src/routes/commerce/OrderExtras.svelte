<script lang="ts">
  import { api, ApiClientError } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { confirmAction } from '$lib/confirm';
  import Button from '$lib/components/ui/Button.svelte';

  interface Item { sku: string; title: string; quantity: number }
  interface Order {
    id: string; created_at: string; payment_status: string; fulfillment_status: string; shipped_at?: string;
    guest_name?: string; guest_email?: string; guest_phone?: string; guest_address?: string; guest_district?: string; guest_city?: string; guest_postal_code?: string; guest_country?: string;
    items: Item[];
  }
  interface Note { id: string; note: string; author?: string; created_at?: string }
  let { order, canWrite, onchanged }: { order: Order; canWrite: boolean; onchanged: () => void } = $props();

  interface Invoice { invoice_type: string; tckn?: string; vkn?: string; company_name?: string; tax_office?: string }
  let invoice = $state<Invoice | null>(null);
  let notes = $state<Note[]>([]);
  let notesAvailable = $state(true);
  let draft = $state('');
  let busy = $state(false);
  let error = $state('');

  // order_notes is an admin-only content type; when a project does not define it the notes box simply does not show.
  async function loadNotes() {
    try {
      const res = await api.get<{ data: Note[] }>('/api/v1/admin/content/order_notes', { params: { 'filter[order_id]': order.id, limit: 50, sort: '-created_at' } });
      notes = res.data ?? [];
      notesAvailable = true;
    } catch (e) {
      notesAvailable = !(e instanceof ApiClientError && e.status === 404);
    }
  }
  // Invoice details the customer typed at checkout (invoice_details content type); absent for older orders.
  async function loadInvoice() {
    invoice = null;
    try {
      const res = await api.get<{ data: Invoice[] }>('/api/v1/admin/content/invoice_details', { params: { 'filter[order_id]': order.id, limit: 1 } });
      invoice = res.data?.[0] ?? null;
    } catch {
      invoice = null;
    }
  }
  $effect(() => { void order.id; void loadNotes(); void loadInvoice(); });

  async function addNote() {
    if (!draft.trim()) return;
    busy = true;
    error = '';
    try {
      await api.post('/api/v1/admin/content/order_notes', { order_id: order.id, note: draft.trim(), author: auth.user?.email ?? '' });
      draft = '';
      await loadNotes();
    } catch (e) {
      error = e instanceof ApiClientError ? e.message : 'Not kaydedilemedi.';
    } finally {
      busy = false;
    }
  }

  async function cancelOrder() {
    const ok = await confirmAction({
      title: 'Sipariş iptal edilsin mi?',
      description: 'Ödeme gelmemiş sipariş başarısız yapılır, stok serbest kalır ve kuponu geri verilir. Müşteri ödemeyi bu sipariş için artık tamamlayamaz.',
      confirmLabel: 'Evet, iptal et',
      cancelLabel: 'Vazgeç',
      variant: 'destructive'
    });
    if (!ok) return;
    busy = true;
    error = '';
    try {
      await api.post(`/api/v1/admin/commerce/orders/${encodeURIComponent(order.id)}/cancel`);
      onchanged();
    } catch (e) {
      error = e instanceof ApiClientError ? e.message : 'Sipariş iptal edilemedi.';
    } finally {
      busy = false;
    }
  }

  const when = (v?: string) => (v ? new Date(v).toLocaleString('tr-TR') : '');
  // Only what is actually recorded: the payment time is not stored, so no time is shown for it.
  const timeline = $derived([
    { label: 'Sipariş oluşturuldu', at: order.created_at },
    ...(order.payment_status === 'paid' ? [{ label: 'Ödeme alındı', at: '' }] : []),
    ...(order.payment_status === 'failed' ? [{ label: 'Ödeme gelmedi / sipariş iptal', at: '' }] : []),
    ...(order.payment_status === 'refunded' ? [{ label: 'Ödeme iade edildi', at: '' }] : []),
    ...(order.shipped_at && !order.shipped_at.startsWith('0001') ? [{ label: 'Kargoya verildi', at: order.shipped_at }] : [])
  ]);

  const esc = (s = '') => s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c] as string));
  const addressLines = () => [order.guest_name, order.guest_address, [order.guest_district, order.guest_city].filter(Boolean).join(' / '), [order.guest_postal_code, order.guest_country].filter(Boolean).join(' '), order.guest_phone].filter(Boolean) as string[];

  function printSheet(kind: 'slip' | 'label') {
    const w = window.open('', '_blank', 'width=800,height=900');
    if (!w) return;
    const addr = addressLines().map((l) => `<div>${esc(l)}</div>`).join('');
    const body = kind === 'label'
      ? `<div class="label"><div class="to">ALICI</div>${addr}<div class="ref">Sipariş #${esc(order.id.replace('order_', '').slice(0, 8).toUpperCase())}</div></div>`
      : `<h1>Paketleme fişi</h1><p>Sipariş #${esc(order.id.replace('order_', '').slice(0, 8).toUpperCase())} · ${esc(when(order.created_at))}</p>
         <h2>Alıcı</h2>${addr}
         <h2>Ürünler</h2><table>${order.items.map((i) => `<tr><td>☐</td><td>${esc(i.title)}<br><small>${esc(i.sku)}</small></td><td class="q">× ${i.quantity}</td></tr>`).join('')}</table>`;
    w.document.write(`<!doctype html><meta charset="utf-8"><title>${kind === 'label' ? 'Adres etiketi' : 'Paketleme fişi'}</title>
      <style>body{font:15px/1.5 system-ui,sans-serif;margin:32px;color:#111}h1{font-size:22px}h2{font-size:13px;text-transform:uppercase;letter-spacing:.1em;margin:22px 0 6px;color:#555}
      table{width:100%;border-collapse:collapse}td{padding:8px 6px;border-bottom:1px solid #ddd}.q{text-align:right;white-space:nowrap}
      .label{border:2px solid #111;padding:24px;max-width:420px;font-size:20px;line-height:1.4}.to{font-size:12px;letter-spacing:.2em;margin-bottom:8px}.ref{margin-top:16px;font-size:12px;color:#555}
      @media print{body{margin:12mm}}</style>${body}<script>window.onload=()=>setTimeout(()=>window.print(),150)<\/script>`);
    w.document.close();
  }
</script>

<div class="space-y-4 border-t border-border pt-4 text-xs">
  <section>
    <h3 class="mb-2 font-medium text-muted-foreground">Süreç</h3>
    <ol class="space-y-1.5 border-l border-border pl-4">
      {#each timeline as step (step.label)}
        <li class="relative"><span class="absolute -left-[1.3rem] top-1.5 size-2 rounded-full bg-primary"></span>{step.label}{#if step.at}<span class="ml-2 text-muted-foreground">{when(step.at)}</span>{/if}</li>
      {/each}
    </ol>
  </section>

  <section class="flex flex-wrap gap-2">
    <Button variant="outline" size="sm" onclick={() => printSheet('slip')}>Paketleme fişi yazdır</Button>
    <Button variant="outline" size="sm" onclick={() => printSheet('label')}>Adres etiketi yazdır</Button>
    {#if canWrite && order.payment_status === 'pending'}
      <Button variant="danger" size="sm" loading={busy} onclick={cancelOrder}>Siparişi iptal et</Button>
    {/if}
  </section>

  {#if invoice}
    <section>
      <h3 class="mb-2 font-medium text-muted-foreground">Fatura bilgisi</h3>
      <dl class="grid grid-cols-[8rem_1fr] gap-y-1 rounded-md border border-border bg-muted/40 p-3">
        <dt class="text-muted-foreground">Tür</dt><dd class="capitalize">{invoice.invoice_type}</dd>
        {#if invoice.invoice_type === 'kurumsal'}
          <dt class="text-muted-foreground">Unvan</dt><dd>{invoice.company_name}</dd>
          <dt class="text-muted-foreground">VKN</dt><dd class="font-mono">{invoice.vkn}</dd>
          <dt class="text-muted-foreground">Vergi dairesi</dt><dd>{invoice.tax_office}</dd>
        {:else}
          <dt class="text-muted-foreground">TCKN</dt><dd class="font-mono">{invoice.tckn || '— (girilmedi)'}</dd>
        {/if}
      </dl>
    </section>
  {/if}

  {#if notesAvailable}
    <section>
      <h3 class="mb-2 font-medium text-muted-foreground">Sipariş notları (yalnızca ekip görür)</h3>
      {#if notes.length === 0}<p class="text-muted-foreground">Henüz not yok.</p>{/if}
      <ul class="space-y-2">
        {#each notes as n (n.id)}
          <li class="rounded-md border border-border bg-muted/40 p-2.5"><p class="whitespace-pre-wrap text-foreground">{n.note}</p><p class="mt-1 text-[11px] text-muted-foreground">{n.author ?? ''} {when(n.created_at)}</p></li>
        {/each}
      </ul>
      {#if canWrite}
        <div class="mt-2 flex gap-2">
          <textarea bind:value={draft} rows="2" placeholder="Not ekle…" class="w-full rounded-md border border-input bg-background px-3 py-1.5 text-sm"></textarea>
          <Button size="sm" disabled={busy || !draft.trim()} onclick={addNote}>Ekle</Button>
        </div>
      {/if}
    </section>
  {/if}
  {#if error}<p role="alert" class="text-destructive">{error}</p>{/if}
</div>
