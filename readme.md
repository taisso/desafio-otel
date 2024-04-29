### Iniciar

```
docker compose up --build
```

### Portas
- service_a: 8081
- service_b: 8080
- zikpin: 9411

### Endpoint
```
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"cep":"18085090"}' \
  http://localhost:8081
```
>> Também tem o arquivo request.http caso queira

### Para visualizar no Zipkin

Para visualizar no zipkin busque no campo de pesquisa por algum dos serviços
que são: "provider-service-a" ou "provider-service-b"
![alt text](https://i.ibb.co/SVK8xDN/2024-04-29-13-02.png)
![alt text](https://i.ibb.co/7XJVzXp/2024-04-29-13-09.png)