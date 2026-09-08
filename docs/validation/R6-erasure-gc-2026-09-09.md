# R6 erasure/storage GC doğrulaması

`internal/erasure.GC` her storage key için silmeden hemen önce pin kontrolü yapar; pinli key atlanır, provider hatasında silme durur ve silinen key listesi deterministically döner. Provider delete işlemi idempotent olmalıdır.

Bu process-local orchestration contract'ıdır. Gerçek S3 uyumlu provider, pin ile delete arasındaki dağıtık atomiklik, crash recovery ve GC yarış yük testi açık kalır.
