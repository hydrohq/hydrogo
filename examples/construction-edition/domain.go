package main

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidOrganization = errors.New("organization hierarchy is invalid")
	ErrInvalidProjectRole  = errors.New("project role is invalid")
	ErrInvalidDocument     = errors.New("document revision is invalid")
	ErrInvalidDailyReport  = errors.New("daily report is invalid")
	ErrInvalidIssue        = errors.New("issue is invalid")
	ErrInvalidApproval     = errors.New("approval transition is invalid")
)

// Organization is a Construction Edition domain object. ParentID is only a
// business hierarchy reference; it is never treated as a tenant boundary.
type Organization struct {
	ID       string
	ParentID string
	Name     string
}

type ProjectRole string

const (
	ProjectRoleOwner       ProjectRole = "owner"
	ProjectRoleManager     ProjectRole = "manager"
	ProjectRoleContributor ProjectRole = "contributor"
	ProjectRoleViewer      ProjectRole = "viewer"
)

type ProjectMembership struct {
	ProjectID string
	UserID    string
	Role      ProjectRole
}

type ProjectCapability string

const (
	CapabilityProjectRead       ProjectCapability = "project.read"
	CapabilityDocumentWrite     ProjectCapability = "document.write"
	CapabilityDailyReportWrite  ProjectCapability = "daily-report.write"
	CapabilityIssueWrite        ProjectCapability = "issue.write"
	CapabilityApprovalDecide    ProjectCapability = "approval.decide"
	CapabilityProjectMembership ProjectCapability = "project.membership.manage"
)

type DocumentRevision struct {
	DocumentID string
	Revision   int
	ObjectKey  string
	Checksum   string
	CreatedBy  string
}

type DailyReport struct {
	ProjectID string
	ReportDay time.Time
	AuthorID  string
	Summary   string
}

type Issue struct {
	ProjectID string
	Title     string
	OpenedBy  string
}

type ApprovalState string

const (
	ApprovalDraft    ApprovalState = "draft"
	ApprovalPending  ApprovalState = "pending"
	ApprovalApproved ApprovalState = "approved"
	ApprovalRejected ApprovalState = "rejected"
)

func ValidateOrganization(value Organization) error {
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.Name) == "" || value.ID == value.ParentID {
		return ErrInvalidOrganization
	}
	return nil
}

func ValidateProjectMembership(value ProjectMembership) error {
	if strings.TrimSpace(value.ProjectID) == "" || strings.TrimSpace(value.UserID) == "" {
		return ErrInvalidProjectRole
	}
	switch value.Role {
	case ProjectRoleOwner, ProjectRoleManager, ProjectRoleContributor, ProjectRoleViewer:
		return nil
	default:
		return ErrInvalidProjectRole
	}
}

// CanProject performs a deny-by-default decision for one already scoped
// project membership. Callers must still verify that the membership belongs to
// the requested project; a role string alone is never authorization.
func CanProject(membership ProjectMembership, capability ProjectCapability) bool {
	if ValidateProjectMembership(membership) != nil {
		return false
	}
	switch membership.Role {
	case ProjectRoleOwner:
		return capability == CapabilityProjectRead || capability == CapabilityDocumentWrite || capability == CapabilityDailyReportWrite || capability == CapabilityIssueWrite || capability == CapabilityApprovalDecide || capability == CapabilityProjectMembership
	case ProjectRoleManager:
		return capability == CapabilityProjectRead || capability == CapabilityDocumentWrite || capability == CapabilityDailyReportWrite || capability == CapabilityIssueWrite || capability == CapabilityApprovalDecide
	case ProjectRoleContributor:
		return capability == CapabilityProjectRead || capability == CapabilityDocumentWrite || capability == CapabilityDailyReportWrite || capability == CapabilityIssueWrite
	case ProjectRoleViewer:
		return capability == CapabilityProjectRead
	default:
		return false
	}
}

func ValidateDocumentRevision(value DocumentRevision) error {
	if strings.TrimSpace(value.DocumentID) == "" || value.Revision < 1 || strings.TrimSpace(value.ObjectKey) == "" || strings.TrimSpace(value.Checksum) == "" || strings.TrimSpace(value.CreatedBy) == "" {
		return ErrInvalidDocument
	}
	return nil
}

func ValidateDailyReport(value DailyReport) error {
	if strings.TrimSpace(value.ProjectID) == "" || value.ReportDay.IsZero() || strings.TrimSpace(value.AuthorID) == "" || strings.TrimSpace(value.Summary) == "" {
		return ErrInvalidDailyReport
	}
	return nil
}

func ValidateIssue(value Issue) error {
	if strings.TrimSpace(value.ProjectID) == "" || strings.TrimSpace(value.Title) == "" || strings.TrimSpace(value.OpenedBy) == "" {
		return ErrInvalidIssue
	}
	return nil
}

func CanTransitionApproval(from, to ApprovalState) bool {
	return (from == ApprovalDraft && to == ApprovalPending) ||
		(from == ApprovalPending && (to == ApprovalApproved || to == ApprovalRejected))
}
