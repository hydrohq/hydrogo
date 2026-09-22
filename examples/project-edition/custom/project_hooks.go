// Client Edition lifecycle hooks for project.
// This file is human-owned; generated artifacts stay under generated/project.
package custom

import "context"

// ProjectHooks is an explicit extension point. Keep database, authorization and
// tenant decisions in the owning service; dispatch side effects through the
// durable outbox rather than performing network I/O inline.
type ProjectHooks struct{}

func (ProjectHooks) BeforeCreate(context.Context, any) error      { return nil }
func (ProjectHooks) AfterCreate(context.Context, any) error       { return nil }
func (ProjectHooks) BeforeUpdate(context.Context, any, any) error { return nil }
func (ProjectHooks) AfterUpdate(context.Context, any, any) error  { return nil }
func (ProjectHooks) BeforeDelete(context.Context, any) error      { return nil }
func (ProjectHooks) AfterDelete(context.Context, any) error       { return nil }
