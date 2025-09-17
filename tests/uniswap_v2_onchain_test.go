package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"bigswapenergy/internal/shared/utils"
)

func TestGetAmountOut_Onchain(t *testing.T) {
	rpcURL := os.Getenv("ETHEREUM_RPC_URL")
	if rpcURL == "" {
		t.Skip("ETHEREUM_RPC_URL not set; skipping on-chain comparison test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		t.Fatalf("dial eth rpc: %v", err)
	}

	abiPath := filepath.Join("abi", "abi.json")
	data, err := os.ReadFile(abiPath)
	if err != nil {
		t.Fatalf("read abi: %v", err)
	}
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("parse abi json: %v", err)
	}
	var methodEntries []map[string]any
	for _, e := range arr {
		if name, _ := e["name"].(string); name == "getAmountOut" {
			methodEntries = append(methodEntries, e)
		}
	}
	if len(methodEntries) == 0 {
		t.Fatalf("getAmountOut not found in ABI file")
	}
	minimalJSON, err := json.Marshal(methodEntries)
	if err != nil {
		t.Fatalf("marshal minimal abi: %v", err)
	}
	contractABI, err := gethabi.JSON(bytes.NewReader(minimalJSON))
	if err != nil {
		t.Fatalf("parse abi: %v", err)
	}

	router := common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")

	cases := []struct {
		name       string
		amountIn   *big.Int
		reserveIn  *big.Int
		reserveOut *big.Int
	}{
		{"small_balanced", big.NewInt(1_000), big.NewInt(1_000_000), big.NewInt(1_000_000)},
		{"skewed_reserves", big.NewInt(50_000_000_000_000), new(big.Int).SetUint64(5_000_000_000_000_000), new(big.Int).SetUint64(100_000_000_000_000_000)},
		{"large_values", new(big.Int).SetUint64(1_000_000_000_000_000), new(big.Int).SetUint64(50_000_000_000_000_000), new(big.Int).SetUint64(75_000_000_000_000_000)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			amountOut := utils.GlobalBigIntPool.Get()
			defer utils.GlobalBigIntPool.Put(amountOut)
			utils.CalculateUniswapV2SwapAmount(tc.amountIn, tc.reserveIn, tc.reserveOut, amountOut, utils.GlobalBigIntPool)

			input, err := contractABI.Pack("getAmountOut", tc.amountIn, tc.reserveIn, tc.reserveOut)
			if err != nil {
				t.Fatalf("abi pack: %v", err)
			}

			call := ethereum.CallMsg{To: &router, Data: input}
			out, err := client.CallContract(ctx, call, nil)
			if err != nil {
				t.Fatalf("eth_call getAmountOut: %v", err)
			}
			values, err := contractABI.Unpack("getAmountOut", out)
			if err != nil {
				t.Fatalf("abi unpack: %v", err)
			}
			if len(values) != 1 {
				t.Fatalf("unexpected outputs: %d", len(values))
			}
			onchain, ok := values[0].(*big.Int)
			if !ok {
				t.Fatalf("unexpected output type: %T", values[0])
			}

			if amountOut.Cmp(onchain) != 0 {
				t.Fatalf("mismatch: local=%s onchain=%s (in=%s rIn=%s rOut=%s)", amountOut, onchain, tc.amountIn, tc.reserveIn, tc.reserveOut)
			}
		})
	}
}
