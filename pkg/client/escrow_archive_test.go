package client

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

type testPokerRefereeServer struct {
	pokerrpc.UnimplementedPokerRefereeServer
	statuses map[string]*pokerrpc.GetEscrowStatusResponse
}

func (s *testPokerRefereeServer) GetEscrowStatus(_ context.Context, req *pokerrpc.GetEscrowStatusRequest) (*pokerrpc.GetEscrowStatusResponse, error) {
	resp, ok := s.statuses[req.GetEscrowId()]
	if !ok {
		return nil, status.Error(codes.NotFound, "escrow not found")
	}
	return proto.Clone(resp).(*pokerrpc.GetEscrowStatusResponse), nil
}

func TestGetBindableEscrowsIncludesLiveReadyEscrowWithStaleCache(t *testing.T) {
	pc, cleanup := newTestPokerClientWithReferee(t, map[string]*pokerrpc.GetEscrowStatusResponse{
		"escrow-stale": {
			EscrowId:              "escrow-stale",
			Ok:                    true,
			UtxoCount:             1,
			Confs:                 6,
			RequiredConfirmations: 2,
			FundingTxid:           "live-txid",
			FundingVout:           0,
			AmountAtoms:           10_000_000,
			FundingState:          "ESCROW_STATE_READY",
		},
	})
	defer cleanup()

	err := pc.CacheEscrowInfo(&EscrowInfo{
		EscrowID:        "escrow-stale",
		DepositAddress:  "TsTestAddress",
		RedeemScriptHex: "51",
		PKScriptHex:     "51",
		CSVBlocks:       64,
		Status:          "opened",
	})
	require.NoError(t, err)

	got, err := pc.GetBindableEscrows(context.Background(), "test-token")
	require.NoError(t, err)
	require.Len(t, got, 1)

	escrow := got[0]
	require.Equal(t, "escrow-stale", escrow["escrow_id"])
	require.Equal(t, "live-txid", escrow["funding_txid"])
	require.EqualValues(t, 0, escrow["funding_vout"])
	require.EqualValues(t, 10_000_000, escrow["funded_amount"])
	require.EqualValues(t, 10_000_000, escrow["amount_atoms"])
	require.EqualValues(t, 6, escrow["confs"])
	require.EqualValues(t, 2, escrow["required_confirmations"])
	require.Equal(t, "ESCROW_STATE_READY", escrow["funding_state"])
	require.Equal(t, true, escrow["ok"])
	require.Equal(t, "opened", escrow["status"])
}

func TestGetBindableEscrowsStillExcludesSpentEscrowWithStaleCache(t *testing.T) {
	pc, cleanup := newTestPokerClientWithReferee(t, map[string]*pokerrpc.GetEscrowStatusResponse{
		"escrow-spent": {
			EscrowId:     "escrow-spent",
			Ok:           true,
			UtxoCount:    0,
			FundingState: "ESCROW_STATE_SPENT",
		},
	})
	defer cleanup()

	err := pc.CacheEscrowInfo(&EscrowInfo{
		EscrowID:        "escrow-spent",
		DepositAddress:  "TsSpentAddress",
		RedeemScriptHex: "51",
		PKScriptHex:     "51",
		CSVBlocks:       64,
		Status:          "opened",
	})
	require.NoError(t, err)

	got, err := pc.GetBindableEscrows(context.Background(), "test-token")
	require.NoError(t, err)
	require.Empty(t, got)
}

func newTestPokerClientWithReferee(t *testing.T, statuses map[string]*pokerrpc.GetEscrowStatusResponse) (*PokerClient, func()) {
	t.Helper()

	logBackend, err := logging.NewLogBackend(logging.LogConfig{
		LogFile:        "",
		DebugLevel:     "error",
		MaxLogFiles:    1,
		MaxBufferLines: 100,
	})
	require.NoError(t, err)

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pokerrpc.RegisterPokerRefereeServer(srv, &testPokerRefereeServer{statuses: statuses})
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	pc := &PokerClient{
		DataDir: t.TempDir(),
		conn:    conn,
		log:     logBackend.Logger("PokerClientTest"),
	}

	cleanup := func() {
		require.NoError(t, conn.Close())
		srv.Stop()
		require.NoError(t, lis.Close())
		logBackend.Close()
	}

	return pc, cleanup
}
