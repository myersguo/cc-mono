package rpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Client represents an RPC client to communicate with the CC-Mono server
type Client struct {
	cmd          *exec.Cmd
	stdin        chan []byte
	stdout       <-chan []byte
	errChan      <-chan error
	doneChan     chan struct{}
	isConnected  bool
}

// NewClient creates a new RPC client instance
func NewClient(ccPath string) (*Client, error) {
	client := &Client{
		stdin:     make(chan []byte, 100),
		doneChan: make(chan struct{}),
	}

	if err := client.connect(ccPath); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) connect(ccPath string) error {
	// Start the cc executable in RPC mode
	c.cmd = exec.Command(ccPath, "chat", "--mode", "rpc")
	
	stdinPipe, err := c.cmd.StdinPipe()
	if err != nil {
		return err
	}
	
	stdoutPipe, err := c.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	
	errPipe, err := c.cmd.StderrPipe()
	if err != nil {
		return err
	}
	
	if err := c.cmd.Start(); err != nil {
		return err
	}
	
	c.isConnected = true
	
	// Reader goroutine for stdout
	outChan := make(chan []byte, 100)
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) > 0 {
				outChan <- line
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading stdout: %v\n", err)
		}
		close(outChan)
		c.isConnected = false
	}()
	
	c.stdout = outChan
	
	// Reader goroutine for stderr
	errChan := make(chan error, 10)
	go func() {
		scanner := bufio.NewScanner(errPipe)
		var errors []string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				errors = append(errors, line)
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- err
		} else if len(errors) > 0 {
			errChan <- fmt.Errorf("%s", strings.Join(errors, "; "))
		}
		close(errChan)
	}()
	
	c.errChan = errChan
	
	// Writer goroutine for stdin
	go func() {
		for data := range c.stdin {
			if _, err := stdinPipe.Write(append(data, '\n')); err != nil {
				fmt.Printf("Error writing to stdin: %v\n", err)
			}
		}
	}()
	
	return nil
}

// SendCommand sends a command to the RPC server
func (c *Client) SendCommand(cmd RpcCommand) error {
	if !c.isConnected {
		return fmt.Errorf("not connected to server")
	}
	
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	
	c.stdin <- data
	return nil
}

// ReadResponse reads a response from the server (blocking)
func (c *Client) ReadResponse() ([]byte, error) {
	select {
	case data, ok := <-c.stdout:
		if !ok {
			return nil, fmt.Errorf("connection closed")
		}
		return data, nil
	case err := <-c.errChan:
		return nil, err
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout")
	}
}

// Close closes the connection and stops the server
func (c *Client) Close() error {
	if !c.isConnected {
		return nil
	}
	
	close(c.stdin)
	
	// Wait for process to terminate
	if err := c.cmd.Wait(); err != nil {
		return err
	}
	
	c.isConnected = false
	return nil
}

// Example usage
func Example() {
	fmt.Println("=== CC-Mono RPC Client Example ===")
	
	client, err := NewClient("./bin/cc")
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}
	defer client.Close()
	
	fmt.Println("Connected to RPC server")
	
	// Send test commands
	testCases := []RpcCommand{
		{
			ID:   "test-get-state",
			Type: "get_state",
		},
		{
			ID:   "test-get-models",
			Type: "get_available_models",
		},
		{
			ID:      "test-pwd",
			Type:    "bash",
			Command: "pwd",
		},
	}
	
	for _, tc := range testCases {
		fmt.Printf("=== Sending command '%s' ===\n", tc.Type)
		
		if err := client.SendCommand(tc); err != nil {
			fmt.Printf("Send error: %v\n", err)
			continue
		}
		
		for i := 0; i < 3; i++ {
			time.Sleep(100 * time.Millisecond)
			
			resp, err := client.ReadResponse()
			if err != nil {
				fmt.Printf("Read error: %v\n", err)
				continue
			}
			
			fmt.Printf("Response received: %s\n", string(resp))
		}
	}
	
	fmt.Println("\n=== Example complete ===")
}
