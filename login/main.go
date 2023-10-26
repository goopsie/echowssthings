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

			//resBytes, _ := hex.DecodeString("f640bb78a2e78cbbf1552163c3e25bed8b01000000000000c603000000000000789c7d92dd6ea3301085dfc5d7d1ca8696fcbcca6a359a98218c626cd636d9ad2adebd3649d3a466f706a47386ef1c337e178c234cd6387da6561ca29f68233c0d2e121877823fe82ddb5310870e4d78f682d38ce6ee0c18750ff16d24082369ee58c3a20d78269f08ef02393f7b172284a9ebf8af3864edf1db34f75390ee1da0278b304e47934069ead746fc9e6822e0545428295fc4bc11cbd40a7591ff032e7175c67d9ee93befa617c0ab0ed50f590255066a371c3196c09bbedef06616c44accf3d30a068a9e7558595c927b183d05b29aee3b227b4924c317fa4c5e9603d798c1b56954b41cf068d2752880052f9c79bcb505b291a3a121bd41f7a4cf6bb5c87be7bf6e9376b6e313b418971da28ea91a245a02c1882140200cce6673f909a5055202db0b1ace8543441f230fa9a27add6ef772b793329fbbbd8a555537bb9d6aa4cc77e71a17a2f394ebfbb77bce839602eae61bbad9be2a59bf48f58056cd5eee55da5281ee08e3e4a95dcd78365398aaff1156ad85a9799e3f005cf04fce")
			// 32 bytes in, zlib
			// todo: constructzLib
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
