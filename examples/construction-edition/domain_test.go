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
	if CanProject("p1", ProjectMembership{ProjectID: "p1", UserID: "u1", Role: ProjectRoleContributor}, CapabilityApprovalDecide) || !CanProject("p1", ProjectMembership{ProjectID: "p1", UserID: "u1", Role: ProjectRoleManager}, CapabilityApprovalDecide) || CanProject("p2", ProjectMembership{ProjectID: "p1", UserID: "u1", Role: ProjectRoleOwner}, CapabilityProjectRead) {
		t.Fatal("project role authorization matrix is not fail-closed")
	}
	if err := ValidateDocumentRevision(DocumentRevision{DocumentID: "d1", Revision: 1, ObjectKey: "documents/d1/r1.pdf", Checksum: "sha256:abc", CreatedBy: "u1"}); err != nil {
		t.Fatal(err)
	}
	manager := ProjectMembership{ProjectID: "p1", UserID: "u1", Role: ProjectRoleManager}
	document, err := ApproveDocument("p1", manager, ProjectDocument{ProjectID: "p1", Revision: DocumentRevision{DocumentID: "d1", Revision: 1, ObjectKey: "documents/d1/r1.pdf", Checksum: "sha256:abc", CreatedBy: "u2"}, Status: DocumentPending}, 1)
	if err != nil || document.Status != DocumentApproved || document.Revision.Revision != 2 {
		t.Fatalf("document approval = %#v, %v", document, err)
	}
	if _, err := ApproveDocument("p1", manager, document, 1); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale document approval = %v", err)
	}
	if err := ValidateDailyReport(DailyReport{ProjectID: "p1", ReportDay: time.Now(), AuthorID: "u1", Summary: "Concrete delivery"}); err != nil {
		t.Fatal(err)
	}
	report, err := ApproveDailyReport("p1", manager, ReviewedDailyReport{DailyReport: DailyReport{ProjectID: "p1", ReportDay: time.Now(), AuthorID: "u2", Summary: "Concrete delivery"}, Revision: 1, Status: DailyReportPending}, 1)
	if err != nil || report.Status != DailyReportApproved || report.Revision != 2 {
		t.Fatalf("daily report approval = %#v, %v", report, err)
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
