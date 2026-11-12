package util_test

import (
	"testing"

	"github.com/ngbcoder/Unseal/internal/util"
)

func TestCRC32(t *testing.T) {
	// The standard check value for CRC-32 (IEEE) of "123456789".
	const check = uint32(0xCBF43926)

	if got := util.CRC32String("123456789"); got != check {
		t.Errorf("CRC32String = %#08x, want %#08x", got, check)
	}
}

func TestCRC32IsStable(t *testing.T) {
	a := util.CRC32String("alias")
	b := util.CRC32String("alias")

	if a != b {
		t.Error("CRC32 must be stable for the same input")
	}
}

func TestCRC32DistinguishesDifferentInputs(t *testing.T) {
	if util.CRC32String("alias") == util.CRC32String("alias2") {
		t.Error("CRC32 must distinguish different inputs")
	}
}
