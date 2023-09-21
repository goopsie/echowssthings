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

// message type from header
var magic, _ = hex.DecodeString("f640bb78a2e78cbb")

var configBeginning, _ = hex.DecodeString("7843eb370b9f8682")

var matchmakeMMenu, _ = hex.DecodeString("f5a39a81012a2c31")
var matchmakerServerList, _ = hex.DecodeString("4fae333004d04760")
var matchmakerServerRegistrationRQ, _ = hex.DecodeString("5a10250f85daf270")
var matchmakerOvrIDK, _ = hex.DecodeString("051ac8a0b2faf29a")

var transactionGetEP, _ = hex.DecodeString("3c57854c45fcd01b")

var upgrader = websocket.Upgrader{}

type matchmakerServerConfig struct {
	IpInternal string `json:"internal_ip"`
	IpExternal string `json:"external_ip"`
	Port       int    `json:"port"`
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
		messageType, _ := hex.DecodeString("12d07b6f58afcdb9")
		configAcknowledge, _ := hex.DecodeString("e4ee6bc73a96e643")
		headerSuffix, _ := hex.DecodeString("7b1d0e4427ee09157b1d0e4427ee0915")

		switch {
		case bytes.Contains(message[8:16], configBeginning):
			resBytes := constructZSTDPacket("./json/config_echopass_textures.json", messageType, headerSuffix)
			c.WriteMessage(mt, resBytes)
			c.WriteMessage(mt, constructPacket([]byte{0x10}, configAcknowledge, []byte{}))
			fmt.Print("Sent config_echopass_textures.json\n\n")
		default:
			fmt.Print("Recieved unknown message.\n\n")

			// have to figure out another way to do this. e.g. reading the response 🤢
			//case bytes.Contains(message, configEchoshop):
			//	resBytes := constructZSTDPacket("./json/config_echopass_info.json", headerPrefix, headerSuffix)
			//	c.WriteMessage(mt, resBytes)
			//	resBytes, _ = hex.DecodeString("f640bb78a2e78cbbe4ee6bc73a96e643010000000000000004")
			//	c.WriteMessage(mt, resBytes)
			//	fmt.Print("Sent config_echopass_info.json\n\n")
			//
			//	resBytes = constructZSTDPacket("./json/config_echoshop_1.json", headerPrefix, headerSuffix)
			//	c.WriteMessage(mt, resBytes)
			//	c.WriteMessage(mt, constructPacket([]byte{0x4}, configAcknowledge, []byte{}))
			//	fmt.Print("Sent config_echoshop_1.json\n\n")
			//
			//	resBytes = constructZSTDPacket("./json/config_echoshop_2.json", headerPrefix, headerSuffix)
			//	c.WriteMessage(mt, resBytes)
			//	c.WriteMessage(mt, c.WriteMessage(mt, constructPacket([]byte{0x4}, configAcknowledge, []byte{})))
			//	fmt.Print("Sent config_echoshop_2.json\n\n")
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
		log.Println("Message in Matchmaking:")
		fmt.Println(hex.Dump(message))

		switch {
		case bytes.Contains(message[8:16], matchmakerServerRegistrationRQ):
			resBytes, _ := hex.DecodeString("f640bb78a2e78cbb0e11e30e65e34d6d0100000000000000ff") // nothing ):
			c.WriteMessage(mt, resBytes)
			fmt.Print("Attempting to respond to matchmaker registration request.\n\n")

		case bytes.Contains(message[8:16], matchmakerOvrIDK):
			ovrID := message[len(message)-8:]
			mtype_MMOvr1, _ := hex.DecodeString("d9fbe0f76a8571ff")
			mdata_MMOvr1, _ := hex.DecodeString("010000000000000000000000000000000000000000000000049690bc9035874d9ccb3b4a9c463abe")
			c.WriteMessage(mt, constructPacket(mdata_MMOvr1, mtype_MMOvr1, []byte{}))

			mtype_MMOvr2, _ := hex.DecodeString("688958f8e1cab9a1")
			mdata_MMOvr2, _ := hex.DecodeString("049690bc9035874d9ccb3b4a9c463abe")
			mdata_MMOvr2Suffix, _ := hex.DecodeString("ffff000000000000")
			c.WriteMessage(mt, constructPacket(mdata_MMOvr2, mtype_MMOvr2, ovrID))
			c.WriteMessage(mt, constructPacket(append(mdata_MMOvr2, mdata_MMOvr2Suffix...), incHeader(mtype_MMOvr2, 1), ovrID))

			mtype_MMOvr3, _ := hex.DecodeString("e4ee6bc73a96e643")
			c.WriteMessage(mt, constructPacket([]byte{0x30}, mtype_MMOvr3, []byte{}))

			fmt.Print("OVR Matchmaker messages sent\n\n")

		case bytes.Contains(message[8:16], matchmakerServerList):
			mlen := len(message)
			servBytes := readServersFromJson("./json/matchmaker_config.json")
			port := servBytes[len(servBytes)-4 : len(servBytes)-2]
			if mlen <= 32 {
				fmt.Print("No servers responded to Echo's pings. Ignoring and attempting connection regardless.\n\n")
				servBytes = servBytes[:len(servBytes)-4]
			} else {
				servBytes = message[(32) : mlen-4] // send back only server (WILL BREAK IF >1 SERVER IS RETURNED. WILL FIX LATER, SORT BY PING INSTEAD OF DISCARD.)
			}

			// will have to len() final data after magic number, 24 bytes in? 							1556B44A-03D4-4BEA-BFF7-3BCC25673B85
			prefix1, _ := hex.DecodeString("f640bb78a2e78cbb0e11e30e65e34d6d080100000000000076cfddcff99c2d0400000000000000000000000000000000")
			prefix2, _ := hex.DecodeString("f640bb78a2e78cbb0f11e30e65e34d6d180100000000000076cfddcff99c2d0400000000000000000000000000000000df489cdd95c4f34eb3174fd6364f329d")
			suffix, _ := hex.DecodeString("000000008300008000088000030100800008800063dc51bc822859801a2bd72296a830099f30c08b99848bd475eeaf7da128dcd74e29f43a2288575073434200aa4aa8b6bdde00698c0644cb4b88ebf3bddbce0715d4d087b0254352457082acd7e7a4b97c9b190e0d778a6973b394be14e3cd9ee8afa4e6a0a25180c387cdca80e45c7fb031db035a2f0f611a3934efb8b734e03db66f8e9ee5c8138271e20edfa853148b5c037e6ef4ffc448b9217c0b9de45a60a59f0765d511d126911f702a5e688bb438a61bb37351fe59016e0ed7d9e75d1da6a10675f48a18f20a58e94cacda4e")
			// todo: figure out what on earth suffix is :(

			resBytes := append(prefix1, append(servBytes, append(port, append([]byte{0xFF, 0xFF}, suffix...)...)...)...)  // first server, port 6792
			resBytes2 := append(prefix2, append(servBytes, append(port, append([]byte{0xFF, 0xFF}, suffix...)...)...)...) // first server, port 6792

			c.WriteMessage(mt, resBytes)
			c.WriteMessage(mt, resBytes2)
			fmt.Print("Sent game server connect request to Echo\n\n")

			messageType, _ := hex.DecodeString("e4ee6bc73a96e643")
			c.WriteMessage(mt, constructPacket([]byte{0x1a}, messageType, []byte{}))

		case bytes.Contains(message, matchmakeMMenu):
			mtype_matchmakerStart, _ := hex.DecodeString("cbbebfda33cf288f")
			c.WriteMessage(mt, constructPacket([]byte{0x00, 0x00, 0x00, 0x00}, mtype_matchmakerStart, []byte{}))
			mtype_serverList, _ := hex.DecodeString("f3ebbf19875fbffa")
			serverDataSuffix, _ := hex.DecodeString("00000400e8030000")
			serverData := readServersFromJson("./json/matchmaker_config.json")

			c.WriteMessage(mt, constructPacket(serverData, mtype_serverList, serverDataSuffix))
			fmt.Print("Sent server list to Echo for ping.\n\n")

			mtype_gameliftList, _ := hex.DecodeString("6c6c16f2c4d35a8d")
			c.WriteMessage(mt, constructJsonPacket("./json/matchmaker_region-endpoints.json", mtype_gameliftList, serverDataSuffix))
			fmt.Print("Sent list of regions & Gamelift endpoints.\n\n")
		default:
			log.Println("Recieved unknown message in Matchmaking:")
			fmt.Println(hex.Dump(message))
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
		case bytes.Contains(message[8:16], transactionGetEP):
			messageType, _ := hex.DecodeString("828a506542c2ab0d")
			headerSuffix, _ := hex.DecodeString("0400000000000000")
			headerSuffix = append(headerSuffix, message[48:56]...) // ovr_id

			resBytes := constructJsonPacket("./json/transaction_EPcount.json", messageType, headerSuffix)
			c.WriteMessage(mt, resBytes)
			messageType, _ = hex.DecodeString("e4ee6bc73a96e643")
			c.WriteMessage(mt, constructPacket([]byte{0x12}, messageType, []byte{}))
			fmt.Print("Sent transaction_EPcount.json\n\n")
		default:
			fmt.Print("Recieved unknown message in Transaction\n\n")
			fmt.Println(hex.Dump(message))
		}
	}
}

func readServersFromJson(path string) []byte {
	// 7f0000017f0000011A880000
	// 4 bytes for each ip, 2 bytes for port, 2 bytes padding
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

	for i := 0; i < len(intIP); i++ { // i think this is gonna break
		ipInt, _ := strconv.Atoi(intIP[i])
		ipByte, _ := hex.DecodeString(fmt.Sprintf("%02x", ipInt))
		finalBytes = append(finalBytes, ipByte...)
	}
	for i := 0; i < len(extIP); i++ { // i think this is gonna break
		ipInt, _ := strconv.Atoi(extIP[i])
		ipByte, _ := hex.DecodeString(fmt.Sprintf("%02x", ipInt))
		finalBytes = append(finalBytes, ipByte...)
	}
	portB, _ := hex.DecodeString(fmt.Sprintf("%04x", sInfo.Port))
	return append(finalBytes, append(portB, []byte{0x00, 0x00}...)...)

}

func constructPacket(data []byte, messageType []byte, suffix []byte) []byte {
	lenBytes, _ := hex.DecodeString(fmt.Sprintf("%016x", len(data)+len(suffix)))
	lenBytes_LE := revArray(lenBytes)
	return append(magic, append(messageType, append(lenBytes_LE, append(suffix, data...)...)...)...)
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

func revArray(arr []byte) []byte { // very convenient :)
	length := len(arr)
	reversed := make([]byte, length)

	for i := 0; i < length; i++ {
		reversed[i] = arr[length-1-i]
	}
	return reversed
}
