package main

import "encoding/hex"

// would be better to write it as shown below but i'm lazy
// example := []byte{0x00, 0x01, 0x02, ...}

var ( // Not tied to any specific service
	magic, _                        = hex.DecodeString("f640bb78a2e78cbb")
	STcpConnectionUnrequireEvent, _ = hex.DecodeString("e4ee6bc73a96e643")
)

var ( // Login
	SNSLoginRequestV2, _             = hex.DecodeString("0a207be6a91eb4bd")
	SNSLoggedInUserProfileRequest, _ = hex.DecodeString("708dfc21422a77fb")
	SNSUpdateProfile, _              = hex.DecodeString("1544f23f9aa1546d")
	// client -> server

	// server -> client
	SNSLogInSuccess, _               = hex.DecodeString("47ce0c0da9c1aca5")
	SNSLoginSettings, _              = hex.DecodeString("f1552163c3e25bed")
	SNSLoggedInUserProfileSuccess, _ = hex.DecodeString("778dfc37503a76fb")
	SNSUpdateProfileSuccess, _       = hex.DecodeString("57f7ce01d09154f2")
	SNSDocumentSuccess, _            = hex.DecodeString("09b5b72f78fd7fd0")
)

var ( // document prefixes
	PrEfIxeula, _ = hex.DecodeString("b112663f483ec3c8")
)
