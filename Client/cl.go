package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

// func StartClient() {
// 	conn, err := net.Dial("tcp", "localhost:9090")
// 	if err != nil {
// 		fmt.Println("dial error:", err)
// 		return
// 	}
// 	defer conn.Close()
// 	fmt.Println("Connected to server")

// 	serverReader := bufio.NewReader(conn)
// 	stdin := bufio.NewReader(os.Stdin)

// 	for {
// 		line, err := serverReader.ReadString('\n')
// 		if err != nil {
// 			fmt.Println("Disconnected:", err)
// 			return
// 		}
// 		line = strings.TrimSpace(line)

// 		// when server asks for input, forward user input
// 		switch {
// 		case strings.HasSuffix(line, "username:") ||
// 			strings.HasSuffix(line, "password:") ||
// 			strings.HasPrefix(line, "Choose troop") ||
// 			strings.HasPrefix(line, "Choose lane"):
// 			fmt.Print(line + " ")
// 			userLine, _ := stdin.ReadString('\n')
// 			conn.Write([]byte(userLine))
// 		default:
// 			fmt.Println(line)
// 		}
// 	}
// }

func StartClient() {
	/*───────────────────────────────────
	    Connect
	───────────────────────────────────*/
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		fmt.Println("dial error:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected to server")

	serverReader := bufio.NewReader(conn) // inbound
	stdinReader := bufio.NewReader(os.Stdin)

	/*───────────────────────────────────
	    Goroutine: forward user input
	    Any line you type is sent verbatim
	───────────────────────────────────*/
	go func() {
		for {
			text, err := stdinReader.ReadString('\n')
			if err != nil {
				fmt.Println("stdin closed:", err)
				return
			}
			_, err = conn.Write([]byte(text))
			if err != nil {
				fmt.Println("send error:", err)
				return
			}
		}
	}()

	/*───────────────────────────────────
	    Main loop: print everything
	    the server sends
	───────────────────────────────────*/
	for {
		line, err := serverReader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected:", err)
			return
		}
		fmt.Print(line) // already \n-terminated
	}
}
