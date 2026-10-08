// Client Edition lifecycle hooks for article.
// This file is human-owned; generated artifacts stay under generated/article.
package custom

import "context"

// ArticleHooks is an explicit extension point. Keep database, authorization and
// tenant decisions in the owning service; dispatch side effects through the
// durable outbox rather than performing network I/O inline.
type ArticleHooks struct{}

func (ArticleHooks) BeforeCreate(context.Context, any) error { return nil }
func (ArticleHooks) AfterCreate(context.Context, any) error  { return nil }
func (ArticleHooks) BeforeUpdate(context.Context, any, any) error { return nil }
func (ArticleHooks) AfterUpdate(context.Context, any, any) error  { return nil }
func (ArticleHooks) BeforeDelete(context.Context, any) error { return nil }
func (ArticleHooks) AfterDelete(context.Context, any) error  { return nil }
