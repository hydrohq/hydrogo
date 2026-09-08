# SDK v1 destek politikası

HydroGo SDK uyumluluğu aynı major sürüm kuralıyla değerlendirilir. `1.x` istemci yüzeyi `1.x` runtime ile çalışır; major değişikliği otomatik uyumlu kabul edilmez ve ayrı migration/ADR gerektirir. Candidate sürümleri (`-candidate`) kararlı release sözü taşımaz.

Her minor sürüm geriye uyumlu ekleme hedefler. Deprecated semboller en az bir sonraki minor sürüm boyunca uyarı ile korunur; kaldırma yalnızca yeni major sürümde yapılır. Kritik güvenlik düzeltmeleri desteklenen son kararlı major sürüme uygulanır. Candidate sürümdeki API, v1 kararlı yayıma girmeden önce değişebilir.

Bu politika satış öncesi destek süresi veya production key ceremony yerine geçmez. Kararlı v1 yayımlandığında desteklenen major, deprecation tarihleri ve güvenlik SLA'sı release manifestiyle birlikte ayrıca ilan edilir.
