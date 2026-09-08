# HydroGo Compact kişisel veri politikası

Bu belge R3 Site Edition form inbox'ı için uygulanabilir veri yaşam döngüsünü tanımlar. Form inbox yalnız allowlist'e alınmış form adları için kullanılır; Compact başlangıcında `contact` formu desteklenir.

## Toplama ve erişim

- Yalnız form şemasında tanımlı alanlar alınır; bilinmeyen alanlar reddedilir.
- Form gönderimi rate limit ve boyut limitiyle korunur.
- Inbox kayıtlarını yalnız admin rolü okuyabilir ve durumunu değiştirebilir. Editor/viewer rolleri kişisel veri inbox'ına erişemez.
- Audit kaydı içerik değerlerini değil, işlem türü ve hedef kimliğini taşır.

## Saklama, silme ve dışa aktarma

- Inbox kaydı varsayılan olarak 30 gün tutulur; `PurgeInbox` yalnız süresi dolmuş kayıtları siler.
- Müşteri talebiyle erişim/dışa aktarma, admin tarafından doğrulanmış kullanıcı talebi üzerinden manuel runbook adımıyla yapılır; export dosyası şifreli geçici depoda tutulur ve işlem bitince silinir.
- Silme talebi için inbox kaydı ve ilgili medya hedefi dış silme günlüğüne idempotent niyet olarak yazılır; replay tamamlanmadan kayıt silinmiş kabul edilmez.
- Yedeklerdeki kayıtlar, yedek retention süresi dolana kadar şifreli kalır; restore sonrası aynı purge ve silme replay adımları tekrar çalıştırılır.

## Operasyon kanıtı

Her erişim, export, durum değişikliği ve purge işlemi audit olayına bağlanır. Operatör checkpoint'i, etkilenen kayıt sayısını ve başarısız replay'leri runbook'a işler.
