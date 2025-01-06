import cv2
import numpy as np
import uuid

# Carregar a imagem
image = cv2.imread("placa.jpg")
generated_uuid = uuid.uuid4()

# Verificar se a imagem foi carregada corretamente
if image is None:
    print("Erro ao carregar a imagem.")
    exit()

# Carregar o classificador Haar Cascade para placas
plate_cascade = cv2.CascadeClassifier(cv2.data.haarcascades + "haarcascade_russian_plate_number.xml")

# Converter para escala de cinza
gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)

# Detectar placas na imagem
plates = plate_cascade.detectMultiScale(gray, scaleFactor=1.1, minNeighbors=4, minSize=(30, 30))

# Lista de imagens para as etapas
steps_images = [
    ("Original", image),
    ("Gray", gray)
]

# Salvar a imagem com os retângulos da detecção (sem cortar a placa ainda)
image_with_rectangles = image.copy()
for (x, y, w, h) in plates:
    # Desenhar um retângulo em torno da placa detectada
    cv2.rectangle(image_with_rectangles, (x, y), (x+w, y+h), (0, 255, 0), 2)

    # Adicionar a imagem com os retângulos no mosaico
    steps_images.append(("With Rectangles", image_with_rectangles))

    # Cortar a região da placa da imagem original
    plate = image[y:y+h, x:x+w]

    # Salvar a imagem da placa
    cv2.imwrite("placa_detectada.jpg", plate)
    steps_images.append(("Plate", plate))



# Garantir que todas as imagens no mosaico tenham o mesmo tamanho
# Redimensionar todas as imagens para o tamanho da imagem original
resize_dim = (image.shape[1], image.shape[0])

# Redimensionar as imagens
resized_images = []
for step_name, step_img in steps_images:
    # Se a imagem for em escala de cinza, convertê-la para 3 canais (BGR)
    if len(step_img.shape) == 2:  # Imagem em escala de cinza
        step_img = cv2.cvtColor(step_img, cv2.COLOR_GRAY2BGR)
    
    # Redimensionar a imagem para o mesmo tamanho da imagem original
    resized_img = cv2.resize(step_img, resize_dim)
    resized_images.append((step_name, resized_img))

# Criar o mosaico com as imagens e textos
rows = []
for i in range(0, len(resized_images), 2):  # 3 imagens por linha no mosaico
    row = np.hstack([cv2.putText(resized_img, step_name, (10, 30), cv2.FONT_HERSHEY_SIMPLEX, 0.7, (0, 255, 0), 2)
                     for step_name, resized_img in resized_images[i:i + 2]])
    rows.append(row)

# Empilhar as linhas para criar o mosaico
# Ajuste de padding para garantir que todas as linhas tenham a mesma largura
max_width = max([row.shape[1] for row in rows])  # Obter a largura máxima entre as linhas
rows_padded = [cv2.copyMakeBorder(row, 0, 0, 0, max_width - row.shape[1], cv2.BORDER_CONSTANT, value=(0, 0, 0)) for row in rows]

# Empilhar as linhas agora com o mesmo tamanho
mosaic = np.vstack(rows_padded)

# Salvar e exibir o mosaico final
filename = f"etapas_completas_{generated_uuid}.jpg"
print(filename)
cv2.imwrite("./results/"+filename, mosaic)
cv2.imshow("Etapas Completas", mosaic)

# Espera até que o usuário pressione uma tecla
cv2.waitKey(0)
cv2.destroyAllWindows()
