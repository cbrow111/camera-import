package main

import (
	"camera-import/config"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/barnybug/go-cast/api"
	"github.com/gogo/protobuf/proto"
)

const appId = "CC1AD845"

func PlayCompletionSound(appConfig *config.Config) {
	localIP := GetLocalIP()

	// Default Media Receiver
	mediaURL := fmt.Sprintf("http://%s:8080/stream", localIP)

	// 1. Start File Server
	go func() {
		http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			log.Println("Nest is downloading audio...")
			http.ServeFile(w, r, appConfig.FileToCast)
		})
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// 2. Connect
	conf := &tls.Config{InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", appConfig.CastIp+":8009", conf)
	if err != nil {
		log.Fatalf("Socket failed: %v", err)
	}
	defer conn.Close()

	// STEP 1: Connect to System
	send(conn, "receiver-0", "urn:x-cast:com.google.cast.tp.connection", `{"type":"CONNECT"}`)

	// STEP 2: Launch App (Plays the Chime)
	launchJSON := fmt.Sprintf(`{"type":"LAUNCH","appId":"%s","requestId":1}`, appId)
	send(conn, "receiver-0", "urn:x-cast:com.google.cast.receiver", launchJSON)

	// STEP 3: Wait and Catch the Transport ID
	// We use a small timeout so we don't hang forever
	status, err := waitForStatus(conn)
	if err != nil {
		log.Fatalf("error getting status: %s", err)
	}
	transportId := extractTransportId(status.GetPayloadUtf8())

	originalVolumeLevel, err := extractVolumeLevel(status.GetPayloadUtf8())
	if err != nil {
		log.Fatalf("error extracting volume level: %s", err)
	}
	log.Printf("OriginalVolume: %v", originalVolumeLevel)

	err = setVolume(conn, appConfig.TargetVolume)
	if err != nil {
		log.Fatalf("error setting volume: %s", err)
	}

	// STEP 4: Connect to the App Session (CRITICAL)
	// This tells the specific Media App that we are talking to IT, not the system.
	send(conn, transportId, "urn:x-cast:com.google.cast.tp.connection", `{"type":"CONNECT"}`)

	// STEP 5: Load Media to the Transport ID
	loadJSON := fmt.Sprintf(`{
		"type": "LOAD",
		"requestId": 2,
		"media": {
			"contentId": "%s",
			"contentType": "audio/wav",
			"streamType": "BUFFERED"
		},
		"autoplay": true
	}`, mediaURL)
	send(conn, transportId, "urn:x-cast:com.google.cast.media", loadJSON)

	log.Println("Commands sent. Listening for download request...")
	waitForFinish(conn, transportId)
	err = setVolume(conn, originalVolumeLevel)
	if err != nil {
		log.Fatalf("error restoring original volume: %s", err)
	}
}

func setVolume(conn *tls.Conn, volume float64) error {
	// SET_VOLUME does not typically send a response, so use send()
	setVolJSON := fmt.Sprintf(`{"type":"SET_VOLUME", "volume": {"level": %.2f}, "requestId": 3}`, volume)
	return send(conn, "receiver-0", "urn:x-cast:com.google.cast.receiver", setVolJSON)
}

// Helper to grab transportId without complex JSON structs
func extractTransportId(jsonStr string) string {
	key := `"transportId":"`
	start := strings.Index(jsonStr, key)
	if start == -1 {
		return ""
	}
	start += len(key)
	end := strings.Index(jsonStr[start:], `"`)
	if end == -1 {
		return ""
	}
	return jsonStr[start : start+end]
}

// extractVolumeLevel pulls the "level" float from a RECEIVER_STATUS payload
func extractVolumeLevel(jsonStr string) (float64, error) {
	// Simple search for "level":0.XX
	key := `"level":`
	start := strings.Index(jsonStr, key)
	if start == -1 {
		return 0.5, fmt.Errorf("unable to get volume level: %s", jsonStr)
	} // Default fallback

	start += len(key)
	end := strings.IndexAny(jsonStr[start:], ",}")
	if end == -1 {
		return 0.5, fmt.Errorf("unable to get volume level from malformed json body: %s", jsonStr)
	}

	val, _ := strconv.ParseFloat(jsonStr[start:start+end], 64)
	return val, nil
}

// waitForStatus: get a status body back from the device
// once it contains a transportId
func waitForStatus(conn *tls.Conn) (*api.CastMessage, error) {
	var transportId string
	var err error
	var msg *api.CastMessage
	for i := 0; i < 20; i++ { // Try for up to 10 seconds
		log.Printf("Attempt %d to catch transportId...", i+1)

		// Send a fresh GET_STATUS to nudge the speaker
		send(conn, "receiver-0", "urn:x-cast:com.google.cast.receiver", `{"type":"GET_STATUS","requestId":1}`)

		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		msg, err = receive(conn)
		if err == nil {
			transportId = extractTransportId(msg.GetPayloadUtf8())
			if transportId != "" {
				break
			}
		}
		// Small sleep between retries if the app is still loading
		time.Sleep(500 * time.Millisecond)
	}

	if transportId == "" {
		log.Fatal("App failed to launch in time")
	}
	log.Printf("Targeting Transport ID: %s", transportId)
	return msg, err
}

func waitForFinish(conn net.Conn, transportId string) {
	for {
		// Use your existing receive function
		msg, err := receive(conn)
		if err != nil {
			break
		}

		// We only care about media status messages from our app
		if msg.GetSourceId() == transportId && strings.Contains(msg.GetPayloadUtf8(), "MEDIA_STATUS") {
			payload := msg.GetPayloadUtf8()

			// Check if the player has entered the IDLE state because it FINISHED
			if strings.Contains(payload, `"playerState":"IDLE"`) &&
				strings.Contains(payload, `"idleReason":"FINISHED"`) {
				log.Println("Playback finished naturally.")
				return
			}

			// Optional: Also catch if it was CANCELLED or had an ERROR
			if strings.Contains(payload, `"idleReason":"CANCELLED"`) ||
				strings.Contains(payload, `"idleReason":"ERROR"`) {
				log.Println("Playback stopped or failed.")
				return
			}
		}
	}
}

func send(conn net.Conn, dest, namespace, payload string) error {
	msg := &api.CastMessage{
		ProtocolVersion: api.CastMessage_CASTV2_1_0.Enum(),
		SourceId:        proto.String("sender-0"),
		DestinationId:   proto.String(dest),
		Namespace:       proto.String(namespace),
		PayloadType:     api.CastMessage_STRING.Enum(),
		PayloadUtf8:     proto.String(payload),
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	// Combine length + data into one slice to ensure a single TCP write
	packet := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(packet[:4], uint32(len(data)))
	copy(packet[4:], data)

	_, err = conn.Write(packet)
	return err
}

func receive(conn net.Conn) (*api.CastMessage, error) {
	// 1. Read the 4-byte header
	lenBuf := make([]byte, 4)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}

	// 2. Determine payload length
	length := binary.BigEndian.Uint32(lenBuf)

	// 3. Read the exact number of bytes for the Protobuf message
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	// 4. Unmarshal the Protobuf
	msg := &api.CastMessage{}
	if err := proto.Unmarshal(payload, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func request(conn net.Conn, dest, namespace string, payloadMap map[string]interface{}) (string, error) {
	// 1. Assign a unique Request ID
	reqID := time.Now().UnixNano()
	payloadMap["requestId"] = reqID
	jsonPayload, _ := json.Marshal(payloadMap)

	// 2. Send the message
	if err := send(conn, dest, namespace, string(jsonPayload)); err != nil {
		return "", err
	}

	// 3. Loop until we find the response with our requestId
	for {
		msg, err := receive(conn)
		if err != nil {
			return "", err
		}

		// Basic check: is this the JSON we're looking for?
		if strings.Contains(msg.GetPayloadUtf8(), fmt.Sprintf(`"requestId":%d`, reqID)) {
			return msg.GetPayloadUtf8(), nil
		}
		// Handle Heartbeats or other traffic here if necessary
	}
}

// GetLocalIP returns the outbound IP of the machine
func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
