package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"strings"
	"encoding/base64"
	"os"
	"github.com/gorilla/websocket"
	"bytes"
	"mime/multipart"
	"io"
	"io/ioutil"
	 "encoding/json"
	 "github.com/google/uuid"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Estrutura do servidor WebSocket
type WebSocketServer struct {
	clients   map[*websocket.Conn]bool  // Lista de conexões ativas
	broadcast chan Message              // Canal para mensagens (inclui remetente)
	mu        sync.Mutex                // Mutex para gerenciar concorrência
}

// Estrutura da mensagem com remetente
type Message struct {
	sender  *websocket.Conn // Conexão do remetente
	message []byte          // Conteúdo da mensagem
}

type ProcessResponse struct {
	Mosaic_base64 string `json:"mosaic_base64"`
	Plate_texts []string `json:"plate_texts"`
}

// Inicializa o servidor WebSocket
func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan Message),
	}
}

// Gerencia conexões e mensagens
func (server *WebSocketServer) handleClient(conn *websocket.Conn) {
	defer func() {
		server.mu.Lock()
		delete(server.clients, conn)
		server.mu.Unlock()
		conn.Close()
	}()

	log.Println("Novo cliente conectado.")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Conexão encerrada ou erro:", err)
			break
		}
		
		if strings.Contains(string(msg), "base64: ") {
			err := os.MkdirAll("./results", os.ModePerm)
			if err != nil {
				fmt.Println("Erro ao criar a pasta:", err)
				return
			}
		
			id := uuid.New()
			err = saveBase64Image(string(msg[8:]), "./results/original_"+id.String()+".jpg")
			if err != nil {
				fmt.Println("Erro:", err)
			} else {
				fmt.Println("Imagem salva com sucesso!")
			}


			url := "http://localhost:8001/process-image/" // URL para onde você vai enviar o POST

			data, err := sendPostRequest(url, "./results/original_"+id.String()+".jpg")
			if err != nil {
				fmt.Println("Erro:", err)
			} else {
				fmt.Println("Arquivo enviado com sucesso!")
			}

			err = saveBase64Image(data.Mosaic_base64, "./results/result_"+id.String()+".jpg")
			if err != nil {
				fmt.Println("Erro:", err)
			} else {
				fmt.Println("Imagem salva com sucesso!")
			}

			fmt.Println("Placas detectadas:", data.Plate_texts)

			continue
		}
		
		log.Printf("Mensagem recebida de %v: %s\n", conn.RemoteAddr(), msg)



		// Envia a mensagem para o canal de broadcast
		server.broadcast <- Message{sender: conn, message: msg}
	}
}

// Inicia o servidor WebSocket
func (server *WebSocketServer) run() {
	for {
		// Lê mensagens do canal de broadcast
		msg := <-server.broadcast

		server.mu.Lock()
		for client := range server.clients {
			// Evita enviar a mensagem para o mandante
			if client != msg.sender {
				err := client.WriteMessage(websocket.TextMessage, msg.message)
				if err != nil {
					log.Println("Erro ao enviar mensagem para o cliente:", err)
					client.Close()
					delete(server.clients, client)
				}
			}
		}
		server.mu.Unlock()
	}
}

func main() {
	fmt.Println("Iniciando servidor WebSocket...")

	server := NewWebSocketServer()

	// Goroutine para gerenciar o broadcast
	go server.run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Erro ao aceitar conexão:", err)
			return
		}

		server.mu.Lock()
		server.clients[conn] = true
		server.mu.Unlock()

		// Goroutine para lidar com o cliente
		go server.handleClient(conn)
	})

	// Obter o IP local
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal("Erro ao obter o IP:", err)
	}

	var localIP string
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			localIP = ipNet.IP.String()
			break
		}
	}

	if localIP == "" {
		localIP = "localhost"
	}

	fmt.Printf("Servidor WebSocket está rodando em: ws://%s:8000\n", localIP)

	// Iniciar o servidor HTTP
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal("Erro ao iniciar o servidor:", err)
	}
}

func saveBase64Image(base64String, filePath string) error {
	// Decodifica a string Base64
	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return fmt.Errorf("erro ao decodificar base64: %v", err)
	}

	// Cria um arquivo para salvar a imagem
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo: %v", err)
	}
	defer file.Close()

	// Escreve os bytes da imagem no arquivo
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("erro ao escrever no arquivo: %v", err)
	}

	return nil
}

func sendPostRequest(url string, filePath string) (ProcessResponse, error) {
	// Abre o arquivo
	file, err := os.Open(filePath)
	if err != nil {
		return ProcessResponse{}, fmt.Errorf("erro ao abrir o arquivo: %v", err)
	}
	defer file.Close()

	// Cria um buffer para armazenar os dados da requisição
	body := &bytes.Buffer{}
	// Cria o escritor multipart que adicionará os campos e o arquivo
	writer := multipart.NewWriter(body)

	// Adiciona o arquivo ao corpo da requisição
	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return ProcessResponse{}, fmt.Errorf("erro ao criar o campo do arquivo: %v", err)
	}
	// Copia o conteúdo do arquivo para o campo de arquivo
	_, err = io.Copy(part, file)
	if err != nil {
		return ProcessResponse{}, fmt.Errorf("erro ao copiar o arquivo: %v", err)
	}

	// Adiciona outros campos, se necessário
	err = writer.WriteField("field1", "value1")
	if err != nil {
		return  ProcessResponse{}, fmt.Errorf("erro ao adicionar campo: %v", err)
	}

	// Finaliza o corpo da requisição
	err = writer.Close()
	if err != nil {
		return  ProcessResponse{}, fmt.Errorf("erro ao fechar o writer: %v", err)
	}

	// Cria a requisição HTTP
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return  ProcessResponse{}, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	// Define o cabeçalho Content-Type com o tipo correto
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Envia a requisição
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return  ProcessResponse{}, fmt.Errorf("erro ao enviar requisição: %v", err)
	}
	defer resp.Body.Close()

	// Lê o corpo da resposta
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return  ProcessResponse{}, fmt.Errorf("erro ao ler o corpo da resposta: %v", err)
	}

	// // Exibe a resposta e o conteúdo do corpo
	// fmt.Println("Status:", resp.Status)
	// fmt.Println("Corpo da resposta:", string(respBody))

	var processResponse ProcessResponse
	err = json.Unmarshal(respBody, &processResponse)
	if err != nil {
		return  ProcessResponse{}, fmt.Errorf("erro ao decodificar JSON: %v", err)
	}

	return processResponse, nil
}