package main

import (
	"fmt"
	repl "github.com/fabian-z/gopherlab/replpkg"
	"go/token"
)

// REPLSession manages the I/O to/from the notebook
var REPLSession *repl.Session
var fset *token.FileSet

// ExecCounter is incremented each time we run user code in the notebook
var ExecCounter int

// SetupExecutionEnvironment initializes the REPL session and set of tmp files
func SetupExecutionEnvironment() {

	var err error
	REPLSession, err = repl.NewSession()
	if err != nil {
		panic(err)
	}

	fset = token.NewFileSet()
}

// OutputMsg holds the data for a pyout message.
type OutputMsg struct {
	Execcount int                    `json:"execution_count"`
	Data      map[string]string      `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ErrMsg encodes the traceback of errors output to the notebook
type ErrMsg struct {
	EName     string   `json:"ename"`
	EValue    string   `json:"evalue"`
	Traceback []string `json:"traceback"`
}

// HandleExecuteRequest runs code from an execute_request method, and sends the various
// reply messages.
func HandleExecuteRequest(receipt MsgReceipt) {

	// Actual execution handling

	reply := NewMsg("execute_reply", receipt.Msg)
	content := make(map[string]interface{})
	reqcontent := receipt.Msg.Content.(map[string]interface{})
	code := reqcontent["code"].(string)
	silent := reqcontent["silent"].(bool)
	if !silent {
		ExecCounter++
	}
	content["execution_count"] = ExecCounter

	// the compilation/execution magic happen here
	val, err, stderr := REPLSession.Eval(code)

	if err == nil {
		content["status"] = "ok"
		content["payload"] = make([]map[string]interface{}, 0)
		content["user_variables"] = make(map[string]string)
		content["user_expressions"] = make(map[string]string)
		if (len(val) > 0 || len(REPLSession.StdoutChannel) > 0 || len(REPLSession.StderrChannel) > 0) && !silent {
			var outContent OutputMsg
			out := NewMsg("execute_result", receipt.Msg)
			outContent.Execcount = ExecCounter
			outContent.Data = make(map[string]string)

			//append output from channel messages

			var stdoutString string
			var stderrString string
			var newlineVal, newlineStderr string

			select {
			case stdout := <-REPLSession.StdoutChannel:
				stdoutString = stdout + "\n"
			default:

			}

			select {
			case stderr := <-REPLSession.StderrChannel:
				stderrString = stderr + "\n"
			default:

			}

			if len(val) > 0 {
				newlineVal = "\n"
			}
			if len(stderrString) > 0 {
				newlineStderr = "\n"
			}
			
			outContent.Data["text/plain"] = fmt.Sprint(val + newlineVal + stdoutString + newlineStderr + stderrString)
			outContent.Metadata = make(map[string]interface{})
			out.Content = outContent
			receipt.SendResponse(receipt.Sockets.IOPub_socket, out)
		}
	} else {
		content["ename"] = "ERROR"
		content["evalue"] = err.Error()
		content["traceback"] = []string{stderr.String()}
		errormsg := NewMsg("error", receipt.Msg)
		errormsg.Content = ErrMsg{"Error", err.Error(), []string{stderr.String()}}
		receipt.SendResponse(receipt.Sockets.IOPub_socket, errormsg)
	}

	// send the output back to the notebook
	reply.Content = content
	receipt.SendResponse(receipt.Sockets.Shell_socket, reply)
}
