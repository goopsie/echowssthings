package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/klauspost/compress/zstd"
)

var upgrader = websocket.Upgrader{}

type matchmakerServerConfig struct {
	IpInternal string `json:"internal_ip"`
	IpExternal string `json:"external_ip"`
	Port       int    `json:"port"`
}

type configData struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func main() {
	csh := http.NewServeMux()
	csh.HandleFunc("/", config)
	msh := http.NewServeMux()
	msh.HandleFunc("/", matchmaking)
	tsh := http.NewServeMux()
	tsh.HandleFunc("/", transaction)

	go http.ListenAndServe("0.0.0.0:8001", msh)
	go http.ListenAndServe("0.0.0.0:8002", tsh) // we have scarce data for this
	http.ListenAndServe("0.0.0.0:8003", csh)
}

func config(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Print("upgrade:", err)
		return
	}
	defer c.Close()
	fmt.Println("Echo client has connected to Config server")
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			fmt.Println("Echo client has disconnected from Config server")
			break
		}
		log.Println("Message in config:")
		fmt.Println(hex.Dump(message))
		headerSuffix, _ := hex.DecodeString("7b1d0e4427ee09157b1d0e4427ee0915")

		switch {
		case bytes.Contains(message[8:16], SNSConfigRequestv2):
			for _, v := range bytes.Split(message, magic) {
				if len(v) == 0 {
					continue
				}
				confReq := configData{}
				json.Unmarshal(v[17:len(v)-1], &confReq)
				fmt.Printf("Got config request ID: %s & Type: %s\n", confReq.ID, confReq.Type)
				c.WriteMessage(mt, constructZSTDPacket("./json/config_"+confReq.Type+".json", SNSConfigSuccessv2, headerSuffix))
				c.WriteMessage(mt, constructPacket([]byte{0x10}, STcpConnectionUnrequireEvent, []byte{}))
			}
		default:
			fmt.Print("Recieved unknown message in Config.\n\n")
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
	fmt.Println("Echo client has connected to Matchmaker server")
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			fmt.Println("Echo client has disconnected from Matchmaker server")
			break
		}

		switch {
		case bytes.Contains(message[8:16], SNSLobbyPendingSessionCancel):
			fmt.Print("Echo client cancelled matchmaking.\n\n")

		case bytes.Contains(message[8:16], SNSLobbyPlayerSessionsRequestv5):
			ovrID := message[len(message)-8:]
			mdata_MMOvr1, _ := hex.DecodeString("010000000000000000000000000000000000000000000000") // ?
			mdata_MMOvr2, _ := hex.DecodeString("049690bc9035874d9ccb3b4a9c463abe")                 // ?
			mdata_MMOvr2Suffix, _ := hex.DecodeString("ffff000000000000")                           // ?

			c.WriteMessage(mt, constructPacket(append(mdata_MMOvr1, mdata_MMOvr2...), SNSLobbyPlayerSessionsSuccessv3, []byte{}))
			c.WriteMessage(mt, constructPacket(mdata_MMOvr2, SNSLobbyPlayerSessionsSuccessv3, ovrID))
			c.WriteMessage(mt, constructPacket(append(mdata_MMOvr2, mdata_MMOvr2Suffix...), incHeader(SNSLobbyPlayerSessionsSuccessv3, 1), ovrID))

			c.WriteMessage(mt, constructPacket([]byte{0x30}, STcpConnectionUnrequireEvent, []byte{}))

			fmt.Print("OVR Matchmaker messages sent\n\n")

		case bytes.Contains(message[8:16], SNSLobbyPingResponse):
			mlen := len(message)
			servBytes := readServersFromJson("./json/matchmaker_config.json")
			port := servBytes[len(servBytes)-4 : len(servBytes)-2]
			if mlen <= 32 {
				fmt.Print("No servers responded to Echo's pings. Attempting to fail echo.\n\n")
				c.WriteMessage(mt, constructPacket([]byte{0x00}, SNSLobbySessionFailurev4, []byte{})) //
				continue
			} else {
				servBytes = message[(32) : mlen-4] // set server address to the one echo responded with (will break if multiple servers returned)
			}

			// will have to len() final data after magic number, 24 bytes in? 							1556B44A-03D4-4BEA-BFF7-3BCC25673B85 lobby uuid
			prefix1, _ := hex.DecodeString("f640bb78a2e78cbb0e11e30e65e34d6d080100000000000076cfddcff99c2d0400000000000000000000000000000000")
			prefix2, _ := hex.DecodeString("f640bb78a2e78cbb0f11e30e65e34d6d180100000000000076cfddcff99c2d0400000000000000000000000000000000df489cdd95c4f34eb3174fd6364f329d")
			suffix, _ := hex.DecodeString("000000008300008000088000030100800008800063dc51bc82285980FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF73434200aa4aa8b6bdde00698c0644cb4b88ebf3bddbce0715d4d087b0254352FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFc387cdca80e45c7fFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF8b5c037e6ef4ffc448b9217c0b9de45a60a59f0765d511d126911f702a5e688bFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")

			resBytes := append(prefix1, append(servBytes, append(port, append([]byte{0xFF, 0xFF}, suffix...)...)...)...)  // first server, port 6792
			resBytes2 := append(prefix2, append(servBytes, append(port, append([]byte{0xFF, 0xFF}, suffix...)...)...)...) // first server, port 6792

			c.WriteMessage(mt, resBytes)
			c.WriteMessage(mt, resBytes2)
			fmt.Print("Sent game server connect request to Echo\n\n")

			c.WriteMessage(mt, constructPacket([]byte{0x1a}, STcpConnectionUnrequireEvent, []byte{}))

		case bytes.Contains(message, SNSLobbyFindSessionRequestv11) || bytes.Contains(message, SNSLobbyJoinSessionRequestv7):
			c.WriteMessage(mt, constructPacket([]byte{0x00, 0x00, 0x00, 0x00}, SNSLobbyMatchmakerStatus, []byte{}))
			serverDataSuffix, _ := hex.DecodeString("00000400e8030000") // ?
			serverData := readServersFromJson("./json/matchmaker_config.json")

			c.WriteMessage(mt, constructPacket(serverData, SNSLobbyPingRequestv3, serverDataSuffix))
			fmt.Print("Sent server list to Echo for ping.\n\n")

			c.WriteMessage(mt, constructJsonPacket("./json/matchmaker_region-endpoints.json", SNSLobbyPingRequestGamelift, serverDataSuffix))
			fmt.Print("Sent list of regions & Gamelift endpoints.\n\n")
		default:
			log.Println("Recieved unknown message in Matchmaking")
			hex.Dump(message)
		}
	}
}

func transaction(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Print("upgrade:", err)
		return
	}
	defer c.Close()
	fmt.Println("Echo client has connected to Transaction")
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			fmt.Println("Echo client has disconnected from Transaction")
			break
		}
		log.Println("Message in Transaction:")
		fmt.Println(hex.Dump(message))
		switch {
		case bytes.Contains(message[8:16], SNSReconcileIAP):
			headerSuffix, _ := hex.DecodeString("0400000000000000") // ?
			headerSuffix = append(headerSuffix, message[48:56]...)  // ovr_id

			c.WriteMessage(mt, constructJsonPacket("./json/transaction_EPcount.json", SNSReconcileIAPResult, headerSuffix))
			c.WriteMessage(mt, constructPacket([]byte{0x12}, STcpConnectionUnrequireEvent, []byte{}))

			fmt.Print("Sent transaction_EPcount.json\n\n")
		default:
			fmt.Print("Recieved unknown message in Transaction\n\n")
		}
	}
}

func readServersFromJson(path string) []byte {
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Failed to open server list: %s\n", path)
		return []byte{}
	}

	var sInfo matchmakerServerConfig
	json.Unmarshal(file, &sInfo)
	finalBytes := []byte{}
	intIP := strings.Split(sInfo.IpInternal, ".")
	extIP := strings.Split(sInfo.IpExternal, ".")

	for i := 0; i < len(intIP); i++ {
		ipInt, _ := strconv.Atoi(intIP[i])
		ipByte, _ := hex.DecodeString(fmt.Sprintf("%02x", ipInt))
		finalBytes = append(finalBytes, ipByte...)
	}
	for i := 0; i < len(extIP); i++ {
		ipInt, _ := strconv.Atoi(extIP[i])
		ipByte, _ := hex.DecodeString(fmt.Sprintf("%02x", ipInt))
		finalBytes = append(finalBytes, ipByte...)
	}
	portB, _ := hex.DecodeString(fmt.Sprintf("%04x", sInfo.Port))
	return append(finalBytes, append(portB, []byte{0x00, 0x00}...)...)

}

func constructJsonPacket(path string, messageType []byte, suffix []byte) []byte {
	fBytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to open json file.\n%s", err)
	}

	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, fBytes); err != nil {
		fmt.Println(err)
	}
	jsonBytes := buffer.Bytes()
	return constructPacket(jsonBytes, messageType, suffix)
}

func constructZSTDPacket(path string, messageType []byte, suffix []byte) []byte {
	zstdBytes, decompSize := zstdCompressJson(path)
	lenBytes, _ := hex.DecodeString(fmt.Sprintf("%016x", len(zstdBytes)+len(suffix)+len(decompSize)))
	lenBytes_LE := revArray(lenBytes)
	return append(magic, append(messageType, append(lenBytes_LE, append(suffix, append(decompSize, zstdBytes...)...)...)...)...)
}

func constructPacket(data []byte, messageType []byte, suffix []byte) []byte {
	lenBytes, _ := hex.DecodeString(fmt.Sprintf("%016x", len(data)+len(suffix)))
	lenBytes_LE := revArray(lenBytes)
	return append(magic, append(messageType, append(lenBytes_LE, append(suffix, data...)...)...)...)
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

func incHeader(header []byte, inc uint64) []byte {
	value, err := strconv.ParseUint(hex.EncodeToString(revArray(header)), 16, 64)
	if err != nil {
		fmt.Printf("Conversion failed: %s\n", err)
	}
	value = value + inc
	revHeader, _ := hex.DecodeString(fmt.Sprintf("%08x", value))
	finalHeader := revArray(revHeader)
	if len(finalHeader) < 8 {
		finalHeader = append(finalHeader, 0x00)
	}
	return finalHeader
}

func revArray(arr []byte) []byte {
	length := len(arr)
	reversed := make([]byte, length)

	for i := 0; i < length; i++ {
		reversed[i] = arr[length-1-i]
	}
	return reversed
}
