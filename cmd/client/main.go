package main

import (
	"fmt"
	"log"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/vpx"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v3"
	"webrtc.ir/rtctes/utils"

	_ "github.com/pion/mediadevices/pkg/driver/screen"
)

var (
	socket *utils.Socket
	rtcapi *webrtc.API
	stream mediadevices.MediaStream

	user = map[string]string{
		"room": "1",
		"name": "a",
	}
	rtcPeerConnections = map[string]*webrtc.PeerConnection{}

	config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"stun:stun.services.mozilla.com"}},
			/*{
				URLs:       []string{"turn:TURN_IP:3478"},
				Username:   "username",
				Credential: "password",
			},*/
		},
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}
)

func main() {
	initSocket()
	initWebRTCAPI()

	select {}
}

func initWebRTCAPI() {
	mediaEngine := webrtc.MediaEngine{}

	vpxParams, err := vpx.NewVP8Params()
	if err != nil {
		panic(err)
	}
	vpxParams.BitRate = 500_000 // 500kbps
	vpxParams.KeyFrameInterval = 120

	codecSelector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&vpxParams),
	)

	codecSelector.Populate(&mediaEngine)
	rtcapi = webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine))

	//
	// Join as Broadcaster
	//
	log.Println("[Init WebRTC API]: ", user["name"], " is broadcasting...")
	log.Println(mediadevices.EnumerateDevices())

	mediaStream, err := mediadevices.GetDisplayMedia(mediadevices.MediaStreamConstraints{
		Video: func(c *mediadevices.MediaTrackConstraints) {
			c.FrameFormat = prop.FrameFormat(frame.FormatYUY2)
			c.Width = prop.Int(320)
			c.Height = prop.Int(240)
		},
		Codec: codecSelector,
	})

	if err != nil {
		log.Fatalln("[Init WebRTC API]: ", err)
	}

	stream = mediaStream
	socket.Emit("register as broadcaster", user["room"])
}

func initSocket() {
	socket = utils.NewSocket()

	socket.On("new viewer", func(args []any) {
		viewer := args[0].(map[string]any)
		viewerId := viewer["id"].(string)
		viewerName := viewer["name"].(string)

		peerConnection, err := rtcapi.NewPeerConnection(config)
		if err != nil {
			log.Println("[New Viewer] ", viewerName, ": ", err)
			return
		}
		rtcPeerConnections[viewerId] = peerConnection

		for _, track := range stream.GetTracks() {
			track.OnEnded(func(err error) {
				fmt.Printf("Track (ID: %s) ended with error: %v\n",
					track.ID(), err)
			})
			_, err = peerConnection.AddTransceiverFromTrack(track,
				webrtc.RtpTransceiverInit{
					Direction: webrtc.RTPTransceiverDirectionSendonly,
				},
			)
			if err != nil {
				log.Println("[New Viewer] ", viewerName, ": ", err)
				return
			}
		}

		peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			if candidate != nil {
				candidateJSON := candidate.ToJSON()

				socket.Emit("candidate", viewerId, map[string]any{
					"type":      "candidate",
					"label":     *candidateJSON.SDPMLineIndex,
					"id":        *candidateJSON.SDPMid,
					"candidate": candidateJSON.Candidate,
				})
			}
		})

		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			log.Println("[New Viewer] ", viewerName, ": ", err)
			return
		}

		err = peerConnection.SetLocalDescription(offer)
		if err != nil {
			log.Println("[New Viewer] ", viewerName, ": ", err)
			return
		}

		socket.Emit("offer", viewerId, map[string]any{
			"type":        "offer",
			"sdp":         peerConnection.LocalDescription(),
			"broadcaster": user,
		})

		log.Println("[New Viewer] ", viewerName, " has joined")
	})

	socket.On("candidate", func(args []any) {
		id := args[0].(string)
		event := args[1].(map[string]any)
		sdpMLineIndex := uint16(event["label"].(float64))

		candidate := webrtc.ICECandidateInit{
			SDPMLineIndex: &sdpMLineIndex,
			Candidate:     event["candidate"].(string),
		}
		rtcPeerConnections[id].AddICECandidate(candidate)
	})

	socket.On("answer", func(args []any) {
		viewerId := args[0].(string)
		event := args[1].(map[string]any)

		rtcPeerConnections[viewerId].SetRemoteDescription(
			webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  event["sdp"].(string),
			},
		)
	})
}
