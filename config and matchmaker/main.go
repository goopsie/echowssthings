package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/klauspost/compress/zstd"
)

var configMainMenu, _ = hex.DecodeString("7b7b2274797065223a226d61696e5f6d656e75222c226964223a226d61696e5f6d656e75227d00")
var matchmakeMMenu, _ = hex.DecodeString("7b2267616d6574797065223a3330313036393334363835313930313330322c226170706964223a2231333639303738343039383733343032227d")
var configStage2, _ = hex.DecodeString("7b2274797065223a226163746976655f626174746c655f706173735f736561736f6e222c226964223a22626174746c655f706173735f736561736f6e5f30305f696e76616c6964227d")
var matchmakerServerList, _ = hex.DecodeString("f640bb78a2e78cbb4fae333004d04760")
var upgrader = websocket.Upgrader{}

func main() {
	csh := http.NewServeMux()
	csh.HandleFunc("/", config)
	msh := http.NewServeMux()
	msh.HandleFunc("/", matchmaking)
	tsh := http.NewServeMux()
	tsh.HandleFunc("/", transaction)

	go http.ListenAndServe("0.0.0.0:8001", msh)
	go http.ListenAndServe("0.0.0.0:8002", tsh) // i don't think we have websocket data for this ):
	http.ListenAndServe("0.0.0.0:8003", csh)
}

func config(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			continue
		}
		fmt.Println("Message in config:")
		fmt.Println(hex.Dump(message))
		headerPrefix, _ := hex.DecodeString("f640bb78a2e78cbb12d07b6f58afcdb9")
		headerSuffix, _ := hex.DecodeString("7b1d0e4427ee09157b1d0e4427ee0915")

		switch {
		case bytes.Contains(message, configMainMenu):
			resBytes := constructZSTDPacket("./json/config_echopass_textures.json", headerPrefix, headerSuffix)
			c.WriteMessage(mt, resBytes)
			resBytes, _ = hex.DecodeString("f640bb78a2e78cbbe4ee6bc73a96e643010000000000000010")
			c.WriteMessage(mt, resBytes)
			log.Printf("Sent config_echopass_textures.json")

		case bytes.Contains(message, configStage2):
			resBytes, _ := hex.DecodeString("")
			c.WriteMessage(mt, resBytes)
			fmt.Printf("Config Replying with: \n %s\n", hex.Dump(resBytes))
		}
	}
}

func matchmaking(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			continue
		}
		fmt.Println("Message in Matchmaking:")
		fmt.Println(hex.Dump(message))

		switch {
		case bytes.Contains(message, matchmakerServerList):
			mlen := len(message)
			servBytes := message[(32) : mlen-4] // send back only server (WILL BREAK IF >1 SERVER IS RETURNED. WILL FIX LATER, SORT BY PING INSTEAD OF DISCARD.)
			// will have to len() final data after magic number, 24 bytes in?
			prefix1, _ := hex.DecodeString("f640bb78a2e78cbb0e11e30e65e34d6d080100000000000076cfddcff99c2d044ab45615d403ea4bbff73bcc25673b85")
			prefix2, _ := hex.DecodeString("f640bb78a2e78cbb0f11e30e65e34d6d180100000000000076cfddcff99c2d044ab45615d403ea4bbff73bcc25673b85df489cdd95c4f34eb3174fd6364f329d")
			suffix, _ := hex.DecodeString("000000008300008000088000030100800008800063dc51bc822859801a2bd72296a830099f30c08b99848bd475eeaf7da128dcd74e29f43a2288575073434200aa4aa8b6bdde00698c0644cb4b88ebf3bddbce0715d4d087b0254352457082acd7e7a4b97c9b190e0d778a6973b394be14e3cd9ee8afa4e6a0a25180c387cdca80e45c7fb031db035a2f0f611a3934efb8b734e03db66f8e9ee5c8138271e20edfa853148b5c037e6ef4ffc448b9217c0b9de45a60a59f0765d511d126911f702a5e688bb438a61bb37351fe59016e0ed7d9e75d1da6a10675f48a18f20a58e94cacda4e")

			resBytes := append(prefix1, append(servBytes, append([]byte{0x1A, 0x88, 0xFF, 0xFF}, suffix...)...)...)  // first server, port 6792
			resBytes2 := append(prefix2, append(servBytes, append([]byte{0x1A, 0x88, 0xFF, 0xFF}, suffix...)...)...) // first server, port 6792

			fmt.Printf("Replying with: \n %s\n", hex.Dump(resBytes))
			c.WriteMessage(mt, resBytes)
			fmt.Printf("Replying with: \n %s\n", hex.Dump(resBytes2))
			c.WriteMessage(mt, resBytes2)

			unknownMessage, _ := hex.DecodeString("f640bb78a2e78cbbe4ee6bc73a96e64301000000000000001a")
			fmt.Printf("Replying with: \n %s\n", hex.Dump(unknownMessage))
			c.WriteMessage(mt, unknownMessage)

		case bytes.Contains(message, matchmakeMMenu): // read ips from json and construct automatically later
			resBytes, _ := hex.DecodeString("f640bb78a2e78cbbcbbebfda33cf288f040000000000000000000000")
			c.WriteMessage(mt, resBytes)
			fmt.Printf("Replying with: \n %s\n", hex.Dump(resBytes))
			resBytes, _ = hex.DecodeString("f640bb78a2e78cbbf3ebbf19875fbffa140000000000000000000400e80300000acbcd4d7f0000011A880000") // one ip, 127.0.0.1:6792
			c.WriteMessage(mt, resBytes)
			fmt.Printf("Replying with: \n %s\n", hex.Dump(resBytes))
			resBytes, _ = hex.DecodeString("f640bb78a2e78cbb6c6c16f2c4d35a8dc10400000000000000000400e80300007b22726567696f6e735b305d7c726567696f6e6964223a22307838453841384141433831453737314237222c22726567696f6e735b305d7c656e64706f696e74223a2267616d656c6966742e61702d6e6f727468656173742d312e616d617a6f6e6177732e636f6d222c22726567696f6e735b315d7c726567696f6e6964223a22307838453841384141433831464137314234222c22726567696f6e735b315d7c656e64706f696e74223a2267616d656c6966742e61702d736f757468656173742d322e616d617a6f6e6177732e636f6d222c22726567696f6e735b325d7c726567696f6e6964223a22307838453841384141433831464137314237222c22726567696f6e735b325d7c656e64706f696e74223a2267616d656c6966742e61702d736f757468656173742d312e616d617a6f6e6177732e636f6d222c22726567696f6e735b335d7c726567696f6e6964223a22307836464239413131353237464141304631222c22726567696f6e735b335d7c656e64706f696e74223a2267616d656c6966742e75732d656173742d312e616d617a6f6e6177732e636f6d222c22726567696f6e735b345d7c726567696f6e6964223a22307836464239413131353237464141304632222c22726567696f6e735b345d7c656e64706f696e74223a2267616d656c6966742e75732d656173742d322e616d617a6f6e6177732e636f6d222c22726567696f6e735b355d7c726567696f6e6964223a22307836464239413131353237464141364631222c22726567696f6e735b355d7c656e64706f696e74223a227261642d6368696361676f2d656e64706f696e742d6c622d313639313736343934332e75732d656173742d312e656c622e616d617a6f6e6177732e636f6d222c22726567696f6e735b365d7c726567696f6e6964223a22307836464239413131353237464141364632222c22726567696f6e735b365d7c656e64706f696e74223a227261642d64616c6c61732d656e64706f696e742d6c622d313639313736343934332e75732d656173742d312e656c622e616d617a6f6e6177732e636f6d222c22726567696f6e735b375d7c726567696f6e6964223a22307836464239413131353237464141364633222c22726567696f6e735b375d7c656e64706f696e74223a227261642d686f7573746f6e2d656e64706f696e742d6c622d3132333435363738392e75732d656173742d312e656c622e616d617a6f6e6177732e636f6d222c22726567696f6e735b385d7c726567696f6e6964223a22307836464239413131353237464142324631222c22726567696f6e735b385d7c656e64706f696e74223a2267616d656c6966742e75732d776573742d312e616d617a6f6e6177732e636f6d222c22726567696f6e735b395d7c726567696f6e6964223a22307836464239413131353337464341364631222c22726567696f6e735b395d7c656e64706f696e74223a2267616d656c6966742e65752d63656e7472616c2d312e616d617a6f6e6177732e636f6d222c22726567696f6e735b31305d7c726567696f6e6964223a22307836464239413131353337464342324632222c22726567696f6e735b31305d7c656e64706f696e74223a2267616d656c6966742e65752d776573742d322e616d617a6f6e6177732e636f6d227d")
			c.WriteMessage(mt, resBytes)
			fmt.Printf("Replying with: \n %s\n", hex.Dump(resBytes))
		}
	}
}

func transaction(w http.ResponseWriter, r *http.Request) { // do we have comms for this?
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			continue
		}
		fmt.Println("Message in Transaction:")
		fmt.Println(hex.Dump(message))
		switch {
		case bytes.Contains(message, []byte{}):
			resBytes, _ := hex.DecodeString("")
			c.WriteMessage(mt, resBytes)
			fmt.Printf("Replying with: \n %s\n", hex.Dump(resBytes))
		}

	}
}

func constructZSTDPacket(path string, prefix []byte, suffix []byte) []byte {
	zstdBytes, decompSize := zstdCompressJson(path)
	lenBytes, _ := hex.DecodeString(fmt.Sprintf("%016x", len(zstdBytes)+len(suffix)+len(decompSize)))
	lenBytes_LE := revArray(lenBytes)
	return append(prefix, append(lenBytes_LE, append(suffix, append(decompSize, zstdBytes...)...)...)...)
}

func zstdCompressJson(path string) ([]byte, []byte) {
	fBytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Error opening file for ZSTD compression.\n%s", err)
	}

	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, fBytes); err != nil {
		fmt.Println(err)
	}
	fBytes = buffer.Bytes()

	decompSize, _ := hex.DecodeString(fmt.Sprintf("%08x", len(fBytes)))

	fBytesZSTD := []byte{}
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		log.Fatalf("Error while compressing file '%s' with ZSTD.\n%s", path, err)
	}
	fBytesZSTD = enc.EncodeAll(fBytes, fBytesZSTD)
	enc.Close()
	return fBytesZSTD, revArray(decompSize)
}

func revArray(arr []byte) []byte {
	length := len(arr)
	reversed := make([]byte, length)

	for i := 0; i < length; i++ {
		reversed[i] = arr[length-1-i]
	}
	return reversed
}
