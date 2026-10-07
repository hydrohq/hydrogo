// Client Edition lifecycle hooks for milestone.
// This file is human-owned; generated artifacts stay under generated/milestone.
package custom

import "context"

// MilestoneHooks is an explicit extension point. Keep database, authorization and
// tenant decisions in the owning service; dispatch side effects through the
// durable outbox rather than performing network I/O inline.
type MilestoneHooks struct{}

func (MilestoneHooks) BeforeCreate(context.Context, any) error { return nil }
func (MilestoneHooks) AfterCreate(context.Context, any) error  { return nil }
func (MilestoneHooks) BeforeUpdate(context.Context, any, any) error { return nil }
func (MilestoneHooks) AfterUpdate(context.Context, any, any) error  { return nil }
func (MilestoneHooks) BeforeDelete(context.Context, any) error { return nil }
func (MilestoneHooks) AfterDelete(context.Context, any) error  { return nil }
