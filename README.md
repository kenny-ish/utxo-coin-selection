# utxo-coin-selection

Coin selection for a Bitcoin wallet, using the two strategies most wallets combine.

1. Branch and Bound (Murch's algorithm, also used by Bitcoin Core). A depth-first search for a set of
   inputs whose effective value falls between the target and the target plus the cost of a change
   output. If it finds one, the transaction needs no change output.
2. Largest first as the fallback. It adds the biggest coins until the payment, the fee and a change
   output are covered, and drops the change if it would be dust.

Every result gets a waste score:

```
waste = sum over inputs of (fee_rate - long_term_fee_rate) * input_vbytes
      + cost_of_change            (if a change output is created)
      + excess                    (if there's no change and we overpay)
```

```bash
go run . -utxos utxos.json -target 79822 -feerate 1
go run . -utxos utxos.json -target 150000 -feerate 12 -longterm 5
```

`utxos.json` is a list of `{"id": "txid:vout", "value": sats}`. Sizes default to P2WPKH: 68 vB per
input, 31 vB per output and 11 vB of overhead.

```bash
go test ./...
```
