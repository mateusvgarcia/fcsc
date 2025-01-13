// api.js
import axios from "axios";

const baseURL = "http://10.1.11.191:8080";

// Criando uma instância do Axios com a URL base configurada
const api = axios.create({
  baseURL, // URL base da sua API
});

export { api, baseURL };
