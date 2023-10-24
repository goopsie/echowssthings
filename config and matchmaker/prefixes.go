package main

import "encoding/hex"

// would be better to write it as shown below but i'm lazy
// example := []byte{0x00, 0x01, 0x02, ...}

var ( // Not tied to any specific service
	magic, _                        = hex.DecodeString("f640bb78a2e78cbb") // Prefix bytes for every(?) message
	STcpConnectionUnrequireEvent, _ = hex.DecodeString("e4ee6bc73a96e643")
)

var ( // Config
	// client -> server
	SNSConfigRequestv2, _ = hex.DecodeString("7843eb370b9f8682")

	// server -> client
	SNSConfigSuccessv2, _ = hex.DecodeString("12d07b6f58afcdb9")
)

var ( // Matchmaking
	// client -> server
	SNSLobbyFindSessionRequestv11, _   = hex.DecodeString("f5a39a81012a2c31")
	SNSLobbyJoinSessionRequestv7, _    = hex.DecodeString("11b2ff778f46032f") // When using --connecttolanserver (?)
	SNSLobbyPlayerSessionsRequestv5, _ = hex.DecodeString("051ac8a0b2faf29a")
	SNSLobbyPendingSessionCancel, _    = hex.DecodeString("5a10250f85daf270")
	SNSLobbyPingResponse, _            = hex.DecodeString("4fae333004d04760")
	SNSLobbyMatchmakerStatusRequest, _ = hex.DecodeString("50b6ebe07a778b12")

	// server -> client
	SNSLobbyPingRequestv3, _           = hex.DecodeString("f3ebbf19875fbffa")
	SNSLobbyPlayerSessionsSuccessv3, _ = hex.DecodeString("698958f8e1cab9a1")
	SNSLobbyMatchmakerStatus, _        = hex.DecodeString("cbbebfda33cf288f")
	SNSLobbySessionSuccessv5, _        = hex.DecodeString("0f11e30e65e34d6d")
	SNSLobbySessionFailurev4, _        = hex.DecodeString("6cf945bc5e36e84a") // Don't know how this is used
	SNSLobbyPingRequestGamelift, _     = hex.DecodeString("6c6c16f2c4d35a8d") // Name isn't canonical, is this ever used?
)

var ( // Transaction
	// client -> server
	SNSReconcileIAP, _ = hex.DecodeString("3c57854c45fcd01b")

	// server -> client
	SNSReconcileIAPResult, _ = hex.DecodeString("828a506542c2ab0d")
)
