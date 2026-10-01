package main

import (
	"errors"
	"testing"
)

var testUTXOs = []UTXO{{"a", 100000}, {"b", 50000}, {"c", 30000}, {"d", 20000}, {"e", 10000}}
var params = Params{FeeRate: 1, LongTermFeeRate: 1, InputVB: 68, OutputVB: 31, OverheadVB: 11, DustLimit: 546}

func TestBnBFindsExactMatch(t *testing.T) {
	// 50000 + 30000 minus two input fees (68 each) minus base fee for one output (42)
	r, err := BranchAndBound(testUTXOs, 79822, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Inputs) != 2 || r.Inputs[0].ID != "b" || r.Inputs[1].ID != "c" {
		t.Fatalf("unexpected inputs %+v", r.Inputs)
	}
	if r.Change != 0 || r.Waste != 0 {
		t.Fatalf("expected changeless zero-waste solution, got %+v", r)
	}
}

func TestLargestFirstMakesChange(t *testing.T) {
	r, err := LargestFirst(testUTXOs, 60000, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Inputs) != 1 || r.Change <= 0 {
		t.Fatalf("expected one input with change, got %+v", r)
	}
	if sum(r.Inputs) != 60000+r.Fee+r.Change {
		t.Fatal("amounts don't balance")
	}
}

func TestDustChangeIsDropped(t *testing.T) {
	// 100000 - 68 - base fee for 2 outputs (73) - target leaves 100 sats of change: dust
	r, err := LargestFirst(testUTXOs, 100000-68-73-100, params)
	if err != nil {
		t.Fatal(err)
	}
	if r.Change != 0 {
		t.Fatalf("dust change should be dropped, got %d", r.Change)
	}
}

func TestInsufficient(t *testing.T) {
	if _, err := LargestFirst(testUTXOs, 1_000_000, params); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("expected ErrInsufficient, got %v", err)
	}
	if _, err := BranchAndBound(testUTXOs, 1_000_000, params); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("expected ErrInsufficient, got %v", err)
	}
}
