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

var connectionInfo ConnectionInfo

// ConnectionInfo stores the contents of the kernel connection file created by Jupyter.
type ConnectionInfo struct {
	Signature_scheme string `json:"signature_scheme"`
	Transport        string `json:"transport"`
	Stdin_port       int    `json:"stdin_port"`
	Control_port     int    `json:"control_port"`
	IOPub_port       int    `json:"iopub_port"`
	HB_port          int    `json:"hb_port"`
	Shell_port       int    `json:"shell_port"`
	Key              string `json:"key"`
	IP               string `json:"ip"`
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
func PrepareSockets() (sg SocketGroup) {

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

	address := fmt.Sprintf("%v://%v:%%v", connectionInfo.Transport, connectionInfo.IP)

	if err = sg.Shell_socket.Bind(fmt.Sprintf(address, connectionInfo.Shell_port)); err != nil {
		logger.Fatal("sg.Shell_socket bind:", err)
	}

	if err = sg.Control_socket.Bind(fmt.Sprintf(address, connectionInfo.Control_port)); err != nil {
		logger.Fatal("sg.Control_socket bind:", err)
	}

	if err = sg.Stdin_socket.Bind(fmt.Sprintf(address, connectionInfo.Stdin_port)); err != nil {
		logger.Fatal("sg.Stdin_socket bind:", err)
	}

	if err = sg.IOPub_socket.Bind(fmt.Sprintf(address, connectionInfo.IOPub_port)); err != nil {
		logger.Fatal("sg.IOPub_socket bind:", err)
	}

	// Message signing key
	sg.Key = []byte(connectionInfo.Key)

	// Start the heartbeat device
	HB_socket, err := context.NewSocket(zmq.REP)

	if err != nil {
		logger.Fatal(err)
	}

	if err = HB_socket.Bind(fmt.Sprintf(address, connectionInfo.HB_port)); err != nil {
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
		HandleWithStatus(receipt, SendKernelInfo)
	case "connect_request":
		HandleWithStatus(receipt, HandleConnectRequest)
	case "execute_request":
		HandleWithStatus(receipt, HandleExecuteRequest)
	case "shutdown_request":
		HandleWithStatus(receipt, HandleShutdownRequest)
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
}

// ShutdownReply encodes a boolean indication of shutdown/restart
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

// ConnectReply encodes the ports necessary for connecting to the kernel
type ConnectReply struct {
	ShellPort int `json:"shell_port"`
	IOPubPort int `json:"iopub_port"`
	StdinPort int `json:"stdin_port"`
	HBPort    int `json:"hb_port"`
}

func HandleConnectRequest(receipt MsgReceipt) {
	reply := NewMsg("connect_reply", receipt.Msg)

	reply.Content = ConnectReply{
		ShellPort: connectionInfo.Shell_port,
		IOPubPort: connectionInfo.IOPub_port,
		StdinPort: connectionInfo.Stdin_port,
		HBPort:    connectionInfo.HB_port,
	}

	receipt.SendResponse(receipt.Sockets.Shell_socket, reply)
}

// RunKernel is the main entry point to start the kernel. This is what is called by the
// gophernotes executable.
func RunKernel(connection_file string, logwriter io.Writer) {

	logger = log.New(logwriter, "gophernotes ", log.LstdFlags)

	// set up the "Session" with the replpkg
	SetupExecutionEnvironment()

	bs, err := ioutil.ReadFile(connection_file)
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(bs, &connectionInfo)
	if err != nil {
		log.Fatalln(err)
	}
	logger.Printf("%+v\n", connectionInfo)

	// Set up the ZMQ sockets through which the kernel will communicate
	sockets := PrepareSockets()

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
