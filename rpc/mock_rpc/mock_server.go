package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/iost-official/go-iost/ilog"
	"github.com/iost-official/go-iost/rpc/pb"
	"google.golang.org/grpc"
)

var (
	grpcAddr    = "0.0.0.0:20000"
	gatewayAddr = "0.0.0.0:20001"
)

// Server is the rpc server including grpc server and json gateway server.
type Server struct {
	grpcServer    *grpc.Server
	gatewayServer *http.Server
}

func newMockAPI() rpcpb.ApiServiceServer {
	api := NewMockApiServiceServer(gomock.NewController(&testing.T{}))

	api.EXPECT().GetNodeInfo(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.NodeInfoResponse{
		BuildTime: "111",
		GitHash:   "222",
		Mode:      "333",
		Network: &rpcpb.NetworkInfo{
			Id:        "444",
			PeerCount: 555,
			PeerInfo: []*rpcpb.PeerInfo{
				{Id: "id1", Addr: "addr1"},
				{Id: "id2", Addr: "addr2"},
			},
		},
	}, nil)

	api.EXPECT().GetChainInfo(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.ChainInfoResponse{
		NetName:         "testnet",
		ProtocolVersion: "1.0.0",
		HeadBlock:       2,
		HeadBlockHash:   "lsodsafdsf",
		LibBlock:        1,
		LibBlockHash:    "sadfasdfa",
		WitnessList:     []string{"bp1", "bp2"},
	}, nil)

	api.EXPECT().GetTxByHash(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.TransactionResponse{
		Status: rpcpb.TransactionResponse_PENDIND,
		Transaction: &rpcpb.Transaction{
			Hash:       "xxxxx",
			Time:       999,
			Expiration: 20,
			GasRatio:   1.01,
			GasLimit:   5.05,
			Delay:      0,
			Actions: []*rpcpb.Action{
				{Contract: "c1", ActionName: "a1", Data: "d1"},
				{Contract: "c2", ActionName: "a2", Data: "d2"},
			},
			Signers:    []string{"s1", "s2"},
			Publisher:  "publisher",
			ReferredTx: "ccc",
			AmountLimit: []*rpcpb.AmountLimit{
				{Token: "iost", Value: 12.2},
				{Token: "10st", Value: 21.1},
			},
			TxReceipt: &rpcpb.TxReceipt{
				TxHash:     "xxx",
				GasUsage:   222.1,
				RamUsage:   map[string]int64{"aaa": 222},
				StatusCode: rpcpb.TxReceipt_SUCCESS,
				Message:    "success",
				Returns:    []string{"aa", "cc"},
				Receipts: []*rpcpb.TxReceipt_Receipt{
					{FuncName: "transfer", Content: "aaaa"},
					{FuncName: "Transfer", Content: "bbbb"},
				},
			},
		},
	}, nil)

	api.EXPECT().GetTxReceiptByTxHash(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.TxReceipt{
		TxHash:     "xxx",
		GasUsage:   222.1,
		RamUsage:   map[string]int64{"aaa": 222},
		StatusCode: rpcpb.TxReceipt_SUCCESS,
		Message:    "success",
		Returns:    []string{"aa", "cc"},
		Receipts: []*rpcpb.TxReceipt_Receipt{
			{FuncName: "transfer", Content: "aaaa"},
			{FuncName: "Transfer", Content: "bbbb"},
		},
	}, nil)

	api.EXPECT().GetBlockByHash(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.BlockResponse{
		Status: rpcpb.BlockResponse_PENDIND,
		Block: &rpcpb.Block{
			Hash:                "hhh",
			Version:             1,
			ParentHash:          "pppp",
			TxMerkleHash:        "ttt",
			TxReceiptMerkleHash: "mmm",
			Number:              3,
			Witness:             "bp3",
			Time:                2342342,
			GasUsage:            500.2,
			TxCount:             1,
			Info: &rpcpb.Block_Info{
				Mode:       0,
				Thread:     1,
				BatchIndex: []int32{},
			},
			Transactions: []*rpcpb.Transaction{
				{
					Hash:       "xxxxx",
					Time:       999,
					Expiration: 20,
					GasRatio:   1.01,
					GasLimit:   5.05,
					Delay:      0,
					Actions: []*rpcpb.Action{
						{Contract: "c1", ActionName: "a1", Data: "d1"},
						{Contract: "c2", ActionName: "a2", Data: "d2"},
					},
					Signers:    []string{"s1", "s2"},
					Publisher:  "publisher",
					ReferredTx: "ccc",
					AmountLimit: []*rpcpb.AmountLimit{
						{Token: "iost", Value: 12.2},
						{Token: "10st", Value: 21.1},
					},
					TxReceipt: &rpcpb.TxReceipt{
						TxHash:     "xxx",
						GasUsage:   222.1,
						RamUsage:   map[string]int64{"aaa": 222},
						StatusCode: rpcpb.TxReceipt_SUCCESS,
						Message:    "success",
						Returns:    []string{"aa", "cc"},
						Receipts: []*rpcpb.TxReceipt_Receipt{
							{FuncName: "transfer", Content: "aaaa"},
							{FuncName: "Transfer", Content: "bbbb"},
						},
					},
				},
			},
		},
	}, nil)

	api.EXPECT().GetBlockByNumber(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.BlockResponse{
		Status: rpcpb.BlockResponse_PENDIND,
		Block: &rpcpb.Block{
			Hash:                "hhh",
			Version:             1,
			ParentHash:          "pppp",
			TxMerkleHash:        "ttt",
			TxReceiptMerkleHash: "mmm",
			Number:              3,
			Witness:             "bp3",
			Time:                2342342,
			GasUsage:            500.2,
			TxCount:             1,
			Info: &rpcpb.Block_Info{
				Mode:       0,
				Thread:     1,
				BatchIndex: []int32{},
			},
			Transactions: []*rpcpb.Transaction{
				{
					Hash:       "xxxxx",
					Time:       999,
					Expiration: 20,
					GasRatio:   1.01,
					GasLimit:   5.05,
					Delay:      0,
					Actions: []*rpcpb.Action{
						{Contract: "c1", ActionName: "a1", Data: "d1"},
						{Contract: "c2", ActionName: "a2", Data: "d2"},
					},
					Signers:    []string{"s1", "s2"},
					Publisher:  "publisher",
					ReferredTx: "ccc",
					AmountLimit: []*rpcpb.AmountLimit{
						{Token: "iost", Value: 12.2},
						{Token: "10st", Value: 21.1},
					},
					TxReceipt: &rpcpb.TxReceipt{
						TxHash:     "xxx",
						GasUsage:   222.1,
						RamUsage:   map[string]int64{"aaa": 222},
						StatusCode: rpcpb.TxReceipt_SUCCESS,
						Message:    "success",
						Returns:    []string{"aa", "cc"},
						Receipts: []*rpcpb.TxReceipt_Receipt{
							{FuncName: "transfer", Content: "aaaa"},
							{FuncName: "Transfer", Content: "bbbb"},
						},
					},
				},
			},
		},
	}, nil)

	api.EXPECT().GetAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.Account{
		Name:       "admin",
		Balance:    20000.3,
		CreateTime: 0,
		GasInfo: &rpcpb.Account_GasInfo{
			CurrentTotal:  1000.2,
			IncreaseSpeed: 20.3,
			Limit:         100,
			PledgedInfo: []*rpcpb.Account_PledgeInfo{
				{Pledger: "root", Amount: 2000.3},
			},
		},
		RamInfo: &rpcpb.Account_RAMInfo{Available: 111111},
		Permissions: map[string]*rpcpb.Account_Permission{
			"owner": {
				Name:   "owner",
				Groups: []string{"active", "owner"},
				Items: []*rpcpb.Account_Item{
					{Id: "aaa", IsKeyPair: false, Weight: 1, Permission: "aaa"},
					{Id: "bbb", IsKeyPair: true, Weight: 2, Permission: "bbbb"},
				},
				Threshold: 3,
			},
		},
		Groups: map[string]*rpcpb.Account_Group{
			"group1": {
				Name: "group1",

				Items: []*rpcpb.Account_Item{
					{Id: "aaa", IsKeyPair: false, Weight: 1, Permission: "aaa"},
					{Id: "bbb", IsKeyPair: true, Weight: 2, Permission: "bbbb"},
				},
			},
		},
		FrozenBalances: []*rpcpb.FrozenBalance{
			{Amount: 111.2, Time: 2343242},
		},
	}, nil)

	api.EXPECT().GetTokenBalance(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.GetTokenBalanceResponse{
		Balance: 20000.3,
		FrozenBalances: []*rpcpb.FrozenBalance{
			{Amount: 111.2, Time: 2343242},
		},
	}, nil)

	api.EXPECT().GetContract(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.Contract{
		Id:       "Contract12312131",
		Code:     "print helloworld",
		Language: "javascript",
		Version:  "1.0",
		Abis: []*rpcpb.Contract_ABI{
			{
				Name: "echo",
				Args: []string{"a1", "a2"},
				AmountLimit: []*rpcpb.AmountLimit{
					{Token: "iost", Value: 1212.2},
				},
			},
		},
	}, nil)

	api.EXPECT().GetContractStorage(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.GetContractStorageResponse{
		Data: `{"key":"value"}`,
	}, nil)

	api.EXPECT().SendTransaction(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.SendTransactionResponse{
		Hash: "12131",
	}, nil)

	api.EXPECT().ExecTransaction(gomock.Any(), gomock.Any()).AnyTimes().Return(&rpcpb.TxReceipt{
		TxHash:     "xxx",
		GasUsage:   222.1,
		RamUsage:   map[string]int64{"aaa": 222},
		StatusCode: rpcpb.TxReceipt_SUCCESS,
		Message:    "success",
		Returns:    []string{"aa", "cc"},
		Receipts: []*rpcpb.TxReceipt_Receipt{
			{FuncName: "transfer", Content: "aaaa"},
			{FuncName: "Transfer", Content: "bbbb"},
		},
	}, nil)

	return api
}

// New returns a new rpc server instance.
func New() *Server {
	s := &Server{}
	s.grpcServer = grpc.NewServer()
	apiService := newMockAPI()
	rpcpb.RegisterApiServiceServer(s.grpcServer, apiService)
	return s
}

// Start starts the rpc server.
func (s *Server) Start() error {
	ilog.Info("start mock rpc server")
	if err := s.startGrpc(); err != nil {
		return err
	}
	return s.startGateway()
}

func (s *Server) startGrpc() error {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			ilog.Fatalf("start grpc failed. err=%v", err)
		}
	}()
	return nil
}

func (s *Server) startGateway() error {
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard,
		&runtime.JSONPb{OrigName: true, EmitDefaults: true}))
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := rpcpb.RegisterApiServiceHandlerFromEndpoint(context.Background(), mux, grpcAddr, opts)
	if err != nil {
		return err
	}
	s.gatewayServer = &http.Server{
		Addr:    gatewayAddr,
		Handler: mux,
	}
	go func() {
		if err := s.gatewayServer.ListenAndServe(); err != http.ErrServerClosed {
			ilog.Fatalf("start gateway failed. err=%v", err)
		}
	}()
	return nil
}

// Stop stops the rpc server.
func (s *Server) Stop() {
	ilog.Info("stop mock rpc server")
	s.gatewayServer.Shutdown(nil)
	s.grpcServer.GracefulStop()
}

func waitExit() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	i := <-c
	ilog.Infof("receive exit signal: %v", i)
}

func main() {
	server := New()
	if err := server.Start(); err != nil {
		panic(err)
	}
	waitExit()
	server.Stop()
}
