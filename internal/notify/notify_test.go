package notify

import "testing"

func TestMessageTextJoinsTitleAndBodyWhenBothArePresent(t *testing.T) {
	got := Message{Title: "Yeni sipariş", Body: "Toplam: 120,00 ₺"}.Text()
	want := "Yeni sipariş\nToplam: 120,00 ₺"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMessageTextIsJustTheTitleWhenBodyIsEmpty(t *testing.T) {
	got := Message{Title: "Yeni sipariş"}.Text()
	if got != "Yeni sipariş" {
		t.Fatalf("got %q", got)
	}
}
