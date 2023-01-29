package prose

import "testing"

func TestTwoElements(t *testing.T) {
	list := []string{"mela", "arancia"}
	right := "mela e arancia"
	left := JoinWithCommas(list)
	if left != right {
		t.Errorf("valore aspettato non matchato: %v != %v", left, right)

	}
}
func TestThreeElements(t *testing.T) {
	list := []string{"mela", "arancia", "pera"}
	right := "mela, arancia e pera"
	left := JoinWithCommas(list)
	if left != right {
		t.Errorf("valore aspettato non matchato: %v != %v", left, right)

	}
}
