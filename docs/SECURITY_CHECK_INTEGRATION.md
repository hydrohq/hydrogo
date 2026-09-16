# security-check geliştirici entegrasyonu

HydroGo için `ersinkoc/security-check` bundle’ı yerel Codex skill dizinine
kuruludur. Kurulum kaynağı commit’i:

```text
80d68f68da642f65883ca1791f03fb3ceaf456d5
```

## Codex

Global skill kurulumundan sonra yeni bir Codex turn’ünde aşağıdaki istekler
doğrudan çalışır:

```text
run security check
scan diff
```

## Antigravity, agy ve diğer Agentskills CLI’ları

Bu araçlar için proje kökünde çalıştırılabilecek resmi upstream yöntemi:

```bash
npx skills add ersinkoc/security-check
```

Kurulumdan sonra `.agents/skills/` altında `security-check` orkestratörü ve
`sc-*` skill’leri görünmelidir. Windows PowerShell alternatifi:

```powershell
irm https://raw.githubusercontent.com/ersinkoc/security-check/main/skills.ps1 | iex
```

Kurulum sırasında kaynak commit’ini ve checksum’ı ekip içinde kaydedin; uzak
`main` dalını production build zincirine otomatik olarak güvenilir kabul
etmeyin. Skill yalnızca kaynak incelemesi ve açıkça izin verilen izole yerel
kontroller yapmalıdır; canlı endpoint, credential veya shared altyapı taraması
ayrı bir yetkilendirme ve staging kapısıdır.

HydroGo’nun son source-only audit çıktısı [`security-report/`](../security-report/)
altındadır. Sonucu production penetrasyon testi yerine kullanmayın.
