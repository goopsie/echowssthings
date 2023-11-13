package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strconv"
)

var pingReq, _ = hex.DecodeString("b0035a06de797299")
var pingRespHeader, _ = hex.DecodeString("9178b7e056e57a4f")
var connectReply1, _ = hex.DecodeString("5d0fdac83acfbbcd84f5a0880e7cb52358e85677d98a6937c45d20260577edb863dc51bc82285900c5dca810c4583d4085142119d6d4e93d91e9f7ab946a208adff17374043f0d208ca5892bc0d42d044cdb22f6f7c50dc17214ca4bce9d10430dcb1248fb46045a")
var connectReply2, _ = hex.DecodeString("59bac8c5043f6d288611848fe0fe2d69e659a21eadc29d29dfb89eef9335af8764dc51bc8228590058a55658ea3b1c8ba5f19cc63811044faa71868447ef888253e87da04eb64a7d")
var connectReply3, _ = hex.DecodeString("943602b5ac3491509a54e7ed2e019ced4b9fe265f1cf2ea1b3926c9a1a69a0a865dc51bc822859008b7c45b62b8751a735faad0de3560393a9c7c12e8ac24f700036d56c2675fa83e709e0c3e286ea7f8df983304cb2e6ae")
var connectReply4, _ = hex.DecodeString("df1ca936bd01bb5346fceb7ee04c5f9c2a4e386cfa44ff49922149a3a2413f0466dc51bc82285900bf27db2caf8ae3ca26e0704849577d2ef2fe9a93b55140b448ced3507485a7cf")
var connectReply5, _ = hex.DecodeString("20e9275a78396282488d4d9d0173aa1410019940d59d8d85d50de204d937f8c967dc51bc822859008298200ddefb69334bd454c5237c742839c07c0e6c0b631ad500b0b82084ecf3419dadf84f159227de29525fcff687ace98c2b8aa9568d3ffde4e9b9d19f44fb2db741a8766751bf0f77d5a2dcb8e6b0086d65e673bbbbc331e26cdf2b9350828a0aff418f25036e5e349c470043de7eac810265ef9ef4d1f93ef19903191e581fb7b11b210cd0cf7502fd99fa84603d8ef410300361d7904a33d5fa8ead1aaac07587278fe6c362675c1cb10934eb0616a3b52400baeb5a93299f4c24d174e5a619db722a0f67d726aebb602ca7c349009b2213e0c88884eb84e25f1a26970f9b294bd0ba0f90822a76c666dba26fb4fd752167dc977bbaef75e0344828b7de")

var setupRespHeader, _ = hex.DecodeString("63dc51bc82285900")
var currentIter uint64 = 0

const port = 6795

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
					if bytes.Contains(incHeader(prevFingerprints[i], 1), fingerprint) {
						// wow this sucks
						//resp = append(resp, connectReply1...)
						tempHeader := setupRespHeader

						tempHeader = incHeader(tempHeader, currentIter)

						log.Printf("Response Fingerprint:\n%s", hex.Dump(tempHeader))
						switch {
						case currentIter == 0:
							resp = append(connectReply1[0:32], append(tempHeader, connectReply1[40:]...)...)
						case currentIter == 1:
							resp = append(connectReply2[0:32], append(tempHeader, connectReply2[40:]...)...)
						case currentIter == 2:
							resp = append(connectReply3[0:32], append(tempHeader, connectReply3[40:]...)...)
						case currentIter == 3:
							resp = append(connectReply2[0:32], append(tempHeader, connectReply4[40:]...)...)
						case currentIter == 4:
							resp = append(connectReply3[0:32], append(tempHeader, connectReply5[40:]...)...)
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
