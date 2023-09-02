package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net"
)

var pingReq, _ = hex.DecodeString("b0035a06de797299")
var pingRespHeader, _ = hex.DecodeString("9178b7e056e57a4f")
var connectReply1, _ = hex.DecodeString("5d0fdac83acfbbcd84f5a0880e7cb52358e85677d98a6937c45d20260577edb863dc51bc82285900c5dca810c4583d4085142119d6d4e93d91e9f7ab946a208adff17374043f0d208ca5892bc0d42d044cdb22f6f7c50dc17214ca4bce9d10430dcb1248fb46045a")
var connectReply2, _ = hex.DecodeString("59bac8c5043f6d288611848fe0fe2d69e659a21eadc29d29dfb89eef9335af8764dc51bc8228590058a55658ea3b1c8ba5f19cc63811044faa71868447ef888253e87da04eb64a7d")
var connectReply3, _ = hex.DecodeString("943602b5ac3491509a54e7ed2e019ced4b9fe265f1cf2ea1b3926c9a1a69a0a865dc51bc822859008b7c45b62b8751a735faad0de3560393a9c7c12e8ac24f700036d56c2675fa83e709e0c3e286ea7f8df983304cb2e6ae")
var setupRespHeader, _ = hex.DecodeString("64dc51bc82285900")
var currentIter = 0

const port = 6792

func main() {
	fmt.Printf("Mock game server listening at port: %d\n", port)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: port})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	var data = make([]byte, 1024*4)
	var raw []byte
	var prevFingerprints [][]byte
	for {
		n, remoteAddr, err := conn.ReadFromUDP(data)
		if err != nil {
			log.Panic(err)
		}
		raw = make([]byte, n)
		copy(raw, data[:n])
		go func() {
			log.Printf("Client:\n%s", hex.Dump(raw))
			resp := []byte{}

			switch {
			case bytes.Contains(raw, pingReq):
				resp = append(resp, append(pingRespHeader, raw[(len(raw)/2):]...)...) // god
			default:
				if len(raw) < 72 { // compare bytes 65-72 with previous packet, if they're the same*, continue
					log.Println("Unknown packet recieved from client.")
					resp = append(resp, []byte("OK")...)
				}

				fingerprint := raw[64:72] // This seems to be some sort of header & packet number? first byte increases by one every packet, once it reaches 0xFF, second byte increases by one & first goes to 0x00, unsure if this behaviour continues further
				log.Printf("Fingerprint:\n%s", hex.Dump(fingerprint))
				if len(prevFingerprints) <= 0 { // don't send anything back, loop another time to confirm fingerprint
					resp = nil
					prevFingerprints = append(prevFingerprints, fingerprint)

					break
				}

				finprintUpdated := false
				for i := 0; i < len(prevFingerprints); i++ {
					if bytes.Contains(incHeader(prevFingerprints[i]), fingerprint) {
						// wow this sucks
						//resp = append(resp, connectReply1...)
						tempHeader := setupRespHeader
						for j := 0; j < currentIter; j++ {
							tempHeader = incHeader(tempHeader)
						}
						log.Printf("Response Fingerprint:\n%s", hex.Dump(tempHeader))
						switch {
						case currentIter == 0:
							resp = append(connectReply1[0:31], append(tempHeader, connectReply1[39:]...)...)
						case currentIter == 1:
							resp = append(connectReply2[0:31], append(tempHeader, connectReply2[39:]...)...)
						case currentIter == 2:
							resp = append(connectReply3[0:31], append(tempHeader, connectReply3[39:]...)...)
						default:
							resp = nil
						}
						prevFingerprints[i] = fingerprint
						currentIter++
						finprintUpdated = true
					}
				}

				if !finprintUpdated {
					prevFingerprints = append(prevFingerprints, fingerprint) // ugh
				}
			}
			if resp != nil {
				log.Printf("Server:\n%s", hex.Dump(resp))
				_, err := conn.WriteToUDP(resp, remoteAddr)
				if err != nil {
					log.Println(err)
				}
			}

		}()
	}
}

func incHeader(header []byte) []byte { // tee hee :) TODO: add second argument for amount to increment by
	if len(header) == 0 {
		header = append(header, 0x00)
	}
	if header[0] != 0xff {
		header[0] = header[0] + 0x01
		return header
	}
	final := []byte{0x00}

	return append(final, incHeader(header[1:])...)
}
