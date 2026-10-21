package management

import "testing"

func TestPurchaseMoneyParsingAndFormatting(t *testing.T) {
	for _, tc := range []struct {
		input string
		kurus int64
		want  string
	}{
		{"0", 0, "0.00"}, {"0.1", 10, "0.10"}, {"0.30", 30, "0.30"},
		{"1", 100, "1.00"}, {"125.99", 12599, "125.99"}, {"9999999.99", 999999999, "9999999.99"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, ok := parsePurchaseAmount(tc.input)
			if !ok || got != tc.kurus || purchaseAmount(got) != tc.want {
				t.Fatalf("parse/format %q = %d/%t/%q; want %d/true/%q", tc.input, got, ok, purchaseAmount(got), tc.kurus, tc.want)
			}
		})
	}
	for _, input := range []string{"", " 1.00", "1.00 ", "-0.01", "+1", ".01", "1.", "01.00", "1,00", "1.001", "10000000.00", "1e2", "NaN", "Infinity", "999999999999999999999999"} {
		if _, ok := parsePurchaseAmount(input); ok {
			t.Errorf("accepted invalid monetary input %q", input)
		}
	}
	original, _ := parsePurchaseAmount("0.30")
	paid, _ := parsePurchaseAmount("0.10")
	if got := purchaseAmount(original - paid); got != "0.20" {
		t.Fatalf("exact kuruş subtraction = %q", got)
	}
}

func TestPurchaseReferenceNormalization(t *testing.T) {
	for _, input := range []string{"Branch A receipt-123", "  BRANCH A RECEIPT-123  ", "Branch\tA\nreceipt-123", "Branch\u00a0A receipt-123"} {
		if got := normalizePurchaseReference(input); got != "branch a receipt-123" {
			t.Errorf("normalize %q = %q", input, got)
		}
	}
	if normalizePurchaseReference("Branch B receipt-123") == normalizePurchaseReference("Branch A receipt-123") {
		t.Fatal("distinct branch references collide")
	}
}
