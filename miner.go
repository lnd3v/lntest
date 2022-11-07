package lntest

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/niftynei/glightning/gbitcoin"
)

type Miner struct {
	harness *TestHarness
	dir     string
	rpc     *gbitcoin.Bitcoin
	rpcPort uint32
	cmd     *exec.Cmd
}

func NewMiner(h *TestHarness) *Miner {
	btcUser := "btcuser"
	btcPass := "btcpass"
	bitcoindDir := h.GetDirectory("miner")
	rpcPort, err := GetPort()
	CheckError(h.T, err)

	binary, err := GetBitcoindBinary()
	CheckError(h.T, err)

	args := []string{
		"-regtest",
		"-server",
		"-logtimestamps",
		"-nolisten",
		"-addresstype=bech32",
		"-txindex",
		"-fallbackfee=0.00000253",
		fmt.Sprintf("-datadir=%s", bitcoindDir),
		fmt.Sprintf("-rpcport=%d", rpcPort),
		fmt.Sprintf("-rpcpassword=%s", btcPass),
		fmt.Sprintf("-rpcuser=%s", btcUser),
	}

	log.Printf("starting %s on rpc port %d in dir %s...", binary, rpcPort, bitcoindDir)
	cmd := exec.CommandContext(h.Ctx, binary, args...)

	err = cmd.Start()
	CheckError(h.T, err)
	log.Printf("bitcoind started (%d)!", cmd.Process.Pid)

	rpc := gbitcoin.NewBitcoin(btcUser, btcPass)
	rpc.SetTimeout(uint(2))

	log.Printf("Starting up bitcoin client")
	rpc.StartUp("http://localhost", bitcoindDir, uint(rpcPort))

	// Go ahead and run 50 blocks
	log.Printf("Get new address")
	addr, err := rpc.GetNewAddress(gbitcoin.Bech32)
	CheckError(h.T, err)

	log.Printf("Generate to address")
	_, err = rpc.GenerateToAddress(addr, 200)
	CheckError(h.T, err)

	miner := &Miner{
		harness: h,
		dir:     bitcoindDir,
		cmd:     cmd,
		rpc:     rpc,
		rpcPort: rpcPort,
	}

	h.AddStoppable(miner)
	h.RegisterLogfile(filepath.Join(bitcoindDir, "regtest", "debug.log"), filepath.Base(bitcoindDir))
	return miner
}

func (m *Miner) MineBlocks(n uint) {
	addr, err := m.rpc.GetNewAddress(gbitcoin.Bech32)
	CheckError(m.harness.T, err)
	_, err = m.rpc.GenerateToAddress(addr, n)
	CheckError(m.harness.T, err)
}

func (m *Miner) SendToAddress(addr string, amountSat uint64) {
	amountBtc := amountSat / uint64(100000000)
	amountSatRemainder := amountSat % 100000000
	amountStr := strconv.FormatUint(amountBtc, 10) + "." + fmt.Sprintf("%08s", strconv.FormatUint(amountSatRemainder, 10))
	log.Printf("Sending %s btc to address %s", amountStr, addr)
	_, err := m.rpc.SendToAddress(addr, amountStr)
	CheckError(m.harness.T, err)

	m.MineBlocks(1)
}

func (m *Miner) GetBlockHeight() uint32 {
	info, err := m.rpc.GetChainInfo()
	CheckError(m.harness.T, err)
	return info.Blocks
}

func (m *Miner) TearDown() error {
	if m.cmd == nil || m.cmd.Process == nil {
		// return if not properly initialized
		// or error starting the process
		return nil
	}

	defer m.cmd.Wait()
	if runtime.GOOS == "windows" {
		return m.cmd.Process.Signal(os.Kill)
	}

	return m.cmd.Process.Signal(os.Interrupt)
}
