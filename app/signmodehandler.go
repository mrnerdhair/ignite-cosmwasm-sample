package app

import (
	"context"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	txsigning "cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/textual"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/fxamacker/cbor"
)

type signDoc struct {
	Screens []screen `cbor:"1,keyasint,omitempty"`
}

type screen struct {
	Title   string `cbor:"1,keyasint,omitempty"`
	Content string `cbor:"2,keyasint,omitempty"`
	Indent  int    `cbor:"3,keyasint,omitempty"`
	Expert  bool   `cbor:"4,keyasint,omitempty"`
}

type EIP191TextualSignModeHandler struct {
	innerHandler txsigning.SignModeHandler
}

var (
	_ txsigning.SignModeHandler = EIP191TextualSignModeHandler{}
)

func NewEIP191TextualSignModeHandler(innerHandler txsigning.SignModeHandler) EIP191TextualSignModeHandler {
	return EIP191TextualSignModeHandler{
		innerHandler,
	}
}

func (h EIP191TextualSignModeHandler) Mode() signingv1beta1.SignMode {
	return h.innerHandler.Mode()
}

func (h EIP191TextualSignModeHandler) GetSignBytes(ctx context.Context, signerData txsigning.SignerData, txData txsigning.TxData) ([]byte, error) {
	mode := h.innerHandler.Mode()
	if mode != signingv1beta1.SignMode_SIGN_MODE_TEXTUAL {
		return nil, fmt.Errorf("unsupported sign mode %s", mode)
	}

	cborBytes, err := h.innerHandler.GetSignBytes(ctx, signerData, txData)
	if err != nil {
		return nil, err
	}

	cborSignDoc := signDoc{}
	err = cbor.Unmarshal(cborBytes, &cborSignDoc)
	if err != nil {
		return nil, err
	}

	lines := []string{}
	for _, screen := range cborSignDoc.Screens {
		line := ""
		if screen.Expert {
			line += "*"
		}
		line += strings.Repeat("\t", screen.Indent)
		if len(screen.Title) > 0 {
			line += screen.Title + ": "
		}
		line += screen.Content
		lines = append(lines, line)
	}
	signBytes := []byte(strings.Join(lines, "\n"))

	out := slices.Concat(
		[]byte{0x19},
		[]byte("Ethereum Signed Message:\n"),
		[]byte(fmt.Sprintf("%d", len(signBytes))),
		signBytes,
	)

	fmt.Println(hex.EncodeToString(out))

	return out, nil
}

func ProvideCustomSignModeHandlers(bk bankkeeper.Keeper) func() []txsigning.SignModeHandler {
	innerHandler, err := textual.NewSignModeHandler(textual.SignModeOptions{
		CoinMetadataQuerier: authtxconfig.NewBankKeeperCoinMetadataQueryFn(bk),
	})
	if err != nil {
		panic(err)
	}
	return func() []txsigning.SignModeHandler {
		return []txsigning.SignModeHandler{
			NewEIP191TextualSignModeHandler(innerHandler),
		}
	}
}
