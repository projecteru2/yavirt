package models

type LambdaOptions struct {
	Cmd       []string `json:"cmd,omitempty"`
	CmdOutput []byte   `json:"cmd_output,omitempty"`
	ExitCode  int      `json:"exit_code,omitempty"`
	Pid       int      `json:"pid,omitempty"`
}
