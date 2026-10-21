// Fills a HydroGo CRM project with demo data through its admin API.
//   HYDRO_API_ORIGIN=http://127.0.0.1:8080 HYDRO_ADMIN_EMAIL=… HYDRO_ADMIN_PASSWORD=… node scripts/seed.mjs
const origin = (process.env.HYDRO_API_ORIGIN ?? 'http://127.0.0.1:8080').replace(/\/$/, '');
const email = process.env.HYDRO_ADMIN_EMAIL;
const password = process.env.HYDRO_ADMIN_PASSWORD;
if (!email || !password) {
  console.error('Set HYDRO_ADMIN_EMAIL and HYDRO_ADMIN_PASSWORD.');
  process.exit(1);
}

const cookieOf = (res, name) => res.headers.getSetCookie().map((c) => c.split(';')[0]).find((c) => c.startsWith(`${name}=`));

const csrfRes = await fetch(`${origin}/csrf-token`);
const csrfCookie = cookieOf(csrfRes, 'hydro_csrf');
const { token } = await csrfRes.json();
const loginRes = await fetch(`${origin}/auth/login`, {
  method: 'POST',
  headers: { 'content-type': 'application/x-www-form-urlencoded', 'x-csrf-token': token, cookie: csrfCookie },
  body: new URLSearchParams({ email, password })
});
if (!loginRes.ok) {
  console.error(`Sign-in failed (${loginRes.status}).`);
  process.exit(1);
}
const cookie = `${cookieOf(loginRes, 'hydro_session')}; ${csrfCookie}`;

async function create(type, fields) {
  const res = await fetch(`${origin}/api/v1/admin/content/${type}`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', 'x-csrf-token': token, cookie },
    body: JSON.stringify(fields)
  });
  if (!res.ok) throw new Error(`${type}: ${res.status} ${await res.text()}`);
  return (await res.json()).data.id;
}

const day = (offset) => new Date(Date.now() + offset * 86400000).toISOString();
const companies = [
  ['Northwind Software', 'software', 'northwind.example'],
  ['Contoso Manufacturing', 'manufacturing', 'contoso.example'],
  ['Fabrikam Retail', 'retail', 'fabrikam.example']
];
let deals = 0;
for (const [index, [name, industry, website]] of companies.entries()) {
  const company = await create('companies', { name, industry, website });
  const contact = await create('contacts', { full_name: `Alex ${name.split(' ')[0]}`, email: `alex@${website}`, company, status: 'active' });
  const stages = ['lead', 'qualified', 'proposal', 'negotiation', 'won'];
  for (let i = 0; i < 2; i++) {
    const stage = stages[(index + i * 2) % stages.length];
    const deal = await create('deals', { name: `${name.split(' ')[0]} ${i ? 'expansion' : 'rollout'}`, company, contact, stage, amount: 5000 * (index + 1) * (i + 1), currency: index === 2 ? 'EUR' : 'USD' });
    await create('activities', { subject: `Follow up: ${name.split(' ')[0]}`, kind: 'call', due: day(i - 1), done: false, deal, company, contact });
    deals++;
  }
}
console.log(`Seeded ${companies.length} companies, ${companies.length} contacts, ${deals} deals.`);
