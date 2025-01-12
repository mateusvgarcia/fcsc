import cv2
import numpy as np
import uuid
from fastapi import FastAPI, File, UploadFile
from fastapi.responses import JSONResponse
import shutil
import os
import easyocr
import base64

# Inicializando a aplicação FastAPI
app = FastAPI()

# Inicializando o leitor EasyOCR
reader = easyocr.Reader(['en', 'pt'])

# Função para converter uma imagem para base64
def image_to_base64(image: np.ndarray) -> str:
    _, buffer = cv2.imencode('.jpg', image)
    return base64.b64encode(buffer).decode('utf-8')

@app.post("/process-image/")
async def process_image(file: UploadFile = File(...)):
    # Gerar UUID único para o nome do arquivo
    generated_uuid = uuid.uuid4()

    # Salvar o arquivo temporariamente
    temp_file_path = f"/tmp/{generated_uuid}.jpg"
    with open(temp_file_path, "wb") as buffer:
        shutil.copyfileobj(file.file, buffer)

    # Carregar a imagem
    image = cv2.imread(temp_file_path)
    if image is None:
        return {"error": "Erro ao carregar a imagem."}

    # Carregar o classificador Haar Cascade para placas
    plate_cascade = cv2.CascadeClassifier(cv2.data.haarcascades + "haarcascade_russian_plate_number.xml")

    # Converter para escala de cinza
    gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)

    # Aplicar thresholding (binarização)
    _, thresh = cv2.threshold(gray, 150, 255, cv2.THRESH_BINARY)

    # Detectar placas na imagem
    plates = plate_cascade.detectMultiScale(thresh, scaleFactor=1.1, minNeighbors=4, minSize=(30, 30))

    # Lista de imagens para as etapas
    steps_images = [
        ("Original", image),
        ("Escala de cinza", gray),
        ("Thresholded", thresh)
    ]

    # Criar a pasta 'result' se não existir
    result_dir = "./result"
    if not os.path.exists(result_dir):
        os.makedirs(result_dir)

    # Salvar a imagem com os retângulos da detecção (sem cortar a placa ainda)
    image_with_rectangles = image.copy()
    plate_texts = []  # Lista para armazenar os textos das placas
    plate_file_path = ""
    if len(plates) == 0:
        # Caso nenhuma placa seja detectada, adicionar o texto 'Nenhuma placa detectada'
        font = cv2.FONT_HERSHEY_SIMPLEX
        text = "Nenhuma placa detectada"
        text_size = cv2.getTextSize(text, font, 1, 2)[0]
        text_x = (image.shape[1] - text_size[0]) // 2  # Centralizar o texto
        text_y = (image.shape[0] + text_size[1]) // 2  # Centralizar o texto
        cv2.putText(image_with_rectangles, text, (text_x, text_y), font, 1, (0, 0, 255), 2, cv2.LINE_AA)
        steps_images.append(("Nenhuma Placa Detectada", image_with_rectangles))
    else:
        # Detectar placas na imagem
        for (x, y, w, h) in plates:
            # Desenhar um retângulo em torno da placa detectada
            cv2.rectangle(image_with_rectangles, (x, y), (x+w, y+h), (0, 255, 0), 2)

            # Adicionar a imagem com os retângulos no mosaico
            steps_images.append(("Contornada", image_with_rectangles))

            # Cortar a região da placa da imagem original
            plate = image[y:y+h, x:x+w]
            plate_gray = gray[y:y+h, x:x+w]  # Versão em escala de cinza da placa
            plate_thresh = thresh[y:y+h, x:x+w]  # Versão binarizada da placa

            # Melhorar a imagem para OCR aplicando uma leve dilatação ou erosão para aumentar os caracteres
            kernel = np.ones((3, 3), np.uint8)
            plate_processed = cv2.dilate(plate_thresh, kernel, iterations=1)

            uuid_img = uuid.uuid4()
            # Salvar a imagem da placa na pasta 'result'
            plate_file_path = f"{result_dir}/{uuid_img}.jpg"
            cv2.imwrite(plate_file_path, plate_processed)
            steps_images.append(("Placa Processada", plate_processed))

            # Aplicar OCR na imagem da placa para tentar extrair o texto com EasyOCR
            result = reader.readtext(plate_processed)

            plate_text = ""
            for _, text, _ in result:
                plate_text += text + " "

            print(uuid_img, plate_text.strip())
            plate_texts.append(plate_text.strip())  # Adicionar o texto detectado à lista

    # Exibir os textos das placas detectadas
    print("Texto(s) extraído(s) da placa(s):")
    for text in plate_texts:
        print(text)

    # Garantir que todas as imagens no mosaico tenham o mesmo tamanho
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
    for i in range(0, len(resized_images), 2):  # 2 imagens por linha no mosaico
        row = np.hstack([cv2.putText(resized_img, step_name, (10, 30), cv2.FONT_HERSHEY_SIMPLEX, 0.7, (0, 255, 0), 2)
                         for step_name, resized_img in resized_images[i:i + 2]])
        rows.append(row)

    # Empilhar as linhas para criar o mosaico
    max_width = max([row.shape[1] for row in rows])  # Obter a largura máxima entre as linhas
    rows_padded = [cv2.copyMakeBorder(row, 0, 0, 0, max_width - row.shape[1], cv2.BORDER_CONSTANT, value=(0, 0, 0)) for row in rows]

    # Empilhar as linhas agora com o mesmo tamanho
    mosaic = np.vstack(rows_padded)

    # Converter o mosaico para base64
    mosaic_base64 = image_to_base64(mosaic)

    # Deletar o arquivo temporário
    if os.path.exists(temp_file_path):
        os.remove(temp_file_path)

    if os.path.exists(plate_file_path):
        os.remove(plate_file_path)

        

    # Retornar o mosaico e os textos das placas
    return JSONResponse(content={
        "mosaic_base64": mosaic_base64,
        "plate_texts": plate_texts
    })

# Iniciar o servidor Uvicorn com o comando:
# uvicorn nome_do_arquivo:app --reload
