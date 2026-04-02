package cli

import (
	"bytes"
	"ekhoes-server/module"
	"html/template"
	"log"
	"net/http"
	"strings"

	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/gorilla/websocket"
)

var thisModule module.Module

func Register() {
	thisModule = module.Module{
		Id:       "cli",
		Name:     "CLI",
		InitFunc: nil,
	}
	module.Register(thisModule)
}

//var termTmpl = template.Must(template.ParseFiles("public/terminal.htm"))

type TerminalData struct {
	Hostname         string
	Authenticated    bool
	UserName         string
	Email            string
	Token            string // solo se serve! vedi note sicurezza
	EkhoesCtlVersion string
}

func OpenTerminal(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("public/terminal.htm")
	if err != nil {
		http.Error(w, "File not found", 404)
		return
	}
	//tmpl.Execute(w, nil)

	hostname, err := os.Hostname()

	if err != nil {
		log.Println(err)
	}

	stdout, stderr, err := RunCommand("C:\\Users\\BE754RD\\OneDrive - EY\\Documents\\Source\\ekhoes-ctl-master\\ekhoes-ctl", "-v")

	if err != nil {
		log.Printf("%s\n", stderr)
	}

	version := strings.TrimRight(stdout, "\r\n")

	data := TerminalData{
		Hostname:         hostname,
		Authenticated:    false,
		UserName:         "Fabio",
		Email:            "foo@bar.com",
		Token:            "", // evita token sensibili dentro l'HTML se puoi
		EkhoesCtlVersion: version,
	}

	/*
	   data, err := os.ReadFile("public/terminal.htm")
	   if err != nil {
	       http.Error(w, "File not found", http.StatusNotFound)
	       return
	   }

	   w.Write(data)
	*/

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

}

func RunCommand(path string, args ...string) (string, string, error) {
	cmd := exec.Command(path, args...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()

	return outBuf.String(), errBuf.String(), err
}

// RunCommandStreaming esegue un comando e notifica in streaming l'output.
// - onOutput(source, data): chiamata per ogni riga di output. `source` è "STDOUT" o "STDERR".
// - onDone(state, err): chiamata quando il comando termina. `state` può essere nil se il processo non è partito.
func RunCommandStreaming(
	path string,
	args []string,
	onOutput func(source string, data string),
	onDone func(state *os.ProcessState, err error),
) (retErr error) {
	if onOutput == nil {
		onOutput = func(string, string) {}
	}
	if onDone == nil {
		onDone = func(*os.ProcessState, error) {}
	}

	cmd := exec.Command(path, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		retErr = fmt.Errorf("errore StdoutPipe: %w", err)
		onDone(nil, retErr)
		return retErr
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		retErr = fmt.Errorf("errore StderrPipe: %w", err)
		onDone(nil, retErr)
		return retErr
	}

	// Avvia il comando
	if err := cmd.Start(); err != nil {
		retErr = fmt.Errorf("errore Start: %w", err)
		onDone(nil, retErr)
		return retErr
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// canale per catturare errori dalle goroutine di lettura
	errCh := make(chan error, 2)

	// helper per leggere una stream con scanner (con buffer aumentato)
	readStream := func(r io.Reader, source string) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)

		// Aumenta il buffer massimo per linee molto lunghe (default ~64KB)
		const maxTokenSize = 1024 * 1024 // 1MB
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, maxTokenSize)

		for scanner.Scan() {
			onOutput(source, scanner.Text())
		}
		if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
			errCh <- fmt.Errorf("errore lettura %s: %w", source, err)
		}
	}

	go readStream(stdout, "STDOUT")
	go readStream(stderr, "STDERR")

	// Attendi la fine delle letture e poi del processo
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Attendi la terminazione del comando
	waitErr := cmd.Wait()

	// Raccogli eventuali errori di lettura
	var readErr error
	for e := range errCh {
		// Accumula il primo errore significativo di lettura
		if readErr == nil {
			readErr = e
		} else {
			// concatena mantenendo contesto
			readErr = fmt.Errorf("%v; %w", readErr, e)
		}
	}

	// Scegli l'errore da restituire: priorità a Wait, altrimenti errori di lettura
	if waitErr != nil {
		retErr = fmt.Errorf("errore Wait: %w", waitErr)
	} else if readErr != nil {
		retErr = readErr
	} else {
		retErr = nil
	}

	onDone(cmd.ProcessState, retErr)
	return retErr
}

func MessageHandler(
	conn *websocket.Conn,
	userId string,
	payload string,
	//onOutput func(source string, data string),
	//onDone func(state *os.ProcessState, err error),
) error {
	binFolder := "C:\\Users\\BE754RD\\OneDrive - EY\\Documents\\Source\\ekhoes-ctl-master\\"

	type CommandLine struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}

	//log.Println("Unmarshalling", payload)

	var cl CommandLine

	err := json.Unmarshal([]byte(payload), &cl)

	if err != nil {
		log.Println(err)
		return err
	}

	c := fmt.Sprintf("%s%s", binFolder, cl.Command)

	onOutput := func(source, data string) {
		// Logga o instrada dove vuoi (UI, websocket, file, ecc.)
		//fmt.Printf("[%s] %s\n", source, data)
		//encoded := base64.StdEncoding.EncodeToString([]byte(data))
		//reply.Payload = encoded
		//jsonStr, _ := json.Marshal(reply)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(data)); err != nil {
			log.Println("Error writing message:", err)
		}
	}

	onDone := func(state *os.ProcessState, err error) {

		msg := ""

		fmt.Println("state = ", state)

		if err != nil {
			fmt.Printf("Comando terminato con errore: %v\n", err)
			if state != nil {
				msg = fmt.Sprintf("{\"exitCode\": %d}", state.ExitCode())
			}
		} else {
			fmt.Println("Comando completato con successo")
			if state != nil {
				fmt.Printf("Exit code: %d\n", state.ExitCode())
				//reply.Payload = fmt.Sprintf("Exit code: %d\n", state.ExitCode())
				//reply.Payload = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("{exitCode: %d}", state.ExitCode())))
				msg = fmt.Sprintf("{\"exitCode\": %d}", state.ExitCode())
			}
		}

		//jsonStr, _ := json.Marshal(reply)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			log.Println("Error writing message:", err)
		}
	}

	err = RunCommandStreaming(c, cl.Args, onOutput, onDone)

	return err
}
