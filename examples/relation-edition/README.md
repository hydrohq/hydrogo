# Relation Edition

This Client Edition fixture demonstrates generated bounded many-valued
relations. `relation_record.tag_ids` uses a tenant-scoped normalized join
table. The generated repository offers transactional replacement and bounded
ordered listing; Edition code remains responsible for resolving and
authorizing target tags before it writes IDs.

The contract is generated from `blueprint.yaml`. Apply its reviewed migration
explicitly through the Edition migration workflow; generation never executes
DDL.
