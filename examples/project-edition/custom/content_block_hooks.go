// Client Edition lifecycle hooks for content_block.
// This file is human-owned; generated artifacts stay under generated/content_block.
package custom

import "context"

// ContentBlockHooks is an explicit extension point. Keep database, authorization and
// tenant decisions in the owning service; dispatch side effects through the
// durable outbox rather than performing network I/O inline.
type ContentBlockHooks struct{}

func (ContentBlockHooks) BeforeCreate(context.Context, any) error      { return nil }
func (ContentBlockHooks) AfterCreate(context.Context, any) error       { return nil }
func (ContentBlockHooks) BeforeUpdate(context.Context, any, any) error { return nil }
func (ContentBlockHooks) AfterUpdate(context.Context, any, any) error  { return nil }
func (ContentBlockHooks) BeforeDelete(context.Context, any) error      { return nil }
func (ContentBlockHooks) AfterDelete(context.Context, any) error       { return nil }
