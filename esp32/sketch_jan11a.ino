#include <WiFi.h>
#include <ArduinoWebsockets.h>
#include "esp_camera.h"
#include "base64.h"

// Definir o modelo da câmera
#define CAMERA_MODEL_AI_THINKER
#include "camera_pins.h"

// Definir o pino do flash (alterar para o pino correto se necessário)
#define FLASH_PIN 4  // Exemplo: pino 4 para o flash

// Variável para controlar o estado do flash
bool flashState = false;

using namespace websockets;

// Configurações da rede Wi-Fi
const char* ssid = "";         // Substitua pelo seu SSID
const char* password = "";   // Substitua pela sua senha

// URL do servidor WebSocket
const char* ws_server = "ws://0.tcp.sa.ngrok.io:16473";  // Substitua pelo IP/URL e porta do servidor WS

WebsocketsClient wsClient;
bool isConnected = false;

// Função para capturar foto e retornar como Base64
String capturePhoto() {
  camera_fb_t *fb = esp_camera_fb_get(); // Captura o frame da câmera

  if (!fb) {
    Serial.println("Falha ao capturar o frame da câmera!");
    return "";
  }

  // Verificar se o tamanho do buffer é válido
  if (fb->len == 0) {
    Serial.println("Frame capturado está vazio!");
    return "";
  }

  // Converte para Base64
  String imageBase64 = base64::encode(fb->buf, fb->len);
  
  // Verificar se a conversão foi bem-sucedida
  if (imageBase64.length() == 0) {
    Serial.println("Falha ao codificar a imagem em Base64!");
    esp_camera_fb_return(fb);
    return "";
  }

  esp_camera_fb_return(fb);
  return imageBase64;
}

// Função para conectar no Wi-Fi
void connectToWiFi() {
  Serial.print("Conectando ao Wi-Fi");
  WiFi.begin(ssid, password);

  while (WiFi.status() != WL_CONNECTED) {
    delay(1000);
    Serial.print(".");
  }

  Serial.println("\nWi-Fi conectado!");
  Serial.print("IP: ");
  Serial.println(WiFi.localIP());
}

// Função para conectar ao servidor WebSocket
void connectToWebSocket() {
  Serial.println("Conectando ao servidor WebSocket...");
  if (wsClient.connect(ws_server)) {
    Serial.println("Conexão WebSocket estabelecida!");
    isConnected = true;
  } else {
    Serial.println("Falha ao conectar ao servidor WebSocket!");
    isConnected = false;
  }
}

// Função para enviar mensagens
void sendMessage() {
  if (isConnected) {
    wsClient.send("Olá, servidor!");  // Mensagem que será enviada
    Serial.println("Mensagem enviada: Olá, servidor!");
  }
}

// Callback para lidar com mensagens recebidas
void onMessage(WebsocketsMessage message) {
  Serial.print("Mensagem recebida: ");
  Serial.println(message.data());

  if (message.data() == "flash"){
    flashState = !flashState;
    digitalWrite(FLASH_PIN, flashState ? HIGH : LOW);    
    String state = flashState ? "flash ligado" : "flash desligado";
    String messageToSend = "flash: " + state; // Adiciona o prefixo
    wsClient.send(state);
    
  }

  if (message.data() == "foto"){
    String photoBase64 = capturePhoto();    
    String messageToSend = "base64: " + photoBase64; // Adiciona o prefixo
    wsClient.send(messageToSend); // Envia o frame ao cliente
  }

  if (message.data() == "start_stream") {
    while (true) {
      if (!isConnected) break; // Interrompe se a conexão for perdida

      String photoBase64 = capturePhoto();
      if (photoBase64.length() > 0) {
        wsClient.send(photoBase64); // Envia o frame ao cliente
      }

      delay(100); // Intervalo entre frames (10 FPS)
    }
  }

}

// Callback para lidar com eventos (ex.: desconexão)
void onEvent(WebsocketsEvent event, String data) {
  if (event == WebsocketsEvent::ConnectionOpened) {
    Serial.println("Conexão WebSocket aberta!");
    isConnected = true;
  } else if (event == WebsocketsEvent::ConnectionClosed) {
    Serial.println("Conexão WebSocket fechada!");
    isConnected = false;
  } else if (event == WebsocketsEvent::GotPing) {
    Serial.println("Ping recebido!");
  } else if (event == WebsocketsEvent::GotPong) {
    Serial.println("Pong recebido!");
  }
}

void setup() {
  // Inicializa o monitor serial
  Serial.begin(115200);

  Serial.println("Inicializando...");

  // Configurar o pino do flash como saída
  pinMode(FLASH_PIN, OUTPUT);
  digitalWrite(FLASH_PIN, LOW); // Garante que o flash comece desligado
  
  // Configuração da câmera
  camera_config_t config;
  config.ledc_channel = LEDC_CHANNEL_0;
  config.ledc_timer = LEDC_TIMER_0;
  config.pin_d0 = Y2_GPIO_NUM;
  config.pin_d1 = Y3_GPIO_NUM;
  config.pin_d2 = Y4_GPIO_NUM;
  config.pin_d3 = Y5_GPIO_NUM;
  config.pin_d4 = Y6_GPIO_NUM;
  config.pin_d5 = Y7_GPIO_NUM;
  config.pin_d6 = Y8_GPIO_NUM;
  config.pin_d7 = Y9_GPIO_NUM;
  config.pin_xclk = XCLK_GPIO_NUM;
  config.pin_pclk = PCLK_GPIO_NUM;
  config.pin_vsync = VSYNC_GPIO_NUM;
  config.pin_href = HREF_GPIO_NUM;
  config.pin_sccb_sda = SIOD_GPIO_NUM;
  config.pin_sccb_scl = SIOC_GPIO_NUM;
  config.pin_pwdn = PWDN_GPIO_NUM;
  config.pin_reset = RESET_GPIO_NUM;
  config.xclk_freq_hz = 20000000;
  config.pixel_format = PIXFORMAT_JPEG;
  config.frame_size = FRAMESIZE_XGA; // Resolução QVGA para estabilidade
  config.jpeg_quality = 5;          // Qualidade JPEG (menor número = melhor qualidade)
  config.fb_count = 1;

  // Inicializar a câmera
  esp_err_t err = esp_camera_init(&config);
  if (err != ESP_OK) {
    Serial.printf("Erro ao inicializar a câmera: 0x%x\n", err);
    while (true); // Pausa se a inicialização falhar
  }

  // Configurar o flip e espelhamento
  sensor_t *s = esp_camera_sensor_get();
  s->set_vflip(s, 1);  // Ativa o flip vertical
  s->set_hmirror(s, 1); // Ativa o espelhamento horizontal

  Serial.println("Câmera inicializada!");
  
  // Conecta ao Wi-Fi
  connectToWiFi();
  
  // Configura o cliente WebSocket
  wsClient.onMessage(onMessage);
  wsClient.onEvent(onEvent);

  // Tenta conectar ao servidor WebSocket
  connectToWebSocket();
}


const long interval = 100; // Enviar imagens a cada 100ms (~10 FPS)

void loop() {

  // Mantém a conexão WebSocket
  wsClient.poll();
  
  // Reconexão caso desconecte
  if (!isConnected) {
    static unsigned long lastReconnectAttempt = 0;
    if (millis() - lastReconnectAttempt >= 5000) {
      Serial.println("Tentando reconectar ao servidor WebSocket...");
      connectToWebSocket();
      lastReconnectAttempt = millis();
    }
  }
}
