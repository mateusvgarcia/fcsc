import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Switch,
  Button,
} from "react-native";
import React, { useState, useEffect } from "react";
import { api } from "../api"; // Já importado
import { useFocusEffect } from "@react-navigation/native"; // Importa o hook
import { useRouter } from "expo-router";

export default function HomeScreen() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const router = useRouter();

  const fetchData = () => {
    setData([] as any);
    setLoading(true);
    setError(null);
    api
      .get("/getPlates")
      .then((response) => {
        setData(
          response.data.map((item: any) => ({ ...item, toggle: item.Status }))
        );
        setLoading(false);
      })
      .catch((err) => {
        console.log(err);
        setError(err);
        setLoading(false);
      });
  };

  useEffect(() => {
    fetchData();
  }, []);

  useFocusEffect(
    React.useCallback(() => {
      fetchData();
    }, [])
  );

  function HandleSwitch(id: number, value: boolean) {
    api
      .patch("/updatePlate/" + id, { status: value })
      .then((response) => {
        console.log(response.data);
      })
      .catch((err) => {
        console.log(err);
      });

    let newData = data.map((item: any) =>
      item.ID === id ? { ...item, Status: value } : item
    );

    setData(newData as any);
  }

  function handleAdd() {
    router.push("/newplate");
  }

  return (
    <View style={styles.container}>
      <Button title="Adicionar Novo Registro" onPress={handleAdd} />
      <ScrollView style={styles.tableContainer}>
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
                <Text style={styles.cell}>{item.Plate || "-"}</Text>
                <Text style={styles.cell}>{item.CreatedAt}</Text>
                <Text
                  style={[
                    styles.cell,
                    item.Status ? styles.authorized : styles.denied,
                  ]}
                >
                  {item.Status ? "Ativo" : "Inativo"}
                </Text>
                <View style={styles.switchCell}>
                  <Switch
                    value={item.Status}
                    onValueChange={() => HandleSwitch(item.ID, !item.Status)}
                    disabled={false}
                  />
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
    backgroundColor: "black",
    paddingTop: 50,
    paddingHorizontal: 20,
  },
  tableContainer: {
    flex: 1,
    marginTop: 10,
  },
  table: {
    width: "100%",
    borderWidth: 1,
    borderColor: "#555",
    borderRadius: 8,
  },
  rowHeader: {
    flexDirection: "row",
    backgroundColor: "#3498db",
    borderBottomWidth: 1,
    borderBottomColor: "#555",
    paddingVertical: 12,
    paddingHorizontal: 20,
  },
  headerCell: {
    flex: 1,
    textAlign: "center",
    color: "white",
    fontWeight: "bold",
    fontSize: 14,
  },
  row: {
    flexDirection: "row",
    backgroundColor: "transparent",
    paddingVertical: 12,
    paddingHorizontal: 20,
  },
  rowWithLine: {
    borderBottomWidth: 1,
    borderBottomColor: "#555",
  },
  cell: {
    flex: 1,
    textAlign: "center",
    fontSize: 14,
    color: "white",
  },
  switchCell: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
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
  noDataContainer: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
    paddingTop: 100,
  },
  noDataText: {
    color: "white",
    fontSize: 16,
    textAlign: "center",
  },
});
