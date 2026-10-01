package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	file := flag.String("utxos", "utxos.json", "JSON list of {id, value}")
	target := flag.Int64("target", 0, "payment amount in sats")
	feeRate := flag.Float64("feerate", 5, "fee rate now (sat/vB)")
	longTerm := flag.Float64("longterm", 3, "expected long-term fee rate (sat/vB)")
	flag.Parse()
	if *target <= 0 {
		log.Fatal("-target is required")
	}

	raw, err := os.ReadFile(*file)
	if err != nil {
		log.Fatal(err)
	}
	var utxos []UTXO
	if err := json.Unmarshal(raw, &utxos); err != nil {
		log.Fatal(err)
	}
	p := Params{FeeRate: *feeRate, LongTermFeeRate: *longTerm, InputVB: 68, OutputVB: 31, OverheadVB: 11, DustLimit: 546}

	fmt.Printf("paying %d sats at %.1f sat/vB from %d UTXOs (cost of change %d)\n\n", *target, *feeRate, len(utxos), p.costOfChange())
	for _, fn := range []func([]UTXO, int64, Params) (Result, error){BranchAndBound, LargestFirst} {
		r, err := fn(utxos, *target, p)
		if err != nil {
			fmt.Printf("%v\n\n", err)
			continue
		}
		fmt.Printf("%s: %d inputs, fee %d, change %d, waste %d\n", r.Strategy, len(r.Inputs), r.Fee, r.Change, r.Waste)
		for _, u := range r.Inputs {
			fmt.Printf("  %-12s %10d\n", u.ID, u.Value)
		}
		fmt.Println()
	}
}
