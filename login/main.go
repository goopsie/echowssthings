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

var loginMainMenu, _ = hex.DecodeString("0a207be6a91eb4bd")
var loginStage2, _ = hex.DecodeString("708dfc21422a77fb")
var upgrader = websocket.Upgrader{}

func main() {
	lsh := http.NewServeMux()
	lsh.HandleFunc("/", login)

	http.ListenAndServe("0.0.0.0:8000", lsh)
}

func login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Login function called")
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("Echo client has disconnected from Login server.")
			break
		}
		switch {
		case bytes.Contains(message[8:16], loginMainMenu):
			fmt.Println("Got Login request")

			ovr_id := message[48:56]
			resBytes, _ := hex.DecodeString("f640bb78a2e78cbb47ce0c0da9c1aca52000000000000000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF0400000000000000") //
			resBytes = append(resBytes, ovr_id...)
			c.WriteMessage(mt, resBytes)

			fmt.Println("Sent login confirmation")
			resBytes, _ = hex.DecodeString("f640bb78a2e78cbbe4ee6bc73a96e64301000000000000004b") // COlPrEfIxSTcpConnectionUnrequireEvent - 0x4b
			c.WriteMessage(mt, resBytes)
			fmt.Println("Sent auth acknowledgement")
			resBytes, _ = hex.DecodeString("f640bb78a2e78cbbf1552163c3e25bed8b01000000000000c603000000000000789c7d92dd6ea3301085dfc5d7d1ca8696fcbcca6a359a98218c626cd636d9ad2adebd3649d3a466f706a47386ef1c337e178c234cd6387da6561ca29f68233c0d2e121877823fe82ddb5310870e4d78f682d38ce6ee0c18750ff16d24082369ee58c3a20d78269f08ef02393f7b172284a9ebf8af3864edf1db34f75390ee1da0278b304e47934069ead746fc9e6822e0545428295fc4bc11cbd40a7591ff032e7175c67d9ee93befa617c0ab0ed50f590255066a371c3196c09bbedef06616c44accf3d30a068a9e7558595c927b183d05b29aee3b227b4924c317fa4c5e9603d798c1b56954b41cf068d2752880052f9c79bcb505b291a3a121bd41f7a4cf6bb5c87be7bf6e9376b6e313b418971da28ea91a245a02c1882140200cce6673f909a5055202db0b1ace8543441f230fa9a27add6ef772b793329fbbbd8a555537bb9d6aa4cc77e71a17a2f394ebfbb77bce839602eae61bbad9be2a59bf48f58056cd5eee55da5281ee08e3e4a95dcd78365398aaff1156ad85a9799e3f005cf04fce")
			// 32 bytes in, zlib
			c.WriteMessage(mt, resBytes)
			fmt.Println("Sent auth login_settings")

		case bytes.Contains(message, loginStage2):
			ovr_id := message[48:56]

			prefix, _ := hex.DecodeString("f640bb78a2e78cbb778dfc37503a76fb")
			suffix, _ := hex.DecodeString("0400000000000000")
			resBytes := constructZSTDPacket("./json/login_userinfo.json", prefix, append(suffix, ovr_id...))

			c.WriteMessage(mt, resBytes)
			fmt.Println("Sending player information")

			resBytes, _ = hex.DecodeString("f640bb78a2e78cbbe4ee6bc73a96e64301000000000000004b")
			c.WriteMessage(mt, resBytes)
			fmt.Println("Sending player info acknowledgement")

			prefix, _ = hex.DecodeString("f640bb78a2e78cbb09b5b72f78fd7fd0")
			suffix, _ = hex.DecodeString("b112663f483ec3c8")
			resBytes = constructZSTDPacket("./json/login_eula.json", prefix, suffix)

			c.WriteMessage(mt, resBytes)
			fmt.Println("Sending eula")

			resBytes, _ = hex.DecodeString("f640bb78a2e78cbbe4ee6bc73a96e64301000000000000004b")
			c.WriteMessage(mt, resBytes)
			fmt.Println("Sending eula acknowledgement")
		}
	}
}

func constructZSTDPacket(path string, prefix []byte, suffix []byte) []byte {
	zstdBytes, decompSize := zstdCompressJson(path)
	lenBytes, _ := hex.DecodeString(fmt.Sprintf("%016x", len(zstdBytes)+len(suffix)+len(decompSize)))
	return append(prefix, append(revArray(lenBytes), append(suffix, append(decompSize, zstdBytes...)...)...)...)
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
