import { useLocalSearchParams } from "expo-router";
import { Image, Text, View, StyleSheet, ScrollView } from "react-native";
import { api, baseURL } from "../api";
import { useEffect, useState } from "react";

export default function DetailsScreen() {
  const { id } = useLocalSearchParams();

  const [data, setData] = useState() as any;

  useEffect(() => {
    api.get("/getAccess/" + id).then((response) => {
      setData(response.data);
    });
  }, []);

  return (
    <ScrollView contentContainerStyle={styles.scrollContainer}>
      <View style={styles.container}>
        {data && (
          <>
            <Image
              source={{
                uri: baseURL + "/image/" + data!.OriginalImage,
              }}
              style={styles.image}
            />
            <Image
              source={{
                uri: baseURL + "/image/" + data!.ResultImage,
              }}
              style={styles.image}
            />
          </>
        )}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  scrollContainer: {
    flexGrow: 1, // Faz com que o conteúdo ocupe toda a área possível
    justifyContent: "center", // Alinha o conteúdo verticalmente
    alignItems: "center", // Alinha o conteúdo horizontalmente
  },
  container: {
    justifyContent: "center",
    alignItems: "center",
    backgroundColor: "black",
    gap: 20,
    paddingVertical: 20, // Espaço extra no topo e na parte inferior para o scroll
  },
  image: {
    width: 400, // Largura da imagem
    height: 400, // Altura da imagem
    resizeMode: "contain", // Como a imagem deve ser redimensionada (ex: 'cover', 'contain')
  },
});
