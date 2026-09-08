# R6 cache fan-out doğrulaması

`internal/cache.Fanout` bounded subscriber buffer kullanır. Slow subscriber mesaj taşmasına uğradığında kuyruk büyütülmez; tek bir resync sinyali üretilir ve monotonic generation korunur. Subscriber güncel generation snapshot'ını `Reconcile` ile doğrulamadan resync durumunu temizleyemez. Kapanan bus subscriber kanallarını kapatır ve yeni publish generation ilerletmez.

Bu contract process-local davranışı doğrular. Gerçek çoklu web/worker, L2 provider, reconnect sonrası generation snapshot uzlaştırması ve network partition yük testi açık kalır.

`internal/cache.TwoTier` L1 hit'i öncelemekte, L2 hit'inde L1'i ısıtmakta ve L2 get/set hatalarında bounded local loader yoluna düşmektedir. Remote backend’in gerçek bağlantı havuzu, partition ve çoklu-instance invalidation testi bu sözleşmenin dışındadır.
