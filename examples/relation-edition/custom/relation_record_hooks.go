// Client Edition lifecycle hooks for relation_record.
// This file is human-owned; generated artifacts stay under generated/relation_record.
package custom

import "context"

// RelationRecordHooks is an explicit extension point. Keep database, authorization and
// tenant decisions in the owning service; dispatch side effects through the
// durable outbox rather than performing network I/O inline.
type RelationRecordHooks struct{}

func (RelationRecordHooks) BeforeCreate(context.Context, any) error      { return nil }
func (RelationRecordHooks) AfterCreate(context.Context, any) error       { return nil }
func (RelationRecordHooks) BeforeUpdate(context.Context, any, any) error { return nil }
func (RelationRecordHooks) AfterUpdate(context.Context, any, any) error  { return nil }
func (RelationRecordHooks) BeforeDelete(context.Context, any) error      { return nil }
func (RelationRecordHooks) AfterDelete(context.Context, any) error       { return nil }
