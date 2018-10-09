package tester

import (
	"encoding/base64"
	"fmt"
	"github.com/andrecronje/lachesis/src/peers"
	"github.com/andrecronje/lachesis/src/proxy/lachesis"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func PingNodesN(participants []*peers.Peer, p peers.PubKeyPeers, n uint64, serviceAddress string) {
	txId := UniqueID{counter: 1}

	wg := new(sync.WaitGroup)
	fmt.Println("PingNodesN::participants: ", participants)
	fmt.Println("PingNodesN::p: ", p)
	for i := uint64(0); i < n; i++ {
		wg.Add(1)
		participant := participants[rand.Intn(len(participants))]
		node := p[participant.PubKeyHex]

		ipAddr, err := transact(*participant, node.ID, txId, serviceAddress)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
			fmt.Printf("Fatal error:\t\t\t%s\n", err.Error())
			if ipAddr != "" {
				fmt.Printf("Failed to ping:\t\t\t%s (id=%d)\n", ipAddr, node)
			} else {
				fmt.Printf("Failed to ping:\t\t\tid=%d\n", node)
			}
			fmt.Printf("Failed to send transaction:\t%d\n\n", txId.Get()-1)
		} else {
			fmt.Printf("Pinged:\t\t\t%s (id=%d)\n", ipAddr, node)
			fmt.Printf("Last transaction sent:\t%d\n\n", txId.Get()-1)
		}

		time.Sleep(1600 * time.Millisecond)
	}

	fmt.Println("Pinging stopped")

	wg.Wait()
}

func sendTransaction(target peers.Peer) {
	ip := &layers.IPv4{
		SrcIP: GetOutboundIP(),
		DstIP: net.IP(target.NetAddr),
		// etc...
	}

	// TODO: Make shared counter for Tx #
	// TODO: Make shared counter for Node #
	payload := fmt.Sprintf("%s{\"method\":\"Lachesis.SubmitTx\",\"params\":[\"whatever\"],\"id\":\"whatever\"}",
		base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Node%d Tx%d"))))

	net.Dial("tcp", ip.DstIP.String())

	buf := gopacket.NewSerializeBufferExpectedSize(len(payload), 0)
	opts := gopacket.SerializeOptions{} // See SerializeOptions for more details.
	err := ip.SerializeTo(buf, opts)
	if err != nil {
		panic(err)
	}
	fmt.Println(buf.Bytes()) // prints out a byte slice containing
}

// https://stackoverflow.com/a/37382208
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func transact(target peers.Peer, nodeId int, txId UniqueID, proxyAddress string) (string, error) {
	addr := fmt.Sprintf("%s:%d", strings.Split(target.NetAddr, ":")[0], 9000)
	proxy := lachesis.NewSocketLachesisProxyClient(addr, 10 * time.Second)

	_, err := proxy.SubmitTx([]byte("oh hai"))
	// fmt.Println("Submitted tx, ack=", ack)  # `ack` is now `_`

	return "hi", err
}

type UniqueID struct {
	counter uint64
}

func (c *UniqueID) Get() uint64 {
	for {
		val := atomic.LoadUint64(&c.counter)
		if atomic.CompareAndSwapUint64(&c.counter, val, val+1) {
			return val
		}
	}
}
