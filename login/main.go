package main

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/klauspost/compress/zstd"
)

var upgrader = websocket.Upgrader{}

func main() {
	lsh := http.NewServeMux()
	lsh.HandleFunc("/", login)

	http.ListenAndServe("0.0.0.0:8000", lsh)
}

func login(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Print("upgrade:", err)
		return
	}
	defer c.Close()
	log.Println("Echo client connected to Login server")
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("Echo client has disconnected from Login server.")
			break
		}
		switch {
		case bytes.Contains(message[8:16], SNSLoginRequestV2):
			fmt.Println("Got Login request")
			ovr_id := message[48:56]

			loginID, _ := hex.DecodeString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF") // Unsure how this is used in echo yet
			unknown, _ := hex.DecodeString("0400000000000000")

			c.WriteMessage(mt, constructPacket(SNSLogInSuccess, append(loginID, append(unknown, ovr_id...)...)))
			fmt.Println("Sent login confirmation")

			c.WriteMessage(mt, constructPacket(STcpConnectionUnrequireEvent, []byte{0x4b}))
			fmt.Println("Sent auth acknowledgement")

			c.WriteMessage(mt, constructZLibPacket(SNSLoginSettings, "./json/login_settings.json"))
			fmt.Println("Sent auth login_settings")

		case bytes.Contains(message[8:16], SNSLoggedInUserProfileRequest):
			ovr_id := message[48:56]

			unknown, _ := hex.DecodeString("0400000000000000")
			c.WriteMessage(mt, constructZSTDPacket(SNSLoggedInUserProfileSuccess, append(unknown, ovr_id...), "./json/login_userinfo.json"))
			fmt.Println("Sending player information")

			c.WriteMessage(mt, constructPacket(STcpConnectionUnrequireEvent, []byte{0x4b}))
			fmt.Println("Sending player info acknowledgement")

			c.WriteMessage(mt, constructZSTDPacket(SNSDocumentSuccess, PrEfIxeula, "./json/login_eula.json"))
			fmt.Println("Sending eula")

			c.WriteMessage(mt, constructPacket(STcpConnectionUnrequireEvent, []byte{0x4b}))
			fmt.Println("Sending eula acknowledgement")
		case bytes.Contains(message[8:16], SNSUpdateProfile):
			ovr_id := message[48:56]
			unknown, _ := hex.DecodeString("0400000000000000")
			c.WriteMessage(mt, constructPacket(SNSUpdateProfileSuccess, append(unknown, ovr_id...)))
			c.WriteMessage(mt, constructPacket(STcpConnectionUnrequireEvent, []byte{0x00}))
			// should *actually* update the profile but as of right now there are no distinct profiles
			// todo: above
			fmt.Println("Acknowledged SNSUpdateProfile request")

		case bytes.Contains(message[8:16], SNSChannelInfoRequest):
			c.WriteMessage(mt, constructZLibPacket(SNSChannelInfoResponse, "./json/lobby_groups.json"))
			c.WriteMessage(mt, constructPacket(STcpConnectionUnrequireEvent, []byte{0x43}))
			fmt.Println("Sent lobby_groups.json")

		default:
			fmt.Printf("Unknown packet recieved of type: %s", fmt.Sprintf("%016x", message[8:16]))
		}
	}
}

func constructPacket(messageType []byte, data []byte) []byte {
	dataLen, _ := hex.DecodeString(fmt.Sprintf("%016x", len(data)))
	return append(magic, append(messageType, append(revArray(dataLen), data...)...)...)
}

func constructZSTDPacket(messageType []byte, documentType []byte, path string) []byte { // ?
	zstdBytes, decompSize := zstdCompressJson(path)
	return constructPacket(messageType, append(documentType, append(decompSize, zstdBytes...)...))
}

// might be better to have a readJson() function but i'm not going to do that right now

func constructZLibPacket(messageType []byte, filePath string) []byte {
	fBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error opening file for ZLib compression.\n%s", err)
	}

	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, fBytes); err != nil {
		fmt.Println(err)
	}
	fBytes = buffer.Bytes()

	decompSize, _ := hex.DecodeString(fmt.Sprintf("%016x", len(fBytes)))

	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(fBytes)
	w.Close()
	return constructPacket(messageType, append(revArray(decompSize), in.Bytes()...))
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
