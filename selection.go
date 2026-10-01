package main

import (
	"errors"
	"math"
	"sort"
)

type UTXO struct {
	ID    string `json:"id"`
	Value int64  `json:"value"`
}

type Params struct {
	FeeRate         float64 // sat/vB now
	LongTermFeeRate float64 // sat/vB we expect to pay later
	InputVB         float64
	OutputVB        float64
	OverheadVB      float64
	DustLimit       int64
}

type Result struct {
	Strategy string
	Inputs   []UTXO
	Fee      int64
	Change   int64
	Waste    int64
}

var ErrInsufficient = errors.New("insufficient funds")

func ceil(x float64) int64 { return int64(math.Ceil(x)) }

func (p Params) inputFee() int64           { return ceil(p.FeeRate * p.InputVB) }
func (p Params) baseFee(outputs int) int64 { return ceil(p.FeeRate * (p.OverheadVB + p.OutputVB*float64(outputs))) }
func (p Params) effective(u UTXO) int64    { return u.Value - p.inputFee() }
func (p Params) costOfChange() int64 {
	return ceil(p.FeeRate*p.OutputVB) + ceil(p.LongTermFeeRate*p.InputVB)
}

func (p Params) inputWaste(n int) int64 {
	return ceil((p.FeeRate - p.LongTermFeeRate) * p.InputVB * float64(n))
}

func sum(us []UTXO) int64 {
	var s int64
	for _, u := range us {
		s += u.Value
	}
	return s
}

// BranchAndBound looks for a changeless input set.
func BranchAndBound(utxos []UTXO, target int64, p Params) (Result, error) {
	var pool []UTXO
	for _, u := range utxos {
		if p.effective(u) > 0 {
			pool = append(pool, u)
		}
	}
	sort.Slice(pool, func(i, j int) bool { return pool[i].Value > pool[j].Value })
	selTarget := target + p.baseFee(1)
	upper := selTarget + p.costOfChange()

	remaining := make([]int64, len(pool)+1) // suffix sums of effective values
	for i := len(pool) - 1; i >= 0; i-- {
		remaining[i] = remaining[i+1] + p.effective(pool[i])
	}
	if remaining[0] < selTarget {
		return Result{}, ErrInsufficient
	}

	var best []int
	bestWaste := int64(math.MaxInt64)
	var chosen []int
	tries := 0
	var dfs func(i int, eff int64)
	dfs = func(i int, eff int64) {
		tries++
		if tries > 100_000 || eff > upper || eff+remaining[i] < selTarget {
			return
		}
		if eff >= selTarget {
			w := p.inputWaste(len(chosen)) + (eff - selTarget)
			if w < bestWaste {
				bestWaste = w
				best = append([]int(nil), chosen...)
			}
			return
		}
		if i == len(pool) {
			return
		}
		chosen = append(chosen, i)
		dfs(i+1, eff+p.effective(pool[i]))
		chosen = chosen[:len(chosen)-1]
		dfs(i+1, eff)
	}
	dfs(0, 0)
	if best == nil {
		return Result{}, errors.New("no changeless solution")
	}
	inputs := make([]UTXO, len(best))
	for k, idx := range best {
		inputs[k] = pool[idx]
	}
	return Result{Strategy: "branch-and-bound", Inputs: inputs, Fee: sum(inputs) - target, Waste: bestWaste}, nil
}

// LargestFirst adds the biggest coins until payment + fee (+ change) is covered.
func LargestFirst(utxos []UTXO, target int64, p Params) (Result, error) {
	sorted := append([]UTXO(nil), utxos...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Value > sorted[j].Value })
	var inputs []UTXO
	var eff int64
	need := target + p.baseFee(2)
	for _, u := range sorted {
		if eff >= need {
			break
		}
		if p.effective(u) <= 0 {
			continue
		}
		inputs = append(inputs, u)
		eff += p.effective(u)
	}
	if eff < need {
		return Result{}, ErrInsufficient
	}
	fee := p.baseFee(2) + p.inputFee()*int64(len(inputs))
	change := sum(inputs) - target - fee
	waste := p.inputWaste(len(inputs)) + p.costOfChange()
	if change < p.DustLimit {
		fee = sum(inputs) - target
		waste = p.inputWaste(len(inputs)) + change
		change = 0
	}
	return Result{Strategy: "largest-first", Inputs: inputs, Fee: fee, Change: change, Waste: waste}, nil
}
