import { View, Text, TextInput, Button, StyleSheet, Alert } from "react-native";
import React, { useState } from "react";
import { useRouter } from "expo-router";
import { api } from "../api";

export default function NewPlate() {
  const router = useRouter();

  // Estados para armazenar os valores do formulário
  const [plate, setPlate] = useState("");

  const handleSubmit = () => {
    if (!plate.trim()) {
      Alert.alert("Erro", "O campo 'Placa' é obrigatório.");
      return;
    }

    api
      .post("/addPlate", { plate })
      .then((response) => {
        Alert.alert("Sucesso", "Placa adicionada com sucesso!");
      })
      .catch((err) => {
        console.log(err);
        Alert.alert("Erro", "Ocorreu um erro ao adicionar a placa.");
      });

    router.back();
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Adicionar Nova Placa</Text>
      <TextInput
        style={styles.input}
        placeholder="Digite a placa"
        placeholderTextColor="#aaa"
        value={plate}
        onChangeText={setPlate}
      />
      <View style={styles.buttonContainer}>
        <Button title="Salvar" onPress={handleSubmit} />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 20,
    justifyContent: "center",
  },
  title: {
    fontSize: 24,
    fontWeight: "bold",
    marginBottom: 20,
    textAlign: "center",
    color: "white",
  },
  label: {
    fontSize: 16,
    fontWeight: "bold",
    marginBottom: 5,
    color: "white",
    textAlign: "center",
  },
  input: {
    borderWidth: 1,
    borderColor: "#ccc",
    borderRadius: 5,
    padding: 10,
    marginBottom: 20,
    fontSize: 16,
    backgroundColor: "#f9f9f9",
  },
  buttonContainer: {
    marginTop: 20,
  },
  cancelButton: {
    marginTop: 10,
  },
});
