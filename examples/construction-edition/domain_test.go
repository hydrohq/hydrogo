package main

import (
	"errors"
	"testing"
	"time"
)

func TestConstructionDomainContracts(t *testing.T) {
	if err := ValidateOrganization(Organization{ID: "org-a", ParentID: "org-a", Name: "Alpha"}); !errors.Is(err, ErrInvalidOrganization) {
		t.Fatalf("self-parent organization = %v", err)
	}
	if err := ValidateProjectMembership(ProjectMembership{ProjectID: "p1", UserID: "u1", Role: ProjectRoleManager}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDocumentRevision(DocumentRevision{DocumentID: "d1", Revision: 1, ObjectKey: "documents/d1/r1.pdf", Checksum: "sha256:abc", CreatedBy: "u1"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDailyReport(DailyReport{ProjectID: "p1", ReportDay: time.Now(), AuthorID: "u1", Summary: "Concrete delivery"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateIssue(Issue{ProjectID: "p1", Title: "Gate access", OpenedBy: "u1"}); err != nil {
		t.Fatal(err)
	}
	if CanTransitionApproval(ApprovalDraft, ApprovalApproved) || !CanTransitionApproval(ApprovalPending, ApprovalApproved) {
		t.Fatal("approval workflow bypass accepted")
	}
	if err := ValidateConstructionEvent("construction.document.approved", 1); err != nil {
		t.Fatal(err)
	}
	if err := ValidateConstructionEvent("construction.document.approved", 2); !errors.Is(err, ErrUnknownConstructionEvent) {
		t.Fatalf("unknown construction event version = %v", err)
	}
}
