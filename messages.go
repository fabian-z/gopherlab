package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	uuid "github.com/nu7hatch/gouuid"
	zmq "github.com/pebbe/zmq4"
)

// MsgHeader encodes header info for ZMQ messages
type MsgHeader struct {
	MsgID    string `json:"msg_id"`
	Username string `json:"username"`
	Session  string `json:"session"`
	MsgType  string `json:"msg_type"`
}

// ComposedMsg represents an entire message in a high-level structure.
type ComposedMsg struct {
	Header       MsgHeader
	ParentHeader MsgHeader
	Metadata     map[string]interface{}
	Content      interface{}
}

// InvalidSignatureError is returned when the signature on a received message does not
// validate.
type InvalidSignatureError struct{}

func (e *InvalidSignatureError) Error() string {
	return "A message had an invalid signature"
}

// WireMsgToComposedMsg translates a multipart ZMQ messages received from a socket into
// a ComposedMsg struct and a slice of return identities. This includes verifying the
// message signature.
func WireMsgToComposedMsg(msgparts [][]byte, signkey []byte) (msg ComposedMsg,
	identities [][]byte, err error) {
	i := 0
	for string(msgparts[i]) != "<IDS|MSG>" {
		i++
	}
	identities = msgparts[:i]
	// msgparts[i] is the delimiter

	// Validate signature
	if len(signkey) != 0 {
		mac := hmac.New(sha256.New, signkey)
		for _, msgpart := range msgparts[i+2 : i+6] {
			mac.Write(msgpart)
		}
		signature := make([]byte, hex.DecodedLen(len(msgparts[i+1])))
		hex.Decode(signature, msgparts[i+1])
		if !hmac.Equal(mac.Sum(nil), signature) {
			return msg, nil, &InvalidSignatureError{}
		}
	}
	json.Unmarshal(msgparts[i+2], &msg.Header)
	json.Unmarshal(msgparts[i+3], &msg.ParentHeader)
	json.Unmarshal(msgparts[i+4], &msg.Metadata)
	json.Unmarshal(msgparts[i+5], &msg.Content)
	return
}

// ToWireMsg translates a ComposedMsg into a multipart ZMQ message ready to send, and
// signs it. This does not add the return identities or the delimiter.
func (msg ComposedMsg) ToWireMsg(signkey []byte) (msgparts [][]byte) {

	msgparts = make([][]byte, 5)
	header, err := json.Marshal(msg.Header)
	if err != nil {
		logger.Fatal(err)
	}
	msgparts[1] = header
	parentHeader, err := json.Marshal(msg.ParentHeader)
	if err != nil {
		logger.Fatal(err)
	}
	msgparts[2] = parentHeader
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	metadata, err := json.Marshal(msg.Metadata)
	if err != nil {
		logger.Fatal(err)
	}
	msgparts[3] = metadata
	content, err := json.Marshal(msg.Content)
	if err != nil {
		logger.Fatal(err)
	}
	msgparts[4] = content

	// Sign the message
	if len(signkey) != 0 {
		mac := hmac.New(sha256.New, signkey)
		for _, msgpart := range msgparts[1:] {
			mac.Write(msgpart)
		}
		msgparts[0] = make([]byte, hex.EncodedLen(mac.Size()))
		hex.Encode(msgparts[0], mac.Sum(nil))
	}

	return
}

// MsgReceipt represents a received message, its return identities, and the sockets for
// communication.
type MsgReceipt struct {
	Msg        ComposedMsg
	Identities [][]byte
	Sockets    SocketGroup
}

// SendResponse sends a message back to return identites of the received message.
func (receipt *MsgReceipt) SendResponse(socket *zmq.Socket, msg ComposedMsg) {

	var err error

	for i := 0; i < len(receipt.Identities)-1; i++ {
		if _, err = socket.SendBytes(receipt.Identities[i], zmq.SNDMORE); err != nil {
			logger.Fatal(err)
		}
	}
	_, err = socket.SendBytes(receipt.Identities[(len(receipt.Identities)-1)], zmq.SNDMORE)
	if err != nil {
		logger.Fatal(err)
	}

	_, err = socket.Send("<IDS|MSG>", zmq.SNDMORE)
	if err != nil {
		logger.Fatal(err)
	}

	newmsg := msg.ToWireMsg(receipt.Sockets.Key)

	for i := 0; i < len(newmsg)-1; i++ {
		if _, err = socket.SendBytes(newmsg[i], zmq.SNDMORE); err != nil {
			logger.Fatal(err)
		}
	}
	_, err = socket.SendBytes(newmsg[(len(newmsg)-1)], 0)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Println("<--", msg.Header.MsgType)
	logger.Printf("%+v\n", msg.Content)
}

// NewMsg creates a new ComposedMsg to respond to a parent message. This includes setting
// up its headers.
func NewMsg(msgType string, parent ComposedMsg) (msg ComposedMsg) {
	msg.ParentHeader = parent.Header
	msg.Header.Session = parent.Header.Session
	msg.Header.Username = parent.Header.Username
	msg.Header.MsgType = msgType
	u, _ := uuid.NewV4()
	msg.Header.MsgID = u.String()
	return
}
