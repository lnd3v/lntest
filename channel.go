package lntest

import (
	"time"

	"github.com/breez/lntest/core_lightning"
	"golang.org/x/exp/slices"
)

const defaultTimeout int = 10

type ChannelInfo struct {
	From        *LightningNode
	To          *LightningNode
	FundingTx   string
	FundingTxId string
	ChannelId   string
}

func (c *ChannelInfo) WaitForChannelReady() {
	timeout := time.Now().Add(time.Duration(defaultTimeout) * time.Second)
	for {
		info, err := c.To.rpc.Getinfo(c.From.harness.ctx, &core_lightning.GetinfoRequest{})
		CheckError(c.To.harness.T, err)

		peers, err := c.From.rpc.ListPeers(c.From.harness.ctx, &core_lightning.ListpeersRequest{
			Id: info.Id,
		})
		CheckError(c.From.harness.T, err)

		if len(peers.Peers) == 0 {
			c.From.harness.T.Fatalf("Peer %s not found", string(info.Id))
		}

		peer := peers.Peers[0]
		if peer.Channels == nil {
			c.From.harness.T.Fatal("no channels for peer")
		}

		channelIndex := slices.IndexFunc(
			peer.Channels,
			func(pc *core_lightning.ListpeersPeersChannels) bool {
				return string(pc.ChannelId) == c.ChannelId
			},
		)

		if channelIndex >= 0 {
			if peer.Channels[channelIndex].State == core_lightning.ListpeersPeersChannels_CHANNELD_NORMAL {
				return
			}
		}

		if time.Now().After(timeout) {
			c.From.harness.T.Fatal("timed out waiting for channel normal")
		}

		time.Sleep(50 * time.Millisecond)
	}
}
