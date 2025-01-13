import { View, Text, StyleSheet, ScrollView } from "react-native";
import React, { useState, useEffect } from "react";
import Ionicons from "react-native-vector-icons/Ionicons"; // Importa a biblioteca de ícones
import { Link } from "expo-router";
import { api } from "../api"; // Já importado

import { useFocusEffect } from "@react-navigation/native"; // Importa o hook

export default function TabTwoScreen() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Função para buscar os dados
  const fetchData = () => {
    setData([]); // Reseta os dados ao tentar fazer a requisição novamente
    setLoading(true);
    setError(null); // Reseta o erro ao tentar fazer a requisição novamente
    api
      .get("/getAccess") // Caminho relativo à URL base configurada
      .then((response) => {
        setData(response.data);
        setLoading(false);
      })
      .catch((err) => {
        console.log(err);
        setError(err);
        setLoading(false);
      });
  };

  useEffect(() => {
    // Chama a função ao carregar pela primeira vez
    fetchData();
  }, []);

  // UseFocusEffect garante que sempre que a página for carregada/focada, a requisição será feita
  useFocusEffect(
    React.useCallback(() => {
      fetchData();
    }, [])
  );

  return (
    <View style={styles.container}>
      <ScrollView style={styles.tableContainer}>
        {/* Condicional para exibir a mensagem ou a tabela */}
        {data.length === 0 && !loading && !error ? (
          <View style={styles.noDataContainer}>
            <Text style={styles.noDataText}>Nenhum registro encontrado</Text>
          </View>
        ) : (
          <View style={styles.table}>
            <View style={styles.rowHeader}>
              <Text style={styles.headerCell}>Placa</Text>
              <Text style={styles.headerCell}>Data</Text>
              <Text style={styles.headerCell}>Situação</Text>
              <Text style={styles.headerCell}>Ação</Text>
            </View>

            {loading && <Text style={styles.textPadding}>Carregando...</Text>}
            {error && (
              <Text style={styles.textPadding}>Erro ao carregar os dados</Text>
            )}

            {data.map((item: any, index) => (
              <View
                key={index}
                style={[
                  styles.row,
                  index !== data.length - 1 && styles.rowWithLine,
                ]}
              >
                <Text style={styles.cell}>{item.Plate ? item.Plate : "-"}</Text>
                <Text style={styles.cell}>{item.CreatedAt}</Text>
                <Text
                  style={[
                    styles.cell,
                    item.Status ? styles.authorized : styles.denied,
                  ]}
                >
                  {item.Status ? "Autorizado" : "Negado"}
                </Text>
                <View style={styles.cell}>
                  <Link href={"/details/" + item.ID}>
                    <Ionicons name="eye" size={20} color="white" />
                  </Link>
                </View>
              </View>
            ))}
          </View>
        )}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "black", // Fundo preto
    paddingTop: 100, // Aumenta o padding no topo para 100
    paddingHorizontal: 20, // Para as laterais
  },
  tableContainer: {
    flex: 1,
  },
  table: {
    width: "100%",
    borderWidth: 1,
    borderColor: "#555", // Cor das bordas da tabela
    borderRadius: 8,
    marginBottom: 100, // Espaço extra na parte inferior
  },
  rowHeader: {
    flexDirection: "row",
    backgroundColor: "#3498db", // Cor de fundo para o cabeçalho
    borderBottomWidth: 1,
    borderBottomColor: "#555", // Cor da linha entre o cabeçalho e o conteúdo
    paddingVertical: 12,
    paddingHorizontal: 20,
  },
  headerCell: {
    flex: 1,
    textAlign: "center",
    color: "white", // Cor do texto do cabeçalho
    fontWeight: "bold",
    fontSize: 14, // Tamanho da fonte do cabeçalho reduzido
  },
  row: {
    flexDirection: "row",
    backgroundColor: "transparent", // Fundo transparente para as linhas
    paddingVertical: 12,
    paddingHorizontal: 20,
  },
  rowWithLine: {
    borderBottomWidth: 1,
    borderBottomColor: "#555", // Cor da linha separadora
  },
  cell: {
    flex: 1,
    textAlign: "center",
    fontSize: 14, // Tamanho da fonte reduzido nas células
    color: "white", // Texto branco
    justifyContent: "center", // Centraliza verticalmente
    alignItems: "center", // Centraliza horizontalment
  },
  authorized: {
    color: "green",
  },
  denied: {
    color: "red",
  },
  textPadding: {
    paddingTop: 50,
    textAlign: "center",
    color: "white",
  },
  // Estilos para a área de "Nenhum registro encontrado"
  noDataContainer: {
    flex: 1,
    justifyContent: "center", // Alinha verticalmente no meio da tela
    alignItems: "center", // Alinha horizontalmente no centro
    paddingTop: 100, // Aumenta o espaço superior para centralizar
  },
  noDataText: {
    color: "white",
    fontSize: 16,
    textAlign: "center",
  },
});
