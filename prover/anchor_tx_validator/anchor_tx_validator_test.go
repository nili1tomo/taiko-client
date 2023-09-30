package anchorTxValidator

import (
	"context"
	"testing"

	"github.com/cenkalti/backoff/v4"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"
	"github.com/taikoxyz/taiko-client/pkg/jwt"
	"github.com/taikoxyz/taiko-client/pkg/rpc"
	"github.com/taikoxyz/taiko-client/testutils"
)

type AnchorTxValidatorTestSuite struct {
	testutils.ClientSuite
	v         *AnchorTxValidator
	rpcClient *rpc.Client
}

func (s *AnchorTxValidatorTestSuite) SetupTest() {
	s.ClientSuite.SetupTest()
	jwtSecret, err := jwt.ParseSecretFromFile(testutils.JwtSecretFile)
	s.NoError(err)
	s.rpcClient, err = rpc.NewClient(context.Background(), &rpc.ClientConfig{
		L1Endpoint:        s.L1.WsEndpoint(),
		L2Endpoint:        s.L2.WsEndpoint(),
		TaikoL1Address:    s.L1.TaikoL1Address,
		TaikoTokenAddress: s.L1.TaikoL1TokenAddress,
		TaikoL2Address:    testutils.TaikoL2Address,
		L2EngineEndpoint:  s.L2.AuthEndpoint(),
		JwtSecret:         string(jwtSecret),
		RetryInterval:     backoff.DefaultMaxInterval,
	})
	s.NoError(err)
	validator, err := New(testutils.TaikoL2Address, s.rpcClient.L2ChainID, s.rpcClient)
	s.Nil(err)
	s.v = validator
}

func (s *AnchorTxValidatorTestSuite) TearDownTest() {
	s.rpcClient.Close()
	s.ClientSuite.TearDownTest()
}

func (s *AnchorTxValidatorTestSuite) TestValidateAnchorTx() {
	wrongPrivKey, err := crypto.HexToECDSA("2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200")
	s.Nil(err)

	goldenTouchPrivKey, err := s.rpcClient.TaikoL2.GOLDENTOUCHPRIVATEKEY(nil)
	s.Nil(err)

	// 0x92954368afd3caa1f3ce3ead0069c1af414054aefe1ef9aeacc1bf426222ce38
	goldenTouchPriKey, err := crypto.HexToECDSA(common.Bytes2Hex(goldenTouchPrivKey.Bytes()))
	s.Nil(err)

	// invalid To
	tx := types.NewTransaction(
		0,
		common.BytesToAddress(testutils.RandomBytes(1024)), common.Big0, 0, common.Big0, []byte{},
	)
	s.ErrorContains(s.v.ValidateAnchorTx(context.Background(), tx), "invalid TaikoL2.anchor transaction to")

	// invalid sender
	dynamicFeeTxTx := &types.DynamicFeeTx{
		ChainID:    s.v.rpc.L2ChainID,
		Nonce:      0,
		GasTipCap:  common.Big1,
		GasFeeCap:  common.Big1,
		Gas:        1,
		To:         &s.v.taikoL2Address,
		Value:      common.Big0,
		Data:       []byte{},
		AccessList: types.AccessList{},
	}

	signer := types.LatestSignerForChainID(s.v.rpc.L2ChainID)
	tx = types.MustSignNewTx(wrongPrivKey, signer, dynamicFeeTxTx)

	s.ErrorContains(
		s.v.ValidateAnchorTx(context.Background(), tx), "invalid TaikoL2.anchor transaction sender",
	)

	// invalid method selector
	tx = types.MustSignNewTx(goldenTouchPriKey, signer, dynamicFeeTxTx)
	s.ErrorContains(s.v.ValidateAnchorTx(context.Background(), tx), "invalid TaikoL2.anchor transaction selector")
}

func (s *AnchorTxValidatorTestSuite) TestGetAndValidateAnchorTxReceipt() {
	tx := types.NewTransaction(
		100,
		common.BytesToAddress(testutils.RandomBytes(32)),
		common.Big1,
		100000,
		common.Big1,
		[]byte{},
	)
	_, err := s.v.GetAndValidateAnchorTxReceipt(context.Background(), tx)
	s.NotNil(err)
}

func TestAnchorTxValidatorTestSuite(t *testing.T) {
	suite.Run(t, new(AnchorTxValidatorTestSuite))
}
