package commerce

import "testing"

func TestValidateCartRequiresExactlyOneOwner(t *testing.T) {
	base := Cart{Currency: "TRY"}
	if err := ValidateCart(base); err == nil {
		t.Fatal("expected an error when neither member_id nor anonymous_token is set")
	}
	both := Cart{Currency: "TRY", MemberID: "m1", AnonymousToken: "a1"}
	if err := ValidateCart(both); err == nil {
		t.Fatal("expected an error when both member_id and anonymous_token are set")
	}
	member := Cart{Currency: "TRY", MemberID: "m1"}
	if err := ValidateCart(member); err != nil {
		t.Fatalf("member-owned cart should validate: %v", err)
	}
	anonymous := Cart{Currency: "TRY", AnonymousToken: "a1"}
	if err := ValidateCart(anonymous); err != nil {
		t.Fatalf("anonymous cart should validate: %v", err)
	}
}

func TestValidateItemRejectsNonPositiveQuantity(t *testing.T) {
	item := CartItem{VariantID: "v1", Quantity: 0, UnitPriceCents: 100}
	if err := ValidateItem(item); err == nil {
		t.Fatal("expected an error for zero quantity")
	}
	item.Quantity = -1
	if err := ValidateItem(item); err == nil {
		t.Fatal("expected an error for negative quantity")
	}
	item.Quantity = 1
	if err := ValidateItem(item); err != nil {
		t.Fatalf("valid item should pass: %v", err)
	}
}

func TestValidateOrderRequiresGuestEmailForGuestOrders(t *testing.T) {
	order := Order{IdempotencyKey: "k1", Currency: "TRY"}
	if err := ValidateOrder(order); err == nil {
		t.Fatal("expected an error: guest order with no email")
	}
	order.GuestEmail = "guest@example.com"
	if err := ValidateOrder(order); err != nil {
		t.Fatalf("guest order with email should validate: %v", err)
	}
	member := Order{IdempotencyKey: "k1", Currency: "TRY", MemberID: "m1"}
	if err := ValidateOrder(member); err != nil {
		t.Fatalf("member order without guest email should validate: %v", err)
	}
}

func TestValidateOrderRequiresIdempotencyKey(t *testing.T) {
	order := Order{Currency: "TRY", MemberID: "m1"}
	if err := ValidateOrder(order); err == nil {
		t.Fatal("expected an error for missing idempotency_key")
	}
}
