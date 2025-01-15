package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tarm/serial"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var database *gorm.DB

func init() {
	db, err := gorm.Open(sqlite.Open("database.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("erro ao conectar com o banco de dados: ", err)
	}

	database = db
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Estrutura do servidor WebSocket
type WebSocketServer struct {
	clients   map[*websocket.Conn]bool // Lista de conexões ativas
	broadcast chan Message             // Canal para mensagens (inclui remetente)
	mu        sync.Mutex               // Mutex para gerenciar concorrência
}

// Estrutura da mensagem com remetente
type Message struct {
	sender  *websocket.Conn // Conexão do remetente
	message []byte          // Conteúdo da mensagem
}

type ProcessResponse struct {
	Mosaic_base64 string   `json:"mosaic_base64"`
	Plate_texts   []string `json:"plate_texts"`
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

		if strings.Contains(string(msg), "base64:") {
			err := os.MkdirAll("./results", os.ModePerm)
			if err != nil {
				fmt.Println("Erro ao criar a pasta:", err)
				return
			}

			id := uuid.New()

			// save to txt file
			f, err := os.Create("./results/base64_" + id.String() + ".txt")
			if err != nil {
				fmt.Println("Erro ao criar o arquivo:", err)
				return
			}

			_, err = f.WriteString(string(msg))
			if err != nil {
				fmt.Println("Erro ao escrever no arquivo:", err)
				return
			}

			err = f.Close()

			if err != nil {
				fmt.Println("Erro ao fechar o arquivo:", err)
				return
			}

			err = saveBase64Image(string(msg[7:]), "./results/original_"+id.String()+".jpg")
			if err != nil {
				fmt.Println("Erro:", err)
			} else {
				fmt.Println("Imagem salva com sucesso!")
			}

			dataDatabase := AccessLog{
				CreatedAt:     time.Now().Format("02/01/2006 15:04:05"),
				OriginalImage: "original_" + id.String() + ".jpg",
			}

			result := database.Create(&dataDatabase)
			if result.Error != nil {
				fmt.Println("Erro ao salvar no banco de dados:", result.Error)
			}

			url := "http://localhost:8001/process-image/" // URL para onde você vai enviar o POST

			data, err := sendPostRequest(url, "./results/original_"+id.String()+".jpg")
			if err != nil {
				fmt.Println("Erro:", err)
				continue
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
			for _, plate := range data.Plate_texts {
				var allowedPlate AllowedPlates
				result := database.Where("plate = ? and status = ?", plate, true).First(&allowedPlate)
				if result.Error != nil {
					fmt.Println("Erro ao buscar placa no banco de dados:", result.Error)
					continue
				}

				if allowedPlate.Status {
					dataDatabase.Status = true
					fmt.Println("Placa permitida:", plate)
					break
				}

				dataDatabase.Status = false
			}

			dataDatabase.ResultImage = "result_" + id.String() + ".jpg"
			dataDatabase.Plate = strings.Join(data.Plate_texts, ", ")
			result = database.Save(&dataDatabase)
			if result.Error != nil {
				fmt.Println("Erro ao salvar no banco de dados:", result.Error)
			}

			// server.broadcast <- Message{sender: conn, message: msg}

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

type AllowedPlates struct {
	ID        uint   `gorm:"primaryKey"`
	CreatedAt string `gorm:"autoCreateTime"`
	Plate     string `gorm:"unique"`
	Status    bool
}

type AccessLog struct {
	ID            uint   `gorm:"primaryKey"`
	CreatedAt     string `gorm:"autoCreateTime"`
	Plate         string
	OriginalImage string
	ResultImage   string
	Status        bool `gorm:"default:false"`
}

func main() {
	err := database.AutoMigrate(&AllowedPlates{}, &AccessLog{})
	if err != nil {
		log.Fatal("erro ao migrar o modelo: ", err)
	}

	fmt.Println("Iniciando servidor WebSocket...")

	server := NewWebSocketServer()

	// Goroutine para gerenciar o broadcast
	go server.run()

	// Iniciar o servidor WebSocket
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			upgrader.CheckOrigin = func(r *http.Request) bool {
				return true
			}
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

		localIP, err := getLocalIP()
		if err != nil {
			log.Fatalf("Erro ao obter o IP local: %v", err)
		}

		fmt.Println("IP local:", localIP)

		// Iniciar o servidor WebSocket na porta 8000
		fmt.Println("Servidor WebSocket está rodando em: ws://localhost:8000/ws")
		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Fatal("Erro ao iniciar o servidor WebSocket:", err)
		}
	}()

	// Usando Gin para configurar as rotas da API
	router := gin.Default()

	// API RESTful
	router.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "Servidor em funcionamento",
		})
	})

	router.GET("/getAccess", func(c *gin.Context) {
		var accessLogs []AccessLog
		result := database.Find(&accessLogs).Order("id asc")
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": result.Error.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, accessLogs)
	})

	router.GET("/getAccess/:id", func(c *gin.Context) {
		id := c.Param("id")

		var accessLog AccessLog
		result := database.First(&accessLog, id)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": result.Error.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, accessLog)
	})

	router.GET("/image/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		imagePath := filepath.Join("./results", filename)
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Imagem não encontrada"})
			return
		}
		c.File(imagePath)
	})

	router.POST("/addPlate", func(c *gin.Context) {
		var plate AllowedPlates
		err := c.BindJSON(&plate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		plate.CreatedAt = time.Now().Format("02/01/2006 15:04:05")
		plate.Status = true

		result := database.Create(&plate)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": result.Error.Error(),
			})

			return
		}

		c.JSON(http.StatusCreated, plate)
	})

	router.GET("/getPlates", func(c *gin.Context) {
		var plates []AllowedPlates
		result := database.Find(&plates).Order("id desc")
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": result.Error.Error(),
			})

			return
		}

		c.JSON(http.StatusOK, plates)
	})

	router.PATCH("/updatePlate/:id", func(c *gin.Context) {
		id := c.Param("id")

		var plate AllowedPlates
		result := database.First(&plate, id)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": result.Error.Error(),
			})

			return
		}

		var newPlate AllowedPlates
		err := c.BindJSON(&newPlate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

			return
		}

		plate.Status = newPlate.Status

		result = database.Save(&plate)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": result.Error.Error(),
			})

			return
		}

		c.JSON(http.StatusOK, plate)
	})

	// Iniciar servidor Gin na porta 8080
	go func() {
		fmt.Println("Servidor Gin está rodando em: http://localhost:8080")
		if err := router.Run(":8080"); err != nil {
			log.Fatal("Erro ao iniciar o servidor Gin:", err)
		}
	}()

	// Manter o servidor principal ativo
	select {}
}

func saveBase64Image(base64String, filePath string) error {
	// Decodifica a string Base64
	// fmt.Println(base64String)
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

	// Cria o corpo da requisição multipart
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return ProcessResponse{}, fmt.Errorf("erro ao criar campo de arquivo: %v", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return ProcessResponse{}, fmt.Errorf("erro ao copiar conteúdo do arquivo: %v", err)
	}
	writer.Close()

	// Envia a requisição POST
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return ProcessResponse{}, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	// Define o tipo do conteúdo da requisição
	request.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return ProcessResponse{}, fmt.Errorf("erro ao enviar requisição: %v", err)
	}
	defer response.Body.Close()

	// Lê a resposta
	respBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ProcessResponse{}, fmt.Errorf("erro ao ler resposta: %v", err)
	}

	// Parse a resposta JSON
	var data ProcessResponse
	err = json.Unmarshal(respBody, &data)
	if err != nil {
		return ProcessResponse{}, fmt.Errorf("erro ao fazer unmarshal: %v", err)
	}

	return data, nil
}

func getLocalIP() (string, error) {
	// Conecta a um endereço remoto para descobrir o IP local
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Obtém o endereço local associado à conexão
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func sendSerial(message string) {
	config := &serial.Config{
		Name: "/dev/ttyACM0",
		Baud: 9600,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("Erro ao abrir a porta serial: %v", err)
	}
	defer port.Close()

	for _, char := range message {
		_, err = port.Write([]byte{byte(char)})
		if err != nil {
			log.Fatalf("Erro ao enviar o caractere '%c': %v", char, err)
		}
	}

	fmt.Println("Mensagem enviada para o Arduino:", message)
}
