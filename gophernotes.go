package main

import (
	"encoding/json"
	"fmt"
	zmq "github.com/pebbe/zmq4"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
)

var logger *log.Logger

const protocolVersion string = "5.0"

// ConnectionInfo stores the contents of the kernel connection file created by Jupyter.
type ConnectionInfo struct {
	Signature_scheme string
	Transport        string
	Stdin_port       int
	Control_port     int
	IOPub_port       int
	HB_port          int
	Shell_port       int
	Key              string
	IP               string
}

// SocketGroup holds the sockets needed to communicate with the kernel, and
// the key for message signing.
type SocketGroup struct {
	Shell_socket   *zmq.Socket
	Control_socket *zmq.Socket
	Stdin_socket   *zmq.Socket
	IOPub_socket   *zmq.Socket
	Key            []byte
}

// PrepareSockets sets up the ZMQ sockets through which the kernel will communicate.
func PrepareSockets(conn_info ConnectionInfo) (sg SocketGroup) {

	var err error

	context, err := zmq.NewContext()
	if err != nil {
		logger.Fatal(err)
	}
	sg.Shell_socket, err = context.NewSocket(zmq.ROUTER)
	if err != nil {
		logger.Fatal(err)
	}
	sg.Control_socket, err = context.NewSocket(zmq.ROUTER)
	if err != nil {
		logger.Fatal(err)
	}
	sg.Stdin_socket, err = context.NewSocket(zmq.ROUTER)
	if err != nil {
		logger.Fatal(err)
	}
	sg.IOPub_socket, err = context.NewSocket(zmq.PUB)
	if err != nil {
		logger.Fatal(err)
	}

	address := fmt.Sprintf("%v://%v:%%v", conn_info.Transport, conn_info.IP)

	if err = sg.Shell_socket.Bind(fmt.Sprintf(address, conn_info.Shell_port)); err != nil {
		logger.Fatal("sg.Shell_socket bind:", err)
	}

	if err = sg.Control_socket.Bind(fmt.Sprintf(address, conn_info.Control_port)); err != nil {
		logger.Fatal("sg.Control_socket bind:", err)
	}

	if err = sg.Stdin_socket.Bind(fmt.Sprintf(address, conn_info.Stdin_port)); err != nil {
		logger.Fatal("sg.Stdin_socket bind:", err)
	}

	if err = sg.IOPub_socket.Bind(fmt.Sprintf(address, conn_info.IOPub_port)); err != nil {
		logger.Fatal("sg.IOPub_socket bind:", err)
	}

	// Message signing key
	sg.Key = []byte(conn_info.Key)

	// Start the heartbeat device
	HB_socket, err := context.NewSocket(zmq.REP)

	if err != nil {
		logger.Fatal(err)
	}

	if err = HB_socket.Bind(fmt.Sprintf(address, conn_info.HB_port)); err != nil {
		logger.Fatal("HB_socket bind:", err)
	}

	go func() {
		err := zmq.Proxy(HB_socket, HB_socket, nil)
		if err != nil {
			logger.Fatal(err)
		}
	}()

	return
}

// HandleShellMsg responds to a message on the shell ROUTER socket.
func HandleShellMsg(receipt MsgReceipt) {
	switch receipt.Msg.Header.MsgType {
	case "kernel_info_request":
		SendKernelInfo(receipt)
	case "execute_request":
		HandleExecuteRequest(receipt)
	case "shutdown_request":
		HandleShutdownRequest(receipt)
	default:
		logger.Println("Unhandled shell message:", receipt.Msg.Header.MsgType)
	}
}

// KernelInfo holds information about the igo kernel, for kernel_info_reply messages.
type KernelInfo struct {
	ProtocolVersion       string             `json:"protocol_version"`
	Implementation        string             `json:"implementation"`
	ImplementationVersion string             `json:"implementation_version"`
	LanguageInfo          KernelLanguageInfo `json:"language_info"`
	Banner                string             `json:"banner"`
}

type KernelLanguageInfo struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Mimetype      string `json:"mimetype"`
	FileExtension string `json:"file_extension"`
}

// KernelStatus holds a kernel state, for status broadcast messages.
type KernelStatus struct {
	ExecutionState string `json:"execution_state"`
}

// SendKernelInfo sends a kernel_info_reply message.
func SendKernelInfo(receipt MsgReceipt) {
	reply := NewMsg("kernel_info_reply", receipt.Msg)

	reply.Content = KernelInfo{
		ProtocolVersion:       protocolVersion,
		Implementation:        "Gophernotes",
		ImplementationVersion: "0.1",
		LanguageInfo: KernelLanguageInfo{
			Name:          "go",
			Version:       runtime.Version(),
			Mimetype:      "application/x-golang", // text/plain would be possible, too
			FileExtension: ".go",
		},
		Banner: "Gophernotes - msg spec v5",
	}

	receipt.SendResponse(receipt.Sockets.Shell_socket, reply)

	idle := NewMsg("status", receipt.Msg)
	idle.Content = KernelStatus{"idle"}
	receipt.SendResponse(receipt.Sockets.IOPub_socket, idle)
}

// ShutdownReply encodes a boolean indication of stutdown/restart
type ShutdownReply struct {
	Restart bool `json:"restart"`
}

// HandleShutdownRequest sends a "shutdown" message
func HandleShutdownRequest(receipt MsgReceipt) {
	reply := NewMsg("shutdown_reply", receipt.Msg)
	content := receipt.Msg.Content.(map[string]interface{})
	restart := content["restart"].(bool)
	reply.Content = ShutdownReply{restart}
	receipt.SendResponse(receipt.Sockets.Shell_socket, reply)
	logger.Println("Shutting down in response to shutdown_request")
	os.Exit(0)
}

// RunKernel is the main entry point to start the kernel. This is what is called by the
// gophernotes executable.
func RunKernel(connection_file string, logwriter io.Writer) {

	logger = log.New(logwriter, "gophernotes ", log.LstdFlags)

	// set up the "Session" with the replpkg
	SetupExecutionEnvironment()

	var conn_info ConnectionInfo
	bs, err := ioutil.ReadFile(connection_file)
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(bs, &conn_info)
	if err != nil {
		log.Fatalln(err)
	}
	logger.Printf("%+v\n", conn_info)

	// Set up the ZMQ sockets through which the kernel will communicate
	sockets := PrepareSockets(conn_info)

	pi := zmq.NewPoller()

	pi.Add(sockets.Shell_socket, zmq.POLLIN)
	pi.Add(sockets.Stdin_socket, zmq.POLLIN)
	pi.Add(sockets.Control_socket, zmq.POLLIN)

	var msgparts [][]byte
	var polled []zmq.Polled
	// Message receiving loop:
	for {
		polled, err = pi.Poll(-1)
		if err != nil {
			log.Fatalln(err)
		}
		switch {
		case polled[0].Events&zmq.POLLIN != 0: // shell socket
			msgparts, _ = polled[0].Socket.RecvMessageBytes(0)
			msg, ids, err := WireMsgToComposedMsg(msgparts, sockets.Key)
			if err != nil {
				logger.Println(err)
				return
			}
			logger.Println("received shell message: ", msg)
			HandleShellMsg(MsgReceipt{msg, ids, sockets})
		case polled[1].Events&zmq.POLLIN != 0: // stdin socket - not implemented.
			polled[1].Socket.RecvMessageBytes(0)
		case polled[2].Events&zmq.POLLIN != 0: // control socket - treat like shell socket.
			msgparts, _ = polled[2].Socket.RecvMessageBytes(0)
			msg, ids, err := WireMsgToComposedMsg(msgparts, sockets.Key)
			if err != nil {
				logger.Println(err)
				return
			}
			logger.Println("received control message: ", msg)
			HandleShellMsg(MsgReceipt{msg, ids, sockets})
		}
	}
}
