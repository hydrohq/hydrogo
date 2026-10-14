// Client Edition lifecycle hooks for media_article.
// This file is human-owned; generated artifacts stay under generated/media_article.
package custom

import "context"

// MediaArticleHooks is an explicit extension point. Keep database, authorization and
// tenant decisions in the owning service; dispatch side effects through the
// durable outbox rather than performing network I/O inline.
type MediaArticleHooks struct{}

func (MediaArticleHooks) BeforeCreate(context.Context, any) error      { return nil }
func (MediaArticleHooks) AfterCreate(context.Context, any) error       { return nil }
func (MediaArticleHooks) BeforeUpdate(context.Context, any, any) error { return nil }
func (MediaArticleHooks) AfterUpdate(context.Context, any, any) error  { return nil }
func (MediaArticleHooks) BeforeDelete(context.Context, any) error      { return nil }
func (MediaArticleHooks) AfterDelete(context.Context, any) error       { return nil }
